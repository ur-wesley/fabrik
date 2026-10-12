package mcpserver
import ("context";"os";"path/filepath";"testing";"github.com/modelcontextprotocol/go-sdk/mcp")
func TestDebugGraph(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("FABRIK_MCP_TOOLS", "graph_explore")
	os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755)
	prev, _ := os.Getwd(); os.Chdir(dir); defer os.Chdir(prev)
	srv := NewServer(); cs := connectTestClient(t, srv)
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name:"graph_explore", Arguments: map[string]any{"query":"main"}})
	t.Logf("err=%v isErr=%v text=%s", err, res.IsError, toolResultJSON(t,res))
}
