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
		"agents/orchestrator.md",
		"agents/explore.md",
		"agents/researcher.md",
		"agents/briefer.md",
		"agents/planner.md",
		"agents/builder.md",
		"agents/tester.md",
		"agents/reviewer.md",
		"agents/style-smells.md",
		"agents/security.md",
		"hub/PROMPT_plan.md",
		"hub/PROMPT_build.md",
		"hub/STYLEGUIDE.md",
		"hub/loop.sh",
		"hub/loop.ps1",
		"hub/fabrik.sh",
		"hub/fabrik.ps1",
		"hub/AGENTS.md",
	} {
		_, err := Read(p)
		assert.NoError(t, err, p)
	}
	// skills dir embeds the repo skills wholesale
	assert.NotEmpty(t, List("skills"))
}

func TestInSyncWithRepo(t *testing.T) {
	root := repoRoot(t)
	pairs := map[string]string{
		"workflow-note.md":       "install/templates/workflow-note.md",
		"fabrik/README.md":       "install/templates/fabrik/README.md",
		"fabrik/CONTEXT.md":      "install/templates/fabrik/CONTEXT.md",
		"fabrik/config.yaml":     "install/templates/fabrik/config.yaml",
		"fabrik/skills.md":       "install/templates/fabrik/skills.md",
		"fabrik/gitignore":       "install/templates/fabrik/gitignore",
		"agents/orchestrator.md": "install/templates/agents/orchestrator.md",
		"agents/explore.md":      "install/templates/agents/explore.md",
		"agents/researcher.md":   "install/templates/agents/researcher.md",
		"agents/briefer.md":      "install/templates/agents/briefer.md",
		"agents/planner.md":      "install/templates/agents/planner.md",
		"agents/builder.md":      "install/templates/agents/builder.md",
		"agents/tester.md":       "install/templates/agents/tester.md",
		"agents/reviewer.md":     "install/templates/agents/reviewer.md",
		"agents/style-smells.md": "install/templates/agents/style-smells.md",
		"agents/security.md":     "install/templates/agents/security.md",
		"hub/PROMPT_plan.md":     ".fabrik/PROMPT_plan.md",
		"hub/PROMPT_build.md":    ".fabrik/PROMPT_build.md",
		"hub/STYLEGUIDE.md":      ".fabrik/styleguide/STYLEGUIDE.md",
		"hub/loop.sh":            ".fabrik/loop.sh",
		"hub/loop.ps1":           ".fabrik/loop.ps1",
		"hub/fabrik.sh":          ".fabrik/fabrik.sh",
		"hub/fabrik.ps1":         ".fabrik/fabrik.ps1",
		"hub/AGENTS.md":          ".fabrik/AGENTS.md",
	}
	for emb, repo := range pairs {
		want, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(repo)))
		require.NoError(t, err, repo)
		assert.Equal(t, string(want), MustRead(emb), "drift: %s — re-copy from %s", emb, repo)
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
