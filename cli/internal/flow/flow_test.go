// Tests for the fabrik.sh/loop.sh Go port (issue cli-6gq.2).
package flow

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

func noopSleep(time.Duration) {}

func mkRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".beads"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".fabrik"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".fabrik", "config.yaml"), []byte("x: 1\n"), 0o644))
	return repo
}

func emptyBdOutputs() map[string]string {
	return map[string]string{
		"bd list --status=open --json":               `[]`,
		"bd list --status=closed --json --limit 100": `[]`,
		"bd ready --json":                            `[]`,
	}
}

func countCalls(f *exec.Fake, prefix string) int {
	n := 0
	for _, c := range f.Calls {
		if c == prefix || strings.HasPrefix(c, prefix+" ") {
			n++
		}
	}
	return n
}

func TestRunSetupGateFails(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"bd": true}}
	err := Run(context.Background(), RunConfig{RepoPath: t.TempDir()}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setup incomplete")
	assert.Empty(t, f.Calls)
}

func TestRunAutoRequiresPrompt(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"bd": true}}
	err := Run(context.Background(), RunConfig{RepoPath: mkRepo(t), Auto: true}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--prompt")
}

func TestRunAutoPausesAfterPlanForApprove(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	prd := prdPromptPrefix + "add login"
	outputs := emptyBdOutputs()
	outputs["opencode run "+prd] = "prd-ok"
	outputs["opencode run "+planPrompt] = "plan-ok"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Run(context.Background(), RunConfig{RepoPath: mkRepo(t), Auto: true, Prompt: "add login"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Starting Fabrik workflow...")
	assert.Contains(t, out, "FABRIK plan")
	assert.Contains(t, out, "APPROVE", "plan must pause for APPROVE before the build wave")
	assert.Equal(t, 1, countCalls(f, "opencode run "+prd))
	assert.Equal(t, 1, countCalls(f, "opencode run "+planPrompt))
	assert.Equal(t, 0, countCalls(f, "opencode run "+tmpl.MustRead("hub/PROMPT_build.md")), "build must not auto-run before APPROVE")
}

func TestRunInteractiveLaunchesTUI(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	outputs := emptyBdOutputs()
	outputs["opencode"] = "tui"
	outputs["opencode run "+planPrompt] = "plan-ok"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Run(context.Background(), RunConfig{RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	require.NotEmpty(t, f.Calls)
	assert.Equal(t, "opencode", f.Calls[0])
	assert.Contains(t, buf.String(), "Step 1:")
}

func TestLoopBadMode(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"bd": true}}
	err := Loop(context.Background(), LoopConfig{Mode: "bogus"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan|build")
}

func TestLoopBdMissing(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{}
	err := Loop(context.Background(), LoopConfig{Mode: "plan"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bd not found")
}

func TestLoopPlanRunsOnce(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	outputs := emptyBdOutputs()
	outputs["opencode run "+planPrompt] = "plan-ok"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Loop(context.Background(), LoopConfig{Mode: "plan"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Equal(t, 1, countCalls(f, "opencode run"))
	assert.Contains(t, buf.String(), "FABRIK plan (iteration 1)")
	assert.Contains(t, buf.String(), "Planning complete")
	assert.False(t, f.Called("git push"), "push must not run without explicit opt-in")
}

func TestLoopPlanPushesWhenOptIn(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	outputs := emptyBdOutputs()
	outputs["opencode run "+planPrompt] = "plan-ok"
	outputs["git branch --show-current"] = "main"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.True(t, f.Called("git push"), "push must run with explicit AutoPush opt-in")
}

func TestLoopBuildNoOpenBreaks(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: emptyBdOutputs()}
	err := Loop(context.Background(), LoopConfig{Mode: "build"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No open Beads issues.")
	assert.Equal(t, 0, countCalls(f, "opencode run"))
}

func TestLoopBuildNoneReadyBreaks(t *testing.T) {
	var buf bytes.Buffer
	outputs := emptyBdOutputs()
	outputs["bd list --status=open --json"] = `[{"id":"bd-1","title":"stuck"}]`
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Loop(context.Background(), LoopConfig{Mode: "build"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "none are ready")
	assert.Equal(t, 0, countCalls(f, "opencode run"))
}

func TestLoopBuildMaxCapsIterations(t *testing.T) {
	var buf bytes.Buffer
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	outputs := emptyBdOutputs()
	outputs["bd list --status=open --json"] = `[{"id":"bd-1","title":"Do thing"}]`
	outputs["bd ready --json"] = `[{"id":"bd-1","title":"Do thing"}]`
	outputs["opencode run "+buildPrompt] = "build-ok"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Loop(context.Background(), LoopConfig{Mode: "build", Max: 2}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Equal(t, 2, countCalls(f, "opencode run"))
	out := buf.String()
	assert.Contains(t, out, "FABRIK build (iteration 2)")
	assert.Contains(t, out, "Target: bd-1 — Do thing")
	assert.Contains(t, out, "Reached max iterations: 2")
}

func TestLoopRunsInRepoDir(t *testing.T) {
	var buf bytes.Buffer
	repo := mkRepo(t)
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	outputs := emptyBdOutputs()
	outputs["opencode run "+planPrompt] = "plan-ok"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Loop(context.Background(), LoopConfig{RepoPath: repo, Mode: "plan"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	require.NotEmpty(t, f.Dirs, "all commands must run in the repo dir")
	for i, dir := range f.Dirs {
		assert.Equal(t, repo, dir, "call %q ran outside the repo", f.Calls[i])
	}
}

func TestLoopBuildWrappedJSON(t *testing.T) {
	var buf bytes.Buffer
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	outputs := emptyBdOutputs()
	outputs["bd list --status=open --json"] = `{"issues":[{"id":"bd-9","title":"Wrapped"}]}`
	outputs["bd ready --json"] = `{"issues":[{"id":"bd-9","title":"Wrapped"}]}`
	outputs["opencode run "+buildPrompt] = "build-ok"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Loop(context.Background(), LoopConfig{Mode: "build", Max: 1}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Equal(t, 1, countCalls(f, "opencode run"))
	assert.Contains(t, buf.String(), "Target: bd-9 — Wrapped")
}

func TestLoopOpencodeFailurePropagates(t *testing.T) {
	var buf bytes.Buffer
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	outputs := emptyBdOutputs()
	outputs["bd list --status=open --json"] = `[{"id":"bd-1","title":"Do thing"}]`
	outputs["bd ready --json"] = `[{"id":"bd-1","title":"Do thing"}]`
	f := &exec.Fake{
		Path:    map[string]bool{"bd": true},
		Outputs: outputs,
		Errors:  map[string]error{"opencode run " + buildPrompt: assert.AnError},
	}
	err := Loop(context.Background(), LoopConfig{Mode: "build", Max: 1}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opencode run build")
}

func TestLoopBdErrorPropagates(t *testing.T) {
	var buf bytes.Buffer
	outputs := emptyBdOutputs()
	outputs["bd list --status=open --json"] = `[{"id":"bd-1","title":"Do thing"}]`
	delete(outputs, "bd ready --json")
	f := &exec.Fake{
		Path:    map[string]bool{"bd": true},
		Outputs: outputs,
		Errors:  map[string]error{"bd ready --json": assert.AnError},
	}
	err := Loop(context.Background(), LoopConfig{Mode: "build"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err, "bd failures must surface, not read as empty")
	assert.Contains(t, err.Error(), "bd ready --json")
}

func TestLoopPushUsesOriginDashDash(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	outputs := emptyBdOutputs()
	outputs["opencode run "+planPrompt] = "plan-ok"
	outputs["git branch --show-current"] = "feature/cool-thing"
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
	err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.True(t, f.Called("git push origin -- feature/cool-thing"), "push must use origin -- <branch>: %v", f.Calls)
}

func TestLoopPushSanitizesBranch(t *testing.T) {
	for _, evil := range []string{"-f", "evil; rm -rf", "a..b", "x y", ".hidden", ""} {
		var buf bytes.Buffer
		planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
		outputs := emptyBdOutputs()
		outputs["opencode run "+planPrompt] = "plan-ok"
		outputs["git branch --show-current"] = evil
		f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: outputs}
		err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
		require.NoError(t, err, "branch %q", evil)
		assert.False(t, f.Called("git push"), "unsafe/unknown branch %q must skip push: %v", evil, f.Calls)
		assert.Contains(t, buf.String(), "unknown branch", "branch %q must report skipped push", evil)
	}
}

func TestLoopPushSkippedWhenBranchUnknown(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	outputs := emptyBdOutputs()
	outputs["opencode run "+planPrompt] = "plan-ok"
	f := &exec.Fake{
		Path:    map[string]bool{"bd": true},
		Outputs: outputs,
		Errors:  map[string]error{"git branch --show-current": assert.AnError},
	}
	err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.False(t, f.Called("git push"), "unknown branch (error) must skip push: %v", f.Calls)
	assert.Contains(t, buf.String(), "unknown branch")
}

func TestCurrentBranchReturnsEmptyOnUnknown(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	f := &exec.Fake{
		Path:    map[string]bool{},
		Outputs: map[string]string{},
		Errors:  map[string]error{"git branch --show-current": assert.AnError},
	}
	assert.Empty(t, currentBranch(ctx, f, repo), "error must yield empty, not main fallback")
	f2 := &exec.Fake{Path: map[string]bool{}, Outputs: map[string]string{"git branch --show-current": ""}}
	assert.Empty(t, currentBranch(ctx, f2, repo), "empty output must yield empty")
	f3 := &exec.Fake{Path: map[string]bool{}, Outputs: map[string]string{"git branch --show-current": "-f"}}
	assert.Empty(t, currentBranch(ctx, f3, repo), "invalid branch must yield empty")
	f4 := &exec.Fake{Path: map[string]bool{}, Outputs: map[string]string{"git branch --show-current": "feature/ok"}}
	assert.Equal(t, "feature/ok", currentBranch(ctx, f4, repo))
}
