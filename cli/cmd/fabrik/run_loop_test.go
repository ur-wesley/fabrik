package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/flow"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

func withFakeRunner(t *testing.T, f *exec.Fake) {
	t.Helper()
	old := execRunner
	execRunner = f
	t.Cleanup(func() { execRunner = old })
}

func loopRepo(t *testing.T) string {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".fabrik", "dolt"), 0o755))
	return repo
}

func withCLILoopStore(t *testing.T, fake flow.LoopStore) {
	old := flow.OpenLoopStore
	flow.OpenLoopStore = func(ctx context.Context, repo string) (flow.LoopStore, error) {
		return fake, nil
	}
	t.Cleanup(func() { flow.OpenLoopStore = old })
}

func TestLoopRejectsBadMode(t *testing.T) {
	withFakeRunner(t, &exec.Fake{})
	withCLILoopStore(t, &flowFakeStore{})
	_, err := run(t, "loop", "bogus")
	require.Error(t, err)
}

type flowFakeStore struct{}

func (flowFakeStore) Snapshot(ctx context.Context) (int, int, []flow.LoopIssue, error) {
	return 0, 0, nil, nil
}
func (flowFakeStore) ClaimNext(ctx context.Context, by string) (flow.LoopIssue, error) {
	return flow.LoopIssue{ID: "f-plan0001"}, nil
}
func (flowFakeStore) Close() error { return nil }

func TestLoopPlanOnceViaCLI(t *testing.T) {
	planPrompt := tmpl.MustRead("hub/PROMPT_plan.md")
	f := &exec.Fake{Outputs: map[string]string{"opencode run " + planPrompt: "plan-ok"}}
	withFakeRunner(t, f)
	withCLILoopStore(t, &flowFakeStore{})
	dir := loopRepo(t)
	t.Chdir(dir)
	out, err := run(t, "loop", "plan")
	require.NoError(t, err)
	assert.Contains(t, out, "FABRIK plan (iteration 1)")
	assert.True(t, f.Called("opencode run"))
}

func TestRunSetupGateViaCLI(t *testing.T) {
	withFakeRunner(t, &exec.Fake{})
	_, err := run(t, "run", "--auto", "-p", "something")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "setup incomplete")
}

func TestRunRepoFlagViaCLI(t *testing.T) {
	withFakeRunner(t, &exec.Fake{})
	_, err := run(t, "run", "--repo", t.TempDir(), "--auto", "-p", "something")
	require.Error(t, err, "--repo must reach the setup gate")
	assert.Contains(t, err.Error(), "setup incomplete")
}

func TestLoopMaxPassthroughViaCLI(t *testing.T) {
	buildPrompt := tmpl.MustRead("hub/PROMPT_build.md")
	f := &exec.Fake{Outputs: map[string]string{"opencode run " + buildPrompt: "build-ok"}}
	withFakeRunner(t, f)
	id := "f-abc12345"
	withCLILoopStore(t, &flowFakeBuildStore{id: id})
	dir := loopRepo(t)
	t.Chdir(dir)
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

type flowFakeBuildStore struct{ id string }

func (s *flowFakeBuildStore) Snapshot(ctx context.Context) (int, int, []flow.LoopIssue, error) {
	return 1, 0, []flow.LoopIssue{{ID: s.id, Title: "Do thing"}}, nil
}
func (s *flowFakeBuildStore) ClaimNext(ctx context.Context, by string) (flow.LoopIssue, error) {
	return flow.LoopIssue{ID: s.id, Title: "Do thing"}, nil
}
func (s *flowFakeBuildStore) Close() error { return nil }
