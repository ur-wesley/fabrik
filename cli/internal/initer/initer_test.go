package initer

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

// expectedThinSkills derives from the embedded templates (single source:
// tmpl.List via thinSkillNames), so new skills are picked up automatically.
func expectedThinSkills() []string { return thinSkillNames() }

func TestInitThinDefault(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, out := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d))

	// thin hub: only config.yaml + docs/specs dirs
	assert.FileExists(t, filepath.Join(repo, ".fabrik", "config.yaml"))
	assert.DirExists(t, filepath.Join(repo, ".fabrik", "docs"))
	assert.DirExists(t, filepath.Join(repo, ".fabrik", "specs"))

	// legacy dump must NOT exist in thin mode
	for _, p := range []string{
		".fabrik/README.md", ".fabrik/CONTEXT.md", ".fabrik/skills.md",
		".fabrik/AGENTS.md", ".fabrik/.gitignore",
		".fabrik/styleguide/STYLEGUIDE.md",
		".fabrik/PROMPT_plan.md", ".fabrik/PROMPT_build.md",
		".fabrik/loop.sh", ".fabrik/loop.ps1", ".fabrik/fabrik.sh", ".fabrik/fabrik.ps1",
	} {
		assert.NoFileExists(t, filepath.Join(repo, filepath.FromSlash(p)), p)
	}
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik", "agents"))
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik", "styleguide"))

	// opencode.json roster (show-shims) kept in thin mode
	assert.FileExists(t, filepath.Join(repo, "opencode.json"))
	assert.NoDirExists(t, filepath.Join(repo, ".opencode", "agents"))

	// cli-s34: thin opencode.json must not dangle — zero {file:} refs,
	// agents resolve via `fabrik show agent <name>` shims instead.
	thinJSON, err := os.ReadFile(filepath.Join(repo, "opencode.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(thinJSON), "{file:")
	assert.Contains(t, string(thinJSON), "fabrik show agent ")

	// skill shims point at fabrik show (canonical skillShim bodies)
	thinSkills := expectedThinSkills()
	require.NotEmpty(t, thinSkills, "thin skills derive from tmpl.List, must not be empty")
	assert.Contains(t, thinSkills, "i-have-adhd")
	assert.Contains(t, thinSkills, "tdd")
	for _, s := range thinSkills {
		for _, p := range []string{
			filepath.Join(".cursor", "rules", s+".mdc"),
			filepath.Join(".opencode", "skills", s+".md"),
			filepath.Join(".pi", "skills", s+".md"),
			filepath.Join(".agents", "skills", s, "SKILL.md"),
		} {
			full := filepath.Join(repo, p)
			assert.FileExists(t, full, p)
			body, err := os.ReadFile(full)
			require.NoError(t, err)
			assert.Contains(t, string(body), "Run: fabrik show skill "+s, p)
		}
	}

	// agent shims point at fabrik show (cta is a fragment, not installed)
	pins, err := deps.Load()
	require.NoError(t, err)
	require.NotEmpty(t, pins.Subagents)
	for _, a := range selectableAgents(pins.Subagents) {
		piPath := filepath.Join(repo, ".pi", "agent", "agents", a+".md")
		assert.FileExists(t, piPath, a)
		piBody, err := os.ReadFile(piPath)
		require.NoError(t, err)
		assert.Contains(t, string(piBody), "Run: fabrik show agent "+a)

		cursorPath := filepath.Join(repo, ".cursor", "agents", a+".md")
		assert.FileExists(t, cursorPath, a)
		cursorBody, err := os.ReadFile(cursorPath)
		require.NoError(t, err)
		assert.Contains(t, string(cursorBody), "name: "+a)
		assert.Contains(t, string(cursorBody), "Run: fabrik show agent "+a)
	}
	assert.NoFileExists(t, filepath.Join(repo, ".cursor", "agents", "cta.md"))
	assert.NoDirExists(t, filepath.Join(repo, ".cursor", "rules", "agents"))

	// root AGENTS.md is a short pointer, not the full workflow note
	agentsPath := filepath.Join(repo, "AGENTS.md")
	body, err := os.ReadFile(agentsPath)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Fabrik workflow")
	assert.Contains(t, string(body), "fabrik show workflow")
	assert.LessOrEqual(t, len(strings.Split(strings.TrimSpace(string(body)), "\n")), 8, "pointer AGENTS.md must stay short")

	// wiring still invoked inside the repo, no stray dirs
	assert.True(t, f.Called("bd setup cursor"))
	assert.True(t, f.Called("bd setup opencode"))
	require.NotEmpty(t, f.Dirs)
	for _, dir := range f.Dirs {
		assert.Equal(t, repo, dir)
	}
	assert.DirExists(t, filepath.Join(repo, ".agents"))
	assert.NoDirExists(t, filepath.Join(repo, ".claude"))
	assert.NoDirExists(t, filepath.Join(repo, ".codex"))
	assert.Contains(t, out.String(), "Done.")
}

func TestInitFullLegacy(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, out := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true, Full: true}, d))

	// hub
	for _, p := range []string{
		".fabrik/docs", ".fabrik/specs", ".fabrik/styleguide", ".fabrik/agents",
	} {
		assert.DirExists(t, filepath.Join(repo, filepath.FromSlash(p)), p)
	}
	for _, p := range []string{
		".fabrik/README.md", ".fabrik/CONTEXT.md", ".fabrik/config.yaml", ".fabrik/skills.md",
		".fabrik/.gitignore", ".fabrik/styleguide/STYLEGUIDE.md",
		".fabrik/agents/plan.md", ".fabrik/agents/orchestrator.md", ".fabrik/agents/explore.md", ".fabrik/agents/researcher.md",
		".fabrik/agents/briefer.md", ".fabrik/agents/planner.md", ".fabrik/agents/builder.md",
		".fabrik/agents/tester.md", ".fabrik/agents/reviewer.md", ".fabrik/agents/style-smells.md",
		".fabrik/agents/security.md",
		"AGENTS.md",
	} {
		assert.FileExists(t, filepath.Join(repo, filepath.FromSlash(p)), p)
	}
	// no shell runners in any mode: CLI run/loop/show cover them
	for _, p := range []string{
		".fabrik/loop.sh", ".fabrik/loop.ps1", ".fabrik/fabrik.sh", ".fabrik/fabrik.ps1",
		".fabrik/setup.sh", ".fabrik/setup.ps1",
	} {
		assert.NoFileExists(t, filepath.Join(repo, filepath.FromSlash(p)), p)
	}
	// skills x4 apps
	for _, s := range []string{"i-have-adhd", "tdd", "guardrails"} {
		assert.FileExists(t, filepath.Join(repo, ".cursor", "rules", s+".mdc"))
		assert.FileExists(t, filepath.Join(repo, ".opencode", "skills", s+".md"))
		assert.FileExists(t, filepath.Join(repo, ".pi", "skills", s+".md"))
		assert.FileExists(t, filepath.Join(repo, ".agents", "skills", s, "SKILL.md"))
	}
	// opencode.json replaces built-in build with Fabrik roster
	assert.FileExists(t, filepath.Join(repo, "opencode.json"))
	assert.NoDirExists(t, filepath.Join(repo, ".opencode", "agents"))
	// cli-s34: full opencode.json carries the same zero-{file:} show-shims
	fullJSON, err := os.ReadFile(filepath.Join(repo, "opencode.json"))
	require.NoError(t, err)
	assert.NotContains(t, string(fullJSON), "{file:")
	assert.Contains(t, string(fullJSON), "fabrik show agent ")
	// subagents -> Pi + Cursor copies
	assert.FileExists(t, filepath.Join(repo, ".pi", "agent", "agents", "reviewer.md"))
	cursorBuilder, err := os.ReadFile(filepath.Join(repo, ".cursor", "agents", "builder.md"))
	require.NoError(t, err)
	assert.Contains(t, string(cursorBuilder), "name: builder")
	assert.Contains(t, string(cursorBuilder), "# Builder subagent")
	assert.NoDirExists(t, filepath.Join(repo, ".cursor", "rules", "agents"))
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
	assert.DirExists(t, filepath.Join(repo, ".agents"))
	assert.NoDirExists(t, filepath.Join(repo, ".claude"))
	assert.NoDirExists(t, filepath.Join(repo, ".codex"))
	// AGENTS.md has workflow block
	legacyBody, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	require.NoError(t, err)
	assert.Contains(t, string(legacyBody), "Fabrik workflow")
	assert.Contains(t, out.String(), "Done.")
}

func TestInitSelectiveAppsCursorOnly(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true, Apps: []string{"cursor"}}, d))

	assert.FileExists(t, filepath.Join(repo, ".cursor", "rules", "tdd.mdc"))
	assert.NoDirExists(t, filepath.Join(repo, ".opencode"))
	assert.NoDirExists(t, filepath.Join(repo, ".pi"))
	assert.NoDirExists(t, filepath.Join(repo, ".agents"))
	assert.NoFileExists(t, filepath.Join(repo, "opencode.json"))

	assert.True(t, f.Called("bd setup cursor"))
	assert.False(t, f.Called("bd setup opencode"))
}

func TestInitSelectiveAppsAntigravityOnly(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true, "engram": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
		"engram setup antigravity-cli":                            "",
	}}
	d, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true, Apps: []string{"antigravity"}}, d))

	assert.FileExists(t, filepath.Join(repo, ".agents", "skills", "tdd", "SKILL.md"))
	assert.NoDirExists(t, filepath.Join(repo, ".cursor"))
	assert.NoDirExists(t, filepath.Join(repo, ".opencode"))
	assert.NoDirExists(t, filepath.Join(repo, ".pi"))
	assert.NoFileExists(t, filepath.Join(repo, "opencode.json"))

	assert.False(t, f.Called("bd setup cursor"))
	assert.True(t, f.Called("engram setup antigravity-cli"))
}

func TestInitInteractiveSelectApps(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, _ := testEnv(t, repo, f)
	d.SelectApps = func(defaults []string, yes bool) ([]string, error) {
		return []string{"pi", "antigravity"}, nil
	}
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: false}, d))

	assert.FileExists(t, filepath.Join(repo, ".pi", "skills", "tdd.md"))
	assert.FileExists(t, filepath.Join(repo, ".agents", "skills", "tdd", "SKILL.md"))
	assert.NoDirExists(t, filepath.Join(repo, ".cursor"))
	assert.NoDirExists(t, filepath.Join(repo, ".opencode"))
}

func TestInitInvalidAppFails(t *testing.T) {
	repo := t.TempDir()
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, _ := testEnv(t, repo, f)
	err := Init(context.Background(), Config{RepoPath: repo, Yes: true, Apps: []string{"unsupported_app"}}, d)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid app")
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

func TestInitAppendsPointerToExistingAgents(t *testing.T) {
	repo := t.TempDir()
	agentsPath := filepath.Join(repo, "AGENTS.md")
	require.NoError(t, os.WriteFile(agentsPath, []byte("# my existing notes\n"), 0o644))
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d))
	body, err := os.ReadFile(agentsPath)
	require.NoError(t, err)
	assert.Contains(t, string(body), "# my existing notes", "existing content must be preserved")
	assert.Contains(t, string(body), "fabrik show workflow", "pointer must be appended")
	assert.Equal(t, 1, strings.Count(string(body), "Fabrik workflow"), "second init must not duplicate the pointer")
	d2, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d2))
	again, err := os.ReadFile(agentsPath)
	require.NoError(t, err)
	assert.Equal(t, string(body), string(again))
}

func TestInitPreservesExistingOpencodeJsonAndShims(t *testing.T) {
	repo := t.TempDir()
	customJSON := []byte("{\"custom\": true}\n")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "opencode.json"), customJSON, 0o644))
	customShim := filepath.Join(repo, ".opencode", "skills", "tdd.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(customShim), 0o755))
	require.NoError(t, os.WriteFile(customShim, []byte("my custom shim\n"), 0o644))
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd init --non-interactive --skip-agents --skip-hooks -q": "",
	}}
	d, _ := testEnv(t, repo, f)
	require.NoError(t, Init(context.Background(), Config{RepoPath: repo, Yes: true}, d))
	gotJSON, err := os.ReadFile(filepath.Join(repo, "opencode.json"))
	require.NoError(t, err)
	assert.Equal(t, string(customJSON), string(gotJSON), "existing opencode.json must not be overwritten")
	gotShim, err := os.ReadFile(customShim)
	require.NoError(t, err)
	assert.Equal(t, "my custom shim\n", string(gotShim), "existing shim must not be overwritten")
}
