package mcpserver

import (
	"context"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ur-wesley/fabrik/cli/internal/memory"
	"github.com/ur-wesley/fabrik/cli/internal/version"
)

const serverInstructions = `1. mem_current_project then mem_context (cwd project full bodies; other projects titles). mem_search is machine-wide.
2. Code questions: graph_explore first when .fabrik/graph.db exists; do not grep the whole repo first.
3. Work queue: issue_claim (omit id for claim-next) → issue_show → implement → issue_close (as assignee).
4. Hand off with mem_session_summary and mem_save for durable decisions.
5. Shared repo: human or script runs fabrik sync (or dolt pull / push); MCP does not auto-sync.`

func NewServer() *mcp.Server {
	impl := &mcp.Implementation{Name: "fabrik", Version: version.Get()}
	srv := mcp.NewServer(impl, &mcp.ServerOptions{Instructions: serverInstructions})
	registerTools(srv, EnabledToolNames())
	return srv
}

func Run(ctx context.Context) error {
	memory.EnableSharedPool(true)
	return runStdio(ctx, NewServer(), os.Stdin, os.Stdout)
}

func EnabledToolNames() []string {
	raw := strings.TrimSpace(os.Getenv("FABRIK_MCP_TOOLS"))
	if raw == "" {
		return append([]string(nil), defaultTools...)
	}
	if strings.EqualFold(raw, "all") {
		return allTools()
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if knownTools[name] {
			out = append(out, name)
		}
	}
	return out
}

func allTools() []string {
	names := append([]string(nil), defaultTools...)
	for _, n := range extraTools {
		names = append(names, n)
	}
	return names
}

var defaultTools = []string{
	"mem_current_project",
	"mem_context",
	"mem_search",
	"mem_get_observation",
	"mem_save",
	"mem_session_summary",
	"issue_ready",
	"issue_show",
	"issue_claim",
	"issue_close",
	"graph_explore",
}

var extraTools = []string{
	"mem_session_start",
	"mem_session_end",
	"mem_stats",
	"mem_update",
	"mem_delete",
	"mem_timeline",
	"mem_save_prompt",
	"mem_suggest_topic_key",
	"issue_list",
	"issue_create",
	"issue_dep",
	"issue_note",
	"graph_context",
	"graph_impact",
	"graph_detect_changes",
	"graph_path",
	"graph_explain",
	"graph_status",
}

var knownTools = map[string]bool{
	"mem_context": true, "mem_search": true, "mem_save": true, "mem_session_summary": true,
	"mem_get_observation": true, "mem_session_start": true, "mem_session_end": true,
	"mem_current_project": true, "mem_stats": true,
	"mem_update": true, "mem_delete": true, "mem_timeline": true,
	"mem_save_prompt": true, "mem_suggest_topic_key": true,
	"issue_ready": true, "issue_show": true, "issue_claim": true, "issue_close": true,
	"issue_list": true, "issue_create": true, "issue_dep": true, "issue_note": true,
	"graph_explore": true, "graph_context": true, "graph_impact": true,
	"graph_detect_changes": true, "graph_path": true, "graph_explain": true, "graph_status": true,
}
