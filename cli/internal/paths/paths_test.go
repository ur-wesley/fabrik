package paths

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinDirUnderHome(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	b, err := BinDir()
	require.NoError(t, err)
	assert.True(t,
		strings.HasSuffix(b, filepath.Join(".local", "bin")) ||
			strings.HasSuffix(b, filepath.Join("fabrik", "bin")), b)
}

func TestOnPATH(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	assert.True(t, OnPATH(dir))
	assert.False(t, OnPATH(filepath.Join(dir, "nope")))
	assert.Empty(t, WarnIfNotOnPATH(dir))
	assert.NotEmpty(t, WarnIfNotOnPATH(filepath.Join(dir, "nope")))
}

func TestGraphifySkillsAllApps(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	dests, err := GraphifySkills()
	require.NoError(t, err)
	require.Len(t, dests, 5)
	joined := strings.Join(dests, "\n")
	assert.Contains(t, joined, ".cursor")
	assert.Contains(t, joined, "opencode")
	assert.Contains(t, joined, ".pi")
	assert.Contains(t, joined, "antigravity")
	assert.Contains(t, joined, filepath.Join(".gemini", "skills", "graphify", "SKILL.md"))
	assert.NotContains(t, joined, ".claude")
	assert.NotContains(t, joined, ".codex")
}

func TestGraphifySkillsFiltered(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	dests, err := GraphifySkills("cursor", "antigravity")
	require.NoError(t, err)
	require.Len(t, dests, 2)
	joined := strings.Join(dests, "\n")
	assert.Contains(t, joined, ".cursor")
	assert.Contains(t, joined, "antigravity")
	assert.NotContains(t, joined, "opencode")
	assert.NotContains(t, joined, ".pi")
}

func TestWorkflowNoteDests(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	dests, err := WorkflowNoteDests()
	require.NoError(t, err)
	require.Len(t, dests, 2)
	assert.True(t, dests[0].Frontmatter)
	assert.False(t, dests[1].Frontmatter)
	for _, d := range dests {
		assert.NotContains(t, d.Path, "opencode")
	}

	filtered, err := WorkflowNoteDests("cursor")
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.True(t, filtered[0].Frontmatter)
}
