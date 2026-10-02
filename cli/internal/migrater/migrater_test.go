package migrater

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

// fileRefRe matches opencode.json {file:<rel>} refs.
var fileRefRe = regexp.MustCompile(`\{file:([^}]+)\}`)

func seed(t *testing.T, repo, rel, body string) {
	t.Helper()
	full := filepath.Join(repo, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
}

func seedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	full := "FULL BODY generated dump without any shim marker\n# line\n"
	// generated hub files
	seed(t, repo, ".fabrik/PROMPT_plan.md", full)
	seed(t, repo, ".fabrik/PROMPT_build.md", full)
	seed(t, repo, ".fabrik/loop.sh", full)
	seed(t, repo, ".fabrik/fabrik.sh", full)
	seed(t, repo, ".fabrik/setup.sh", full)
	seed(t, repo, ".fabrik/state.md", full)
	seed(t, repo, ".fabrik/README.md", full)
	seed(t, repo, ".fabrik/CONTEXT.md", full)
	seed(t, repo, ".fabrik/skills.md", full)
	seed(t, repo, ".fabrik/AGENTS.md", full)
	seed(t, repo, ".fabrik/styleguide/STYLEGUIDE.md", full)
	seed(t, repo, ".fabrik/agents/builder.md", full)
	// generated skill/agent copies (full bodies)
	seed(t, repo, ".cursor/rules/tdd.mdc", full)
	seed(t, repo, ".cursor/rules/agents/builder.mdc", full)
	seed(t, repo, ".cursor/agents/builder.md", full)
	seed(t, repo, ".opencode/skills/tdd.md", full)
	seed(t, repo, ".pi/skills/tdd.md", full)
	seed(t, repo, ".pi/agent/agents/builder.md", full)
	// shims that must be kept (canonical skillShim/agentShim bodies)
	seed(t, repo, ".cursor/rules/ponytail.mdc", "# Fabrik skill ponytail\n\nRun: fabrik show skill ponytail\n")
	seed(t, repo, ".opencode/skills/ponytail.md", "# Fabrik skill ponytail\n\nRun: fabrik show skill ponytail\n")
	seed(t, repo, ".pi/skills/ponytail.md", "# Fabrik skill ponytail\n\nRun: fabrik show skill ponytail\n")
	seed(t, repo, ".pi/agent/agents/planner.md", "# Fabrik subagent planner\n\nRun: fabrik show agent planner\n")
	seed(t, repo, ".cursor/agents/planner.md", "---\nname: planner\ndescription: Post-APPROVE land\nmodel: inherit\nreadonly: false\n---\n\n# Fabrik subagent planner\n\nRun: fabrik show agent planner\n")
	// user files that must never be touched
	seed(t, repo, ".fabrik/config.yaml", "version: 1\n")
	seed(t, repo, ".fabrik/docs/PRD.md", "# user PRD\n")
	seed(t, repo, ".fabrik/docs/extra.md", "docs\n")
	seed(t, repo, ".fabrik/specs/001.md", "spec\n")
	seed(t, repo, "AGENTS.md", "# user root agents\n")
	seed(t, repo, "opencode.json", "{}\n")
	return repo
}

func TestPlanListsGeneratedOnly(t *testing.T) {
	repo := seedRepo(t)
	got, err := Plan(repo)
	require.NoError(t, err)
	set := map[string]bool{}
	for _, p := range got {
		rel, _ := filepath.Rel(repo, p)
		set[filepath.ToSlash(rel)] = true
	}
	joined := ""
	for k := range set {
		joined += k + "\n"
	}
	for _, want := range []string{
		".fabrik/PROMPT_plan.md", ".fabrik/loop.sh", ".fabrik/fabrik.sh",
		".fabrik/setup.sh", ".fabrik/state.md", ".fabrik/README.md",
		".fabrik/CONTEXT.md", ".fabrik/skills.md", ".fabrik/AGENTS.md",
		".fabrik/styleguide/STYLEGUIDE.md", ".fabrik/agents/builder.md",
		".cursor/rules/tdd.mdc", ".opencode/skills/tdd.md",
		".pi/skills/tdd.md", ".pi/agent/agents/builder.md",
		".cursor/rules/agents/builder.mdc", ".cursor/agents/builder.md",
	} {
		assert.True(t, set[want], "plan must include generated file %s (got:\n%s)", want, joined)
	}
	for _, keep := range []string{
		".fabrik/config.yaml", ".fabrik/docs/PRD.md", ".fabrik/specs/001.md",
		"AGENTS.md", "opencode.json",
		".cursor/rules/ponytail.mdc", ".opencode/skills/ponytail.md",
		".pi/skills/ponytail.md", ".pi/agent/agents/planner.md",
		".cursor/agents/planner.md",
	} {
		assert.False(t, set[keep], "plan must not include user/shim file %s (got:\n%s)", keep, joined)
	}
}

func TestDryRunTouchesNothing(t *testing.T) {
	repo := seedRepo(t)
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true, DryRun: true}, Deps{Out: &out}))
	assert.Contains(t, out.String(), "would delete")
	// generated file still on disk after dry-run
	assert.FileExists(t, filepath.Join(repo, ".fabrik", "README.md"))
	assert.FileExists(t, filepath.Join(repo, ".cursor", "rules", "tdd.mdc"))
}

func TestPruneDeletesGeneratedOnly(t *testing.T) {
	repo := seedRepo(t)
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true}, Deps{Out: &out}))
	for _, gone := range []string{
		".fabrik/PROMPT_plan.md", ".fabrik/loop.sh", ".fabrik/README.md",
		".fabrik/CONTEXT.md", ".fabrik/skills.md", ".fabrik/AGENTS.md",
		".fabrik/styleguide/STYLEGUIDE.md", ".fabrik/agents/builder.md",
		".cursor/rules/tdd.mdc", ".opencode/skills/tdd.md",
		".pi/skills/tdd.md", ".pi/agent/agents/builder.md",
	} {
		assert.NoFileExists(t, filepath.Join(repo, filepath.FromSlash(gone)), gone)
	}
	// user files untouched
	for _, keep := range []string{
		".fabrik/config.yaml", ".fabrik/docs/PRD.md", ".fabrik/specs/001.md",
		"AGENTS.md", "opencode.json",
		".cursor/rules/ponytail.mdc", ".pi/agent/agents/planner.md",
	} {
		assert.FileExists(t, filepath.Join(repo, filepath.FromSlash(keep)), keep)
	}
}

// cli-s34: init output (embedded opencode.json template) followed by
// migrate --prune must leave zero dangling {file:} refs. Seeds the real
// template plus the full bodies the old refs pointed at, prunes, then
// resolves every remaining ref against disk.
func TestInitTemplatePruneLeavesNoDanglingFileRefs(t *testing.T) {
	repo := t.TempDir()
	seed(t, repo, "opencode.json", tmpl.MustRead("opencode.json"))
	for _, a := range []string{"orchestrator", "builder"} {
		body, err := tmpl.Read("agents/" + a + ".md")
		require.NoError(t, err)
		seed(t, repo, ".fabrik/agents/"+a+".md", body)
	}
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true}, Deps{Out: &out}))
	assert.FileExists(t, filepath.Join(repo, "opencode.json"), "opencode.json is user config, never pruned")
	raw, err := os.ReadFile(filepath.Join(repo, "opencode.json"))
	require.NoError(t, err)
	// explicit non-vacuous guard: the template must carry zero {file:} refs
	// (a pure FindAll loop would pass vacuously when no refs exist).
	assert.NotContains(t, string(raw), "{file:")
	assert.Empty(t, fileRefRe.FindAllString(string(raw), -1), "opencode.json must carry zero {file:} refs")
	for _, m := range fileRefRe.FindAllStringSubmatch(string(raw), -1) {
		target := filepath.Join(repo, filepath.FromSlash(m[1]))
		assert.FileExists(t, target, "dangling ref %s after prune", m[0])
	}
}

func TestPruneRequiresFlag(t *testing.T) {
	var out bytes.Buffer
	err := Migrate(Config{RepoPath: t.TempDir()}, Deps{Out: &out})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--prune")
}

func TestPlanGitignoreNestedAndFalseShim(t *testing.T) {
	repo := t.TempDir()
	full := "FULL BODY generated dump without any shim marker\n"
	// generated .gitignore from init --full
	seed(t, repo, ".fabrik/.gitignore", full)
	// nested skill copies (recursive dirs)
	seed(t, repo, ".opencode/skills/nested/deep.md", full)
	seed(t, repo, ".opencode/agents/stray.md", full)
	seed(t, repo, ".pi/skills/team/custom.md", full)
	seed(t, repo, ".cursor/rules/extra/nested.mdc", full)
	// Cursor .md (non-mdc) full body
	seed(t, repo, ".cursor/rules/tdd.md", full)
	// false shim: mentions the words but is not an exact pointer marker
	seed(t, repo, ".opencode/skills/false.md", "notes about fabrik show skillset ideas\n")
	// true nested shim must be kept
	seed(t, repo, ".pi/skills/nested/keep.md", "# Fabrik skill tdd\n\nRun: fabrik show skill tdd\n")

	got, err := Plan(repo)
	require.NoError(t, err)
	set := map[string]bool{}
	for _, p := range got {
		rel, _ := filepath.Rel(repo, p)
		set[filepath.ToSlash(rel)] = true
	}
	for _, want := range []string{
		".fabrik/.gitignore",
		".opencode/skills/nested/deep.md",
		".opencode/agents/stray.md",
		".pi/skills/team/custom.md",
		".cursor/rules/extra/nested.mdc",
		".cursor/rules/tdd.md",
		".opencode/skills/false.md",
	} {
		assert.True(t, set[want], "plan must include %s (got %v)", want, set)
	}
	assert.False(t, set[".pi/skills/nested/keep.md"], "exact nested shim must be kept (got %v)", set)
}
