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
	"github.com/ur-wesley/fabrik/cli/internal/store"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

func noopSleep(time.Duration) {}

type fakeLoopStore struct {
	unfinished int
	done       int
	ready      []LoopIssue
	claimBy    map[string]LoopIssue
	claims     []string
}

func (f *fakeLoopStore) Snapshot(ctx context.Context) (int, int, []LoopIssue, error) {
	return f.unfinished, f.done, f.ready, nil
}

func (f *fakeLoopStore) ClaimNext(ctx context.Context, by string) (LoopIssue, error) {
	if len(f.ready) == 0 {
		return LoopIssue{}, nil
	}
	id := f.ready[0].ID
	f.claims = append(f.claims, id)
	if f.claimBy != nil {
		if iss, ok := f.claimBy[id]; ok {
			return iss, nil
		}
	}
	return f.ready[0], nil
}

func (f *fakeLoopStore) Close() error { return nil }

func withFakeStore(t *testing.T, fake LoopStore) {
	t.Helper()
	old := OpenLoopStore
	OpenLoopStore = func(ctx context.Context, repo string) (LoopStore, error) {
		if _, ok := store.RepoRoot(repo); !ok {
			return nil, store.ErrNoStore
		}
		return fake, nil
	}
	t.Cleanup(func() { OpenLoopStore = old })
}

func mkRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".fabrik", "dolt"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".fabrik", "config.yaml"), []byte("x: 1\n"), 0o644))
	return repo
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
	f := &exec.Fake{}
	withFakeStore(t, &fakeLoopStore{})
	err := Run(context.Background(), RunConfig{RepoPath: t.TempDir()}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setup incomplete")
	assert.Empty(t, f.Calls)
}

func TestRunAutoRequiresPrompt(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{}
	withFakeStore(t, &fakeLoopStore{})
	err := Run(context.Background(), RunConfig{RepoPath: mkRepo(t), Auto: true}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--prompt")
}

func TestRunAutoPausesAfterPlanForApprove(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	prd := prdPromptPrefix + "add login"
	f := &exec.Fake{Outputs: map[string]string{
		"opencode run " + prd:         "prd-ok",
		"opencode run " + planPrompt: "plan-ok",
	}}
	withFakeStore(t, &fakeLoopStore{})
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
	f := &exec.Fake{Outputs: map[string]string{
		"opencode":                   "tui",
		"opencode run " + planPrompt: "plan-ok",
	}}
	withFakeStore(t, &fakeLoopStore{})
	err := Run(context.Background(), RunConfig{RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	require.NotEmpty(t, f.Calls)
	assert.Equal(t, "opencode", f.Calls[0])
	assert.Contains(t, buf.String(), "Step 1:")
}

func TestLoopBadMode(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{}
	withFakeStore(t, &fakeLoopStore{})
	err := Loop(context.Background(), LoopConfig{Mode: "bogus"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan|build")
}

func TestLoopNoStore(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{}
	old := OpenLoopStore
	OpenLoopStore = func(ctx context.Context, repo string) (LoopStore, error) {
		if _, ok := store.RepoRoot(repo); !ok {
			return nil, store.ErrNoStore
		}
		return nil, store.ErrNoStore
	}
	t.Cleanup(func() { OpenLoopStore = old })
	err := Loop(context.Background(), LoopConfig{Mode: "plan"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no store")
}

func TestLoopPlanRunsOnce(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	f := &exec.Fake{Outputs: map[string]string{"opencode run " + planPrompt: "plan-ok"}}
	withFakeStore(t, &fakeLoopStore{})
	err := Loop(context.Background(), LoopConfig{Mode: "plan", RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Equal(t, 1, countCalls(f, "opencode run"))
	assert.Contains(t, buf.String(), "FABRIK plan (iteration 1)")
	assert.Contains(t, buf.String(), "Planning complete")
	assert.False(t, f.Called("git push"), "push must not run without explicit opt-in")
}

func TestLoopPlanPushesWhenOptIn(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	f := &exec.Fake{Outputs: map[string]string{
		"opencode run " + planPrompt: "plan-ok",
		"git branch --show-current":  "main",
	}}
	withFakeStore(t, &fakeLoopStore{})
	err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true, RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.True(t, f.Called("git push"), "push must run with explicit AutoPush opt-in")
}

func TestLoopBuildNoOpenBreaks(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{}
	withFakeStore(t, &fakeLoopStore{})
	err := Loop(context.Background(), LoopConfig{Mode: "build", RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No open issues.")
	assert.Equal(t, 0, countCalls(f, "opencode run"))
}

func TestLoopBuildNoneReadyBreaks(t *testing.T) {
	var buf bytes.Buffer
	f := &exec.Fake{}
	withFakeStore(t, &fakeLoopStore{unfinished: 1, done: 0, ready: nil})
	err := Loop(context.Background(), LoopConfig{Mode: "build", RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "none are ready")
	assert.Equal(t, 0, countCalls(f, "opencode run"))
}

func TestLoopBuildMaxCapsIterations(t *testing.T) {
	var buf bytes.Buffer
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	id := "f-abc12345"
	fake := &fakeLoopStore{
		unfinished: 1,
		ready:      []LoopIssue{{ID: id, Title: "Do thing"}},
	}
	f := &exec.Fake{Outputs: map[string]string{"opencode run " + buildPrompt: "build-ok"}}
	withFakeStore(t, fake)
	err := Loop(context.Background(), LoopConfig{Mode: "build", Max: 2, RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Equal(t, 2, countCalls(f, "opencode run"))
	out := buf.String()
	assert.Contains(t, out, "FABRIK build (iteration 2)")
	assert.Contains(t, out, "claim "+id)
	assert.Contains(t, out, "Target: "+id+" — Do thing")
	assert.Contains(t, out, "Reached max iterations: 2")
	assert.Equal(t, []string{id, id}, fake.claims)
}

func TestLoopRunsInRepoDir(t *testing.T) {
	var buf bytes.Buffer
	repo := mkRepo(t)
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	f := &exec.Fake{Outputs: map[string]string{"opencode run " + planPrompt: "plan-ok"}}
	withFakeStore(t, &fakeLoopStore{})
	err := Loop(context.Background(), LoopConfig{RepoPath: repo, Mode: "plan"}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	require.NotEmpty(t, f.Dirs, "all commands must run in the repo dir")
	for i, dir := range f.Dirs {
		assert.Equal(t, repo, dir, "call %q ran outside the repo", f.Calls[i])
	}
}

func TestLoopBuildClaimsFirstReady(t *testing.T) {
	var buf bytes.Buffer
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	id := "f-deadbeef"
	fake := &fakeLoopStore{
		unfinished: 1,
		ready:      []LoopIssue{{ID: id, Title: "Wrapped"}},
	}
	f := &exec.Fake{Outputs: map[string]string{"opencode run " + buildPrompt: "build-ok"}}
	withFakeStore(t, fake)
	err := Loop(context.Background(), LoopConfig{Mode: "build", Max: 1, RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.Equal(t, 1, countCalls(f, "opencode run"))
	assert.Contains(t, buf.String(), "Target: "+id+" — Wrapped")
	assert.Equal(t, []string{id}, fake.claims)
}

func TestLoopOpencodeFailurePropagates(t *testing.T) {
	var buf bytes.Buffer
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	id := "f-abc12345"
	f := &exec.Fake{
		Outputs: map[string]string{"opencode run " + buildPrompt: "build-ok"},
		Errors:  map[string]error{"opencode run " + buildPrompt: assert.AnError},
	}
	withFakeStore(t, &fakeLoopStore{unfinished: 1, ready: []LoopIssue{{ID: id, Title: "Do thing"}}})
	err := Loop(context.Background(), LoopConfig{Mode: "build", Max: 1, RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opencode run build")
}

func TestLoopStoreErrorPropagates(t *testing.T) {
	var buf bytes.Buffer
	old := OpenLoopStore
	OpenLoopStore = func(ctx context.Context, repo string) (LoopStore, error) {
		return &errLoopStore{}, nil
	}
	t.Cleanup(func() { OpenLoopStore = old })
	err := Loop(context.Background(), LoopConfig{Mode: "build", RepoPath: mkRepo(t)}, Deps{Exec: &exec.Fake{}, Out: &buf, Sleep: noopSleep})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "store")
}

type errLoopStore struct{}

func (e *errLoopStore) Snapshot(ctx context.Context) (int, int, []LoopIssue, error) {
	return 0, 0, nil, assert.AnError
}
func (e *errLoopStore) ClaimNext(ctx context.Context, by string) (LoopIssue, error) {
	return LoopIssue{}, assert.AnError
}
func (e *errLoopStore) Close() error { return nil }

func TestLoopPushUsesOriginDashDash(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	f := &exec.Fake{Outputs: map[string]string{
		"opencode run " + planPrompt: "plan-ok",
		"git branch --show-current":  "feature/cool-thing",
	}}
	withFakeStore(t, &fakeLoopStore{})
	err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true, RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.True(t, f.Called("git push origin -- feature/cool-thing"), "push must use origin -- <branch>: %v", f.Calls)
}

func TestLoopPushSanitizesBranch(t *testing.T) {
	for _, evil := range []string{"-f", "evil; rm -rf", "a..b", "x y", ".hidden", ""} {
		var buf bytes.Buffer
		planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
		f := &exec.Fake{Outputs: map[string]string{
			"opencode run " + planPrompt: "plan-ok",
			"git branch --show-current": evil,
		}}
		withFakeStore(t, &fakeLoopStore{})
		err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true, RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
		require.NoError(t, err, "branch %q", evil)
		assert.False(t, f.Called("git push"), "unsafe/unknown branch %q must skip push: %v", evil, f.Calls)
		assert.Contains(t, buf.String(), "unknown branch", "branch %q must report skipped push", evil)
	}
}

func TestLoopPushSkippedWhenBranchUnknown(t *testing.T) {
	var buf bytes.Buffer
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	f := &exec.Fake{
		Outputs: map[string]string{"opencode run " + planPrompt: "plan-ok"},
		Errors:  map[string]error{"git branch --show-current": assert.AnError},
	}
	withFakeStore(t, &fakeLoopStore{})
	err := Loop(context.Background(), LoopConfig{Mode: "plan", AutoPush: true, RepoPath: mkRepo(t)}, Deps{Exec: f, Out: &buf, Sleep: noopSleep})
	require.NoError(t, err)
	assert.False(t, f.Called("git push"), "unknown branch (error) must skip push: %v", f.Calls)
	assert.Contains(t, buf.String(), "unknown branch")
}

func TestCurrentBranchReturnsEmptyOnUnknown(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	f := &exec.Fake{
		Errors: map[string]error{"git branch --show-current": assert.AnError},
	}
	assert.Empty(t, currentBranch(ctx, f, repo), "error must yield empty, not main fallback")
	f2 := &exec.Fake{Outputs: map[string]string{"git branch --show-current": ""}}
	assert.Empty(t, currentBranch(ctx, f2, repo), "empty output must yield empty")
	f3 := &exec.Fake{Outputs: map[string]string{"git branch --show-current": "-f"}}
	assert.Empty(t, currentBranch(ctx, f3, repo), "invalid branch must yield empty")
	f4 := &exec.Fake{Outputs: map[string]string{"git branch --show-current": "feature/ok"}}
	assert.Equal(t, "feature/ok", currentBranch(ctx, f4, repo))
}
