package mcpconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterCursorMergesAndRemovesEngram(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	repo := t.TempDir()

	cursorPath := filepath.Join(home, ".cursor", "mcp.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(cursorPath), 0o755))
	require.NoError(t, os.WriteFile(cursorPath, []byte(`{"mcpServers":{"engram":{"command":"engram","args":["mcp"]}}}`), 0o644))

	require.NoError(t, Register([]string{"cursor"}, repo))

	var root map[string]any
	require.NoError(t, json.Unmarshal(mustRead(cursorPath), &root))
	servers := root["mcpServers"].(map[string]any)
	assert.NotContains(t, servers, "engram")
	fabrik := servers["fabrik"].(map[string]any)
	assert.Equal(t, "fabrik", fabrik["command"])
	assert.Equal(t, []any{"mcp"}, fabrik["args"])

	projectPath := filepath.Join(repo, ".cursor", "mcp.json")
	assert.FileExists(t, projectPath)
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}
