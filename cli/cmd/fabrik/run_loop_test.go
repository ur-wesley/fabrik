package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

func withFakeRunner(t *testing.T, f *exec.Fake) {
	t.Helper()
	old := execRunner
	execRunner = f
	t.Cleanup(func() { execRunner = old })
}

func TestLoopRejectsBadMode(t *testing.T) {
	withFakeRunner(t, &exec.Fake{Path: map[string]bool{"bd": true}})
	_, err := run(t, "loop", "bogus")
	require.Error(t, err)
}

func TestLoopPlanOnceViaCLI(t *testing.T) {
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd list --status=open --json":               `[]`,
		"bd list --status=closed --json --limit 100": `[]`,
		"bd ready --json":                            `[]`,
		"opencode run " + planPrompt:                 "plan-ok",
	}}
	withFakeRunner(t, f)
	out, err := run(t, "loop", "plan")
	require.NoError(t, err)
	assert.Contains(t, out, "FABRIK plan (iteration 1)")
	assert.True(t, f.Called("opencode run"))
}

func TestRunSetupGateViaCLI(t *testing.T) {
	withFakeRunner(t, &exec.Fake{Path: map[string]bool{"bd": true}})
	_, err := run(t, "run", "--auto", "-p", "something")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setup incomplete")
}

func TestRunRepoFlagViaCLI(t *testing.T) {
	withFakeRunner(t, &exec.Fake{Path: map[string]bool{"bd": true}})
	_, err := run(t, "run", "--repo", t.TempDir(), "--auto", "-p", "something")
	require.Error(t, err, "--repo must reach the setup gate")
	assert.Contains(t, err.Error(), "setup incomplete")
}

func TestLoopMaxPassthroughViaCLI(t *testing.T) {
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd list --status=open --json":               `[{"id":"bd-1","title":"Do thing"}]`,
		"bd list --status=closed --json --limit 100": `[]`,
		"bd ready --json":                            `[{"id":"bd-1","title":"Do thing"}]`,
		"opencode run " + buildPrompt:                "build-ok",
	}}
	withFakeRunner(t, f)
	out, err := run(t, "loop", "build", "--max", "1")
	require.NoError(t, err)
	assert.Contains(t, out, "Reached max iterations: 1")
	n := 0
	for _, c := range f.Calls {
		if strings.HasPrefix(c, "opencode run") {
			n++
		}
	}
	assert.Equal(t, 1, n, "--max must cap iterations: %v", f.Calls)
}
