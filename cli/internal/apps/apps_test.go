package apps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllApps(t *testing.T) {
	assert.Contains(t, All, Cursor)
	assert.Contains(t, All, Pi)
	assert.Contains(t, All, Antigravity)
	assert.Contains(t, All, OpenCode)
	assert.Len(t, All, 4)
}

func TestNormalizeEmptyGivesAll(t *testing.T) {
	res, err := Normalize(nil)
	require.NoError(t, err)
	assert.Equal(t, All, res)

	res2, err := Normalize([]string{})
	require.NoError(t, err)
	assert.Equal(t, All, res2)
}

func TestNormalizeValid(t *testing.T) {
	res, err := Normalize([]string{"cursor", "pi"})
	require.NoError(t, err)
	assert.Equal(t, []string{"cursor", "pi"}, res)

	// Comma separated
	res2, err := Normalize([]string{"cursor,antigravity", "opencode"})
	require.NoError(t, err)
	assert.Equal(t, []string{"cursor", "antigravity", "opencode"}, res2)

	// With spaces & casing
	res3, err := Normalize([]string{"  Cursor , Pi  "})
	require.NoError(t, err)
	assert.Equal(t, []string{"cursor", "pi"}, res3)

	// "all" keyword
	res4, err := Normalize([]string{"all"})
	require.NoError(t, err)
	assert.Equal(t, All, res4)
}

func TestNormalizeInvalid(t *testing.T) {
	_, err := Normalize([]string{"cursor", "unknown_app"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid app \"unknown_app\"")
}

func TestContains(t *testing.T) {
	list := []string{"cursor", "antigravity"}
	assert.True(t, Contains(list, "cursor"))
	assert.True(t, Contains(list, "antigravity"))
	assert.True(t, Contains(list, "Antigravity"))
	assert.False(t, Contains(list, "pi"))
	assert.False(t, Contains(list, "opencode"))
}

func TestFormatList(t *testing.T) {
	assert.Equal(t, "Cursor, Antigravity", FormatList([]string{"cursor", "antigravity"}))
}
