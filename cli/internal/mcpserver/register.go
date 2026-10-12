package mcpserver

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTools(s *mcp.Server, names []string) {
	enabled := map[string]bool{}
	for _, n := range names {
		enabled[n] = true
	}
	if enabled["mem_context"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_context", Description: "Load machine-wide memory context"}, handleMemContext)
	}
	if enabled["mem_search"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_search", Description: "FTS search observations"}, handleMemSearch)
	}
	if enabled["mem_save"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_save", Description: "Save an observation"}, handleMemSave)
	}
	if enabled["mem_session_summary"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_session_summary", Description: "Summarize open session"}, handleMemSessionSummary)
	}
	if enabled["mem_get_observation"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_get_observation", Description: "Get observation by id"}, handleMemGetObservation)
	}
	if enabled["mem_session_start"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_session_start", Description: "Start global memory session"}, handleMemSessionStart)
	}
	if enabled["mem_session_end"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_session_end", Description: "End open memory session"}, handleMemSessionEnd)
	}
	if enabled["mem_current_project"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_current_project", Description: "Detect project from cwd"}, handleMemCurrentProject)
	}
	if enabled["mem_stats"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_stats", Description: "Memory totals and project list"}, handleMemStats)
	}
	if enabled["mem_update"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_update", Description: "Update an observation"}, handleMemUpdate)
	}
	if enabled["mem_delete"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_delete", Description: "Soft-delete an observation"}, handleMemDelete)
	}
	if enabled["mem_timeline"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_timeline", Description: "Timeline around an observation"}, handleMemTimeline)
	}
	if enabled["mem_save_prompt"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_save_prompt", Description: "Save a prompt observation"}, handleMemSavePrompt)
	}
	if enabled["mem_suggest_topic_key"] {
		mcp.AddTool(s, &mcp.Tool{Name: "mem_suggest_topic_key", Description: "Suggest topic_key from title"}, handleMemSuggestTopicKey)
	}
	if enabled["issue_ready"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_ready", Description: "List ready issues"}, handleIssueReady)
	}
	if enabled["issue_show"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_show", Description: "Show one issue"}, handleIssueShow)
	}
	if enabled["issue_claim"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_claim", Description: "Claim an issue"}, handleIssueClaim)
	}
	if enabled["issue_close"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_close", Description: "Close an issue"}, handleIssueClose)
	}
	if enabled["issue_list"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_list", Description: "List issues"}, handleIssueList)
	}
	if enabled["issue_create"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_create", Description: "Create an issue"}, handleIssueCreate)
	}
	if enabled["issue_dep"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_dep", Description: "Add issue dependency"}, handleIssueDep)
	}
	if enabled["issue_note"] {
		mcp.AddTool(s, &mcp.Tool{Name: "issue_note", Description: "Add issue note"}, handleIssueNote)
	}
	if enabled["graph_explore"] {
		mcp.AddTool(s, &mcp.Tool{Name: "graph_explore", Description: "Explore code graph"}, handleGraphExplore)
	}
	if enabled["graph_context"] {
		mcp.AddTool(s, &mcp.Tool{Name: "graph_context", Description: "Symbol context with memory"}, handleGraphContext)
	}
	if enabled["graph_impact"] {
		mcp.AddTool(s, &mcp.Tool{Name: "graph_impact", Description: "Symbol blast radius"}, handleGraphImpact)
	}
	if enabled["graph_detect_changes"] {
		mcp.AddTool(s, &mcp.Tool{Name: "graph_detect_changes", Description: "Map git diff to symbols"}, handleGraphDetectChanges)
	}
	if enabled["graph_path"] {
		mcp.AddTool(s, &mcp.Tool{Name: "graph_path", Description: "Path between symbols"}, handleGraphPath)
	}
	if enabled["graph_explain"] {
		mcp.AddTool(s, &mcp.Tool{Name: "graph_explain", Description: "Explain a symbol"}, handleGraphExplain)
	}
	if enabled["graph_status"] {
		mcp.AddTool(s, &mcp.Tool{Name: "graph_status", Description: "Graph index status"}, handleGraphStatus)
	}
}
