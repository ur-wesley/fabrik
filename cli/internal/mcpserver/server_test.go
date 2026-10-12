package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func connectTestClient(t *testing.T, srv *mcp.Server) *mcp.ClientSession {
	t.Helper()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(t.Context(), st, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ss.Close() })
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	cs, err := c.Connect(t.Context(), ct, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func listToolNames(t *testing.T, cs *mcp.ClientSession) []string {
	res, err := cs.ListTools(t.Context(), nil)
	require.NoError(t, err)
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

func TestDefaultToolNames(t *testing.T) {
	t.Setenv("FABRIK_MCP_TOOLS", "")
	want := slices.Sorted(slices.Values(defaultTools))
	got := slices.Sorted(slices.Values(EnabledToolNames()))
	assert.Equal(t, want, got)

	srv := NewServer()
	cs := connectTestClient(t, srv)
	assert.Equal(t, want, listToolNames(t, cs))
}

func TestAllToolsIncludesExtras(t *testing.T) {
	t.Setenv("FABRIK_MCP_TOOLS", "all")
	names := EnabledToolNames()
	for _, extra := range extraTools {
		assert.Contains(t, names, extra)
	}
	srv := NewServer()
	cs := connectTestClient(t, srv)
	listed := listToolNames(t, cs)
	for _, extra := range extraTools {
		assert.Contains(t, listed, extra)
	}
}

func TestMemSaveAndSearch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	t.Setenv("FABRIK_MCP_TOOLS", "mem_save,mem_search")

	srv := NewServer()
	cs := connectTestClient(t, srv)
	ctx := context.Background()

	saveRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name: "mem_save",
		Arguments: map[string]any{
			"title":   "mcp test",
			"content": "unique mcp token xyzzy",
			"kind":    "discovery",
		},
	})
	require.NoError(t, err)
	require.False(t, saveRes.IsError)

	searchRes, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "mem_search",
		Arguments: map[string]any{"query": "xyzzy"},
	})
	require.NoError(t, err)
	require.False(t, searchRes.IsError)
	body := toolResultJSON(t, searchRes)
	assert.Contains(t, body, "xyzzy")
}

func TestGraphExploreNotIndexed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MCP_TOOLS", "graph_explore")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(prev) })

	srv := NewServer()
	cs := connectTestClient(t, srv)
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph_explore",
		Arguments: map[string]any{"query": "main"},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var out struct {
		Indexed bool `json:"indexed"`
	}
	require.NoError(t, json.Unmarshal([]byte(toolResultJSON(t, res)), &out))
	assert.False(t, out.Indexed)
}

func toolResultJSON(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res.StructuredContent != nil {
		b, err := json.Marshal(res.StructuredContent)
		require.NoError(t, err)
		return string(b)
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("no tool result content")
	return ""
}
