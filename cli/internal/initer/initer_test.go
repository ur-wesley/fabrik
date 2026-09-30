package initer

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
)

func testEnv(t *testing.T, repo string, f *exec.Fake) (Deps, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	return Deps{Exec: f, Out: &out, Check: func(ctx context.Context) error { return nil }}, &out
}

func TestInitFreshRepo(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, out := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d))

	// hub
	for _, p := range []string{
		".fabrik/docs", ".fabrik/specs", ".fabrik/styleguide", ".fabrik/agents",
	} {
		assert.DirExists(t, filepath.Join(repo, filepath.FromSlash(p)), p)
	}
	for _, p := range []string{
		".fabrik/README.md", ".fabrik/CONTEXT.md", ".fabrik/config.yaml", ".fabrik/skills.md",
		".fabrik/.gitignore", ".fabrik/styleguide/STYLEGUIDE.md",
		".fabrik/agents/orchestrator.md", ".fabrik/agents/explore.md", ".fabrik/agents/researcher.md",
		".fabrik/agents/briefer.md", ".fabrik/agents/planner.md", ".fabrik/agents/builder.md",
		".fabrik/agents/tester.md", ".fabrik/agents/reviewer.md", ".fabrik/agents/style-smells.md",
		".fabrik/agents/security.md",
		"AGENTS.md",
	} {
		assert.FileExists(t, filepath.Join(repo, filepath.FromSlash(p)), p)
	}
	// skills x3 apps
	for _, s := range []string{"caveman", "tdd", "guardrails"} {
		assert.FileExists(t, filepath.Join(repo, ".cursor", "rules", s+".mdc"))
		assert.FileExists(t, filepath.Join(repo, ".opencode", "skills", s+".md"))
		assert.FileExists(t, filepath.Join(repo, ".pi", "skills", s+".md"))
	}
	// opencode.json replaces built-in build with Fabrik roster
	assert.FileExists(t, filepath.Join(repo, "opencode.json"))
	assert.NoDirExists(t, filepath.Join(repo, ".opencode", "agents"))
	// subagents -> Pi + Cursor copies
	assert.FileExists(t, filepath.Join(repo, ".pi", "agent", "agents", "reviewer.md"))
	assert.FileExists(t, filepath.Join(repo, ".cursor", "rules", "agents", "builder.mdc"))
	// every pinned subagent lands in the hub
	pins, err := deps.Load()
	require.NoError(t, err)
	for _, a := range pins.Subagents {
		assert.FileExists(t, filepath.Join(repo, ".fabrik", "agents", a+".md"), a)
	}
	// agent wiring invoked
	assert.True(t, f.Called("bd setup cursor"))
	assert.True(t, f.Called("bd setup opencode"))
	// all repo-scoped commands must run inside the repo, not the CLI cwd
	require.NotEmpty(t, f.Dirs)
	for _, dir := range f.Dirs {
		assert.Equal(t, repo, dir)
	}
	// no stray dirs
	assert.NoDirExists(t, filepath.Join(repo, ".agents"))
	assert.NoDirExists(t, filepath.Join(repo, ".claude"))
	assert.NoDirExists(t, filepath.Join(repo, ".codex"))
	// AGENTS.md has workflow block
	body, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "Fabrik workflow")
	assert.Contains(t, out.String(), "Done.")
}

func TestInitIdempotent(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d))
	agents, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	require.NoError(t, err)
	d2, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d2))
	again, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	require.NoError(t, err)
	assert.Equal(t, string(agents), string(again), "second init must not duplicate workflow block")
}

func TestInitRequiresBd(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{}}
	d, _ := testEnv(t, repo, f)
	err := Init(context.Background(), Config{RepoPath: repo, Yes: true, SkipChecks: true}, d)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bd not found")
}

func TestInitDryRunTouchesNothing(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}}
	d, out := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true, DryRun: true}, d))
	assert.Contains(t, out.String(), "dry-run")
	entries, err := os.ReadDir(repo)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.False(t, f.Called("bd init"))
}

func TestInitPreservesExistingSideEffectDirs(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".agents"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".agents", "keep.md"), []byte("x"), 0o644))
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d))
	assert.FileExists(t, filepath.Join(repo, ".agents", "keep.md"))
}
