package reseter

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/initer"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

func testEnv(t *testing.T, f *exec.Fake) (Deps, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	return Deps{
		Exec:    f,
		Out:     &out,
		Confirm: func(prompt string, defaultYes bool) bool { return true },
	}, &out
}

func initTestRepo(t *testing.T, f *exec.Fake) string {
	t.Helper()
	repo := t.TempDir()
	initDeps := initer.Deps{
		Exec:  f,
		Out:   &bytes.Buffer{},
		Check: func(ctx context.Context) error { return nil },
	}
	require.NoError(t, initer.Init(context.Background(), initer.Config{RepoPath: repo, Yes: true}, initDeps))
	return repo
}

func TestResetThinHubCleanWithBeads(t *testing.T) {
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd setup cursor --remove":   "",
		"bd setup opencode --remove": "",
	}}
	repo := initTestRepo(t, f)

	// verify files exist after init
	assert.FileExists(t, filepath.Join(repo, "opencode.json"))
	assert.FileExists(t, filepath.Join(repo, "AGENTS.md"))
	assert.FileExists(t, filepath.Join(repo, ".fabrik", "config.yaml"))
	assert.DirExists(t, filepath.Join(repo, ".fabrik"))
	assert.DirExists(t, filepath.Join(repo, ".cursor"))
	assert.DirExists(t, filepath.Join(repo, ".opencode"))
	assert.DirExists(t, filepath.Join(repo, ".pi"))

	// create mock beads files
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".beads"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".beads", "config.yaml"), []byte("beads: true\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".cursor", "rules", "beads.mdc"), []byte("beads rule\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".cursor", "hooks.json"), []byte("{}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".beads.gate.lock"), []byte("lock\n"), 0o644))

	deps, out := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true, Beads: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	if body, err := os.ReadFile(filepath.Join(repo, "opencode.json")); err == nil {
		assert.NotContains(t, string(body), "fabrik show agent")
	}
	assert.NoFileExists(t, filepath.Join(repo, "AGENTS.md"))
	assert.FileExists(t, filepath.Join(repo, ".fabrik", "config.yaml"))
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik", "dolt"))
	assert.NoDirExists(t, filepath.Join(repo, ".beads"))
	assert.NoFileExists(t, filepath.Join(repo, ".beads.gate.lock"))
	assert.NoDirExists(t, filepath.Join(repo, ".cursor"))
	assert.NoDirExists(t, filepath.Join(repo, ".opencode"))
	assert.NoDirExists(t, filepath.Join(repo, ".pi"))

	assert.True(t, f.Called("bd setup cursor --remove"))
	assert.True(t, f.Called("bd setup opencode --remove"))
	assert.Contains(t, out.String(), "Done.")
}

func TestResetPreservesFabrikHub(t *testing.T) {
	f := &exec.Fake{}
	repo := initTestRepo(t, f)

	// Add user specs and docs into .fabrik/
	specPath := filepath.Join(repo, ".fabrik", "specs", "feature-xyz.md")
	docPath := filepath.Join(repo, ".fabrik", "docs", "architecture.md")
	require.NoError(t, os.WriteFile(specPath, []byte("# My Spec\n"), 0o644))
	require.NoError(t, os.WriteFile(docPath, []byte("# My Doc\n"), 0o644))

	deps, _ := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true, NoBeads: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	assert.DirExists(t, filepath.Join(repo, ".fabrik"))
	assert.FileExists(t, specPath)
	assert.FileExists(t, docPath)
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik", "dolt"))
}

func TestResetPreservesUserFiles(t *testing.T) {
	f := &exec.Fake{}
	repo := initTestRepo(t, f)

	// Add custom user rules and skills
	customRule := filepath.Join(repo, ".cursor", "rules", "my-custom.mdc")
	customSkill := filepath.Join(repo, ".opencode", "skills", "custom-skill.md")
	require.NoError(t, os.WriteFile(customRule, []byte("# custom cursor rule\n"), 0o644))
	require.NoError(t, os.WriteFile(customSkill, []byte("# custom opencode skill\n"), 0o644))

	deps, _ := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true, NoBeads: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	// Fabrik skills are deleted
	assert.NoFileExists(t, filepath.Join(repo, ".cursor", "rules", "tdd.mdc"))
	assert.NoFileExists(t, filepath.Join(repo, ".opencode", "skills", "tdd.md"))

	// User files are preserved!
	assert.FileExists(t, customRule)
	assert.FileExists(t, customSkill)
	assert.DirExists(t, filepath.Join(repo, ".cursor", "rules"))
	assert.DirExists(t, filepath.Join(repo, ".opencode", "skills"))
}

func TestResetPreservesUserAgentsMd(t *testing.T) {
	f := &exec.Fake{Path: map[string]bool{"bd": true}}
	repo := t.TempDir()

	// Existing AGENTS.md with user instructions
	initial := "# Agent instructions\n\n## Custom Team Rules\nDo not break things.\n\n## Fabrik workflow\n\nRun `fabrik show workflow` for the full workflow.\n"
	agentsPath := filepath.Join(repo, "AGENTS.md")
	require.NoError(t, os.WriteFile(agentsPath, []byte(initial), 0o644))

	deps, _ := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true, NoBeads: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	assert.FileExists(t, agentsPath)
	body, err := os.ReadFile(agentsPath)
	require.NoError(t, err)
	assert.Contains(t, string(body), "## Custom Team Rules")
	assert.NotContains(t, string(body), "## Fabrik workflow")
}

func TestResetDryRunTouchesNothing(t *testing.T) {
	f := &exec.Fake{}
	repo := initTestRepo(t, f)

	deps, out := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true, DryRun: true, Beads: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	assert.Contains(t, out.String(), "would delete")
	// All files still exist
	assert.FileExists(t, filepath.Join(repo, "opencode.json"))
	assert.FileExists(t, filepath.Join(repo, "AGENTS.md"))
	assert.DirExists(t, filepath.Join(repo, ".fabrik"))
}

func TestResetDeclined(t *testing.T) {
	f := &exec.Fake{}
	repo := t.TempDir()

	var out bytes.Buffer
	deps := Deps{
		Exec:    f,
		Out:     &out,
		Confirm: func(prompt string, defaultYes bool) bool { return false },
	}

	cfg := Config{RepoPath: repo, Yes: false}
	err := Reset(context.Background(), cfg, deps)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "declined")
}

func TestResetBeadsPrompt(t *testing.T) {
	f := &exec.Fake{Path: map[string]bool{"bd": true}, Outputs: map[string]string{
		"bd setup cursor --remove":   "",
		"bd setup opencode --remove": "",
	}}
	repo := initTestRepo(t, f)
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".beads"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".cursor", "rules", "beads.mdc"), []byte("beads\n"), 0o644))

	// Prompt answers yes to reset, but no to beads
	var out bytes.Buffer
	deps := Deps{
		Exec: f,
		Out:  &out,
		Confirm: func(prompt string, defaultYes bool) bool {
			if prompt == "Remove Beads integration and .beads as well?" {
				return false
			}
			return true
		},
	}

	cfg := Config{RepoPath: repo, Yes: false}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	assert.FileExists(t, filepath.Join(repo, ".fabrik", "config.yaml"))
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik", "dolt"))
	assert.DirExists(t, filepath.Join(repo, ".beads"))
	assert.FileExists(t, filepath.Join(repo, ".cursor", "rules", "beads.mdc"))
	assert.False(t, f.Called("bd setup cursor --remove"))
}

func TestResetUpdatesCustomOpencodeJson(t *testing.T) {
	f := &exec.Fake{}
	repo := t.TempDir()

	customJSON := `{
  "$schema": "https://opencode.ai/config.json",
  "default_agent": "plan",
  "agent": {
    "plan": {
      "prompt": "# Fabrik agent Plan\n\nRun: fabrik show agent plan\n"
    },
    "my-custom-agent": {
      "prompt": "Custom prompt"
    }
  }
}
`
	opencodePath := filepath.Join(repo, "opencode.json")
	require.NoError(t, os.WriteFile(opencodePath, []byte(customJSON), 0o644))

	deps, _ := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	assert.FileExists(t, opencodePath)
	body, err := os.ReadFile(opencodePath)
	require.NoError(t, err)
	assert.Contains(t, string(body), "my-custom-agent")
	assert.NotContains(t, string(body), "fabrik show agent")
	assert.NotContains(t, string(body), `"default_agent": "plan"`)
}

func TestResetDeletesPureOpencodeJson(t *testing.T) {
	f := &exec.Fake{}
	repo := t.TempDir()

	opencodePath := filepath.Join(repo, "opencode.json")
	require.NoError(t, os.WriteFile(opencodePath, []byte(tmpl.MustRead("opencode.json")), 0o644))

	deps, _ := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	assert.NoFileExists(t, opencodePath)
}

// hasBareLF reports a "\n" not preceded by "\r" (i.e. a line ending that
// broke CRLF consistency).
func hasBareLF(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' && (i == 0 || s[i-1] != '\r') {
			return true
		}
	}
	return false
}

// cli-5t3: CRLF fixtures must round-trip with CRLF preserved — stripping the
// Fabrik workflow must not flip the surviving lines to LF.
func TestStripAgentsMdPreservesCRLF(t *testing.T) {
	body := "# Agent instructions\r\n\r\n## Custom Team Rules\r\nDo not break things.\r\n\r\n## Fabrik workflow\r\n\r\nRun `fabrik show workflow`.\r\n\r\n## More user docs\r\nKeep me.\r\n"
	action, updated := stripAgentsMd(body, false)
	require.Equal(t, AgentsMdUpdate, action)
	assert.Contains(t, updated, "## Custom Team Rules")
	assert.NotContains(t, updated, "## Fabrik workflow")
	assert.Contains(t, updated, "## More user docs")
	assert.False(t, hasBareLF(updated), "CRLF must be preserved, got %q", updated)
	assert.True(t, strings.HasSuffix(updated, "\r\n"))
}

// cli-5t3: content without Fabrik markers is byte-identical (LF and CRLF).
func TestStripAgentsMdRoundTripsCleanFiles(t *testing.T) {
	lf := "# Agent instructions\n\n## Custom\nKeep.\n"
	action, updated := stripAgentsMd(lf, false)
	assert.Equal(t, AgentsMdNone, action)
	assert.Equal(t, lf, updated)

	crlf := "# Agent instructions\r\n\r\n## Custom\r\nKeep.\r\n"
	action, updated = stripAgentsMd(crlf, false)
	assert.Equal(t, AgentsMdNone, action)
	assert.Equal(t, crlf, updated)
}

// cli-5t3: end-to-end Reset on a CRLF AGENTS.md preserves CRLF endings.
func TestResetPreservesCRLFAgentsMd(t *testing.T) {
	f := &exec.Fake{}
	repo := t.TempDir()

	initial := "# Agent instructions\r\n\r\n## Custom Team Rules\r\nDo not break things.\r\n\r\n## Fabrik workflow\r\n\r\nRun `fabrik show workflow` for the full workflow.\r\n"
	agentsPath := filepath.Join(repo, "AGENTS.md")
	require.NoError(t, os.WriteFile(agentsPath, []byte(initial), 0o644))

	deps, _ := testEnv(t, f)
	cfg := Config{RepoPath: repo, Yes: true, NoBeads: true}
	require.NoError(t, Reset(context.Background(), cfg, deps))

	raw, err := os.ReadFile(agentsPath)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "## Custom Team Rules")
	assert.NotContains(t, string(raw), "## Fabrik workflow")
	assert.False(t, hasBareLF(string(raw)), "CRLF must be preserved, got %q", string(raw))
}

// cli-5t3: .gitignore stripping preserves CRLF and keeps user lines.
func TestStripFabrikGitignorePreservesCRLF(t *testing.T) {
	repo := t.TempDir()
	p := filepath.Join(repo, ".gitignore")
	require.NoError(t, os.WriteFile(p, []byte("node_modules/\r\n.fabrik/dolt/\r\n.fabrik/graph.db\r\ndist/\r\n"), 0o644))

	var out bytes.Buffer
	require.NoError(t, stripFabrikGitignore(repo, &out))
	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "node_modules/\r\ndist/\r\n", string(raw))
	assert.False(t, hasBareLF(string(raw)))
}
