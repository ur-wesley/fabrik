package tmpl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot locates the fabrik checkout root when tests run inside it.
func repoRoot(t *testing.T) string {
	t.Helper()
	// cli/internal/tmpl -> cli -> fabrik
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(root, "install", "deps.json")); err != nil {
		t.Skipf("not inside fabrik checkout: %v", err)
	}
	return root
}

func TestExpectedFilesEmbedded(t *testing.T) {
	for _, p := range []string{
		"workflow-note.md",
		"fabrik/README.md",
		"fabrik/CONTEXT.md",
		"fabrik/config.yaml",
		"fabrik/skills.md",
		"fabrik/gitignore",
		"agents/plan.md",
		"agents/orchestrator.md",
		"agents/build.md",
		"agents/explore.md",
		"agents/researcher.md",
		"agents/briefer.md",
		"agents/planner.md",
		"agents/builder.md",
		"agents/tester.md",
		"agents/reviewer.md",
		"agents/style-smells.md",
		"agents/security.md",
		"agents/backlog.md",
		"agents/cta.md",
		"hub/PROMPT_plan.md",
		"hub/PROMPT_build.md",
		"hub/PROMPT_backlog.md",
		"hub/STYLEGUIDE.md",
		"hub/AGENTS.md",
	} {
		_, err := Read(p)
		assert.NoError(t, err, p)
	}
	// skills dir embeds the repo skills wholesale
	assert.NotEmpty(t, List("skills"))
}

// Every shipped subagent must end with a Done/Next footer (i-have-adhd
// style) so orchestrated waves report uniformly (issue cli-5zc).
func TestAgentDefsHaveDoneNext(t *testing.T) {
	for _, p := range []string{
		"agents/reviewer.md",
		"agents/security.md",
		"agents/style-smells.md",
	} {
		assert.Contains(t, MustRead(p), "Done/Next", p)
	}
}

// opencode.json must never carry {file:.fabrik/agents/...} refs: thin init
// and migrate --prune drop that dir, so such refs would dangle (cli-s34).
// Canonical is the show-shim (consistent with thin agent shims).
func TestPromptPlanNoBdCreate(t *testing.T) {
	body := MustRead("hub/PROMPT_plan.md")
	assert.NotContains(t, body, "USE `bd create`")
	assert.NotContains(t, body, "NEVER ASK FOR PERMISSION")
	assert.Contains(t, body, "Do NOT `bd create`")
}

// Backlog gate: no bd create before APPROVE; post-APPROVE lands epic +
// p3/p4 tasks with acceptance (cli-6aj). Mirrors TestPromptPlanNoBdCreate.
func TestPromptBacklogGate(t *testing.T) {
	for _, p := range []string{"hub/PROMPT_backlog.md", "skills/to-backlog.md", "agents/backlog.md"} {
		body := MustRead(p)
		assert.Contains(t, body, "Do NOT `bd create`", p)
		assert.Contains(t, body, "APPROVE", p)
		assert.Contains(t, body, "p3/p4", p)
		assert.Contains(t, body, "epic", p)
	}
}

func TestOpencodeJsonHasNoFileRefs(t *testing.T) {
	body := MustRead("opencode.json")
	assert.NotContains(t, body, "{file:")
	assert.Contains(t, body, "Run: fabrik show agent ")
	assert.Contains(t, body, `"default_agent": "plan"`)

	// Canonical is the embedded plan-default (default_agent plan + plan
	// agent + rewritten orchestrator/planner, matching workflow-note + embedded
	// agents). All live copies must be byte-identical to it.
	root := repoRoot(t)
	for _, live := range []string{
		"opencode.json",
		filepath.Join("cli", "opencode.json"),
		filepath.Join("install", "templates", "opencode.json"),
	} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(live)))
		require.NoError(t, err, live)
		assert.Equal(t, body, string(raw), "drift: %s — re-copy from embedded opencode.json", live)
	}
}

func TestInSyncWithRepo(t *testing.T) {
	root := repoRoot(t)
	pairs := map[string]string{
		"workflow-note.md":       "install/templates/workflow-note.md",
		"opencode.json":          "install/templates/opencode.json",
		"fabrik/README.md":       "install/templates/fabrik/README.md",
		"fabrik/CONTEXT.md":      "install/templates/fabrik/CONTEXT.md",
		"fabrik/config.yaml":     "install/templates/fabrik/config.yaml",
		"fabrik/skills.md":       "install/templates/fabrik/skills.md",
		"fabrik/gitignore":       "install/templates/fabrik/gitignore",
		"agents/plan.md":         "install/templates/agents/plan.md",
		"agents/orchestrator.md": "install/templates/agents/orchestrator.md",
		"agents/build.md":        "install/templates/agents/build.md",
		"agents/explore.md":      "install/templates/agents/explore.md",
		"agents/researcher.md":   "install/templates/agents/researcher.md",
		"agents/briefer.md":      "install/templates/agents/briefer.md",
		"agents/planner.md":      "install/templates/agents/planner.md",
		"agents/builder.md":      "install/templates/agents/builder.md",
		"agents/tester.md":       "install/templates/agents/tester.md",
		"agents/reviewer.md":     "install/templates/agents/reviewer.md",
		"agents/style-smells.md": "install/templates/agents/style-smells.md",
		"agents/security.md":     "install/templates/agents/security.md",
		"agents/backlog.md":      "install/templates/agents/backlog.md",
		"agents/cta.md":          "install/templates/agents/cta.md",
		"hub/PROMPT_plan.md":     "install/templates/hub/PROMPT_plan.md",
		"hub/PROMPT_build.md":    "install/templates/hub/PROMPT_build.md",
		"hub/PROMPT_backlog.md":  "install/templates/hub/PROMPT_backlog.md",
		"hub/STYLEGUIDE.md":      "install/templates/hub/STYLEGUIDE.md",
		"hub/AGENTS.md":          "install/templates/hub/AGENTS.md",
	}
	for emb, repo := range pairs {
		want, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repo)))
		require.NoError(t, err, repo)
		assert.Equal(t, string(want), MustRead(emb), "drift: %s — re-copy from %s", emb, repo)
	}
	// agents/plan.md + agents/cta.md sync: install/templates copies are
	// authoritative (covered in pairs above); repo .fabrik/agents mirrors are
	// legacy full bodies — when present they must match embedded, otherwise
	// the roster is embedded-only (thin init serves them via fabrik show).
	for _, a := range []string{"agents/plan.md", "agents/cta.md"} {
		legacy := filepath.Join(root, ".fabrik", filepath.FromSlash(a))
		if raw, err := os.ReadFile(legacy); err == nil {
			assert.Equal(t, MustRead(a), string(raw), "drift: .fabrik/%s", a)
		}
	}
	// every repo skill must be embedded
	entries, err := os.ReadDir(filepath.Join(root, "skills"))
	require.NoError(t, err)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		want, err := os.ReadFile(filepath.Join(root, "skills", e.Name()))
		require.NoError(t, err)
		assert.Equal(t, string(want), MustRead("skills/"+e.Name()), "drift: skills/%s", e.Name())
	}
}
