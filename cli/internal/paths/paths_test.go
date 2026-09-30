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
	b, err := BinDir()
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(b, filepath.Join(".local", "bin")), b)
}

func TestGraphifySkillsThreeAppsOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	dests, err := GraphifySkills()
	require.NoError(t, err)
	require.Len(t, dests, 3)
	joined := strings.Join(dests, "\n")
	assert.Contains(t, joined, ".cursor")
	assert.Contains(t, joined, "opencode")
	assert.Contains(t, joined, ".pi")
	assert.NotContains(t, joined, ".claude")
	assert.NotContains(t, joined, ".codex")
	assert.NotContains(t, joined, ".agents")
}

func TestWorkflowNoteDests(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	dests, err := WorkflowNoteDests()
	require.NoError(t, err)
	require.Len(t, dests, 3)
	assert.True(t, dests[0].Frontmatter)
	assert.False(t, dests[1].Frontmatter)
}
