package mcpserver

import (
	"context"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/graph"
	"github.com/ur-wesley/fabrik/cli/internal/memory"
)

type memContextIn struct {
	Project *string `json:"project,omitempty"`
	Limit   *int    `json:"limit,omitempty"`
}

type memSearchIn struct {
	Query   string  `json:"query"`
	Limit   *int    `json:"limit,omitempty"`
	Project *string `json:"project,omitempty"`
}

type memSaveIn struct {
	Title         string  `json:"title"`
	Content       string  `json:"content"`
	Kind          *string `json:"kind,omitempty"`
	TopicKey      *string `json:"topic_key,omitempty"`
	CapturePrompt *string `json:"capture_prompt,omitempty"`
	SymbolID      *string `json:"symbol_id,omitempty"`
}

type memUpdateIn struct {
	ID       string  `json:"id"`
	Title    *string `json:"title,omitempty"`
	Content  *string `json:"content,omitempty"`
	Kind     *string `json:"kind,omitempty"`
	TopicKey *string `json:"topic_key,omitempty"`
	SymbolID *string `json:"symbol_id,omitempty"`
}

type memDeleteIn struct {
	ID string `json:"id"`
}

type memTimelineIn struct {
	ID      string  `json:"id"`
	Limit   *int    `json:"limit,omitempty"`
	Project *string `json:"project,omitempty"`
}

type memSavePromptIn struct {
	Title         string  `json:"title"`
	CapturePrompt string  `json:"capture_prompt"`
}

type memSuggestTopicKeyIn struct {
	Title string `json:"title"`
}

type memSessionSummaryIn struct {
	Goal         *string `json:"goal"`
	Discoveries  *string `json:"discoveries"`
	Accomplished *string `json:"accomplished"`
	NextSteps    *string `json:"next_steps"`
	Files        *string `json:"files"`
}

type memGetIn struct {
	ID string `json:"id" jsonschema:"required observation id o-…"`
}

type projectPathIn struct {
	ProjectPath *string `json:"project_path,omitempty"`
}

type graphExploreIn struct {
	Query       string  `json:"query"`
	IssueID     *string `json:"issue_id,omitempty"`
	Budget      *int    `json:"budget,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type graphSymbolIn struct {
	Symbol      string  `json:"symbol"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type graphImpactIn struct {
	Symbol      string  `json:"symbol"`
	Depth       *int    `json:"depth,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type graphDetectIn struct {
	Ref         *string `json:"ref,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type graphPathIn struct {
	From        string  `json:"from"`
	To          string  `json:"to"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueShowIn struct {
	ID          string  `json:"id"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueClaimIn struct {
	ID          string  `json:"id,omitempty"`
	Actor       *string `json:"actor,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueCloseIn struct {
	ID          string  `json:"id"`
	Reason      *string `json:"reason,omitempty"`
	Force       *bool   `json:"force,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueListIn struct {
	Status      *string `json:"status,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueReadyIn struct {
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueCreateIn struct {
	Title       string  `json:"title"`
	Body        *string `json:"body,omitempty"`
	Acceptance  *string `json:"acceptance,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	Kind        *string `json:"kind,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueDepIn struct {
	BlockedID   string  `json:"blocked_id"`
	BlockerID   string  `json:"blocker_id"`
	Kind        *string `json:"kind,omitempty"`
	ProjectPath *string `json:"project_path,omitempty"`
}

type issueNoteIn struct {
	IssueID     string  `json:"issue_id"`
	Body        string  `json:"body"`
	ProjectPath *string `json:"project_path,omitempty"`
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func intVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func handleMemContext(ctx context.Context, req *mcp.CallToolRequest, in memContextIn) (*mcp.CallToolResult, memory.ContextResult, error) {
	wd, _ := os.Getwd()
	r, err := memory.Context(ctx, memory.ContextOptions{
		Project: strVal(in.Project), Limit: intVal(in.Limit),
		AgentDefault: strVal(in.Project) == "", Cwd: wd, Actor: actorFrom(req),
	})
	if err != nil {
		return nil, memory.ContextResult{}, err
	}
	return nil, r, nil
}

func handleMemSearch(ctx context.Context, _ *mcp.CallToolRequest, in memSearchIn) (*mcp.CallToolResult, []memory.Observation, error) {
	obs, err := memory.Search(ctx, memory.SearchOptions{
		Query: in.Query, Limit: intVal(in.Limit), Project: strVal(in.Project),
	})
	if err != nil {
		return nil, nil, err
	}
	if obs == nil {
		obs = []memory.Observation{}
	}
	return nil, obs, nil
}

func handleMemSave(ctx context.Context, _ *mcp.CallToolRequest, in memSaveIn) (*mcp.CallToolResult, memory.Observation, error) {
	wd, _ := os.Getwd()
	o, err := memory.Save(ctx, memory.SaveOptions{
		Title: in.Title, Body: in.Content, Kind: strVal(in.Kind),
		TopicKey: strVal(in.TopicKey), CapturePrompt: strVal(in.CapturePrompt),
		SymbolID: strVal(in.SymbolID), Cwd: wd,
	})
	if err != nil {
		return nil, memory.Observation{}, err
	}
	return nil, o, nil
}

func handleMemSessionSummary(ctx context.Context, req *mcp.CallToolRequest, in memSessionSummaryIn) (*mcp.CallToolResult, memory.Session, error) {
	wd, _ := os.Getwd()
	opt := memory.SessionSummaryOptions{Cwd: wd, Actor: actorFrom(req)}
	if in.Goal != nil {
		opt.GoalSet = true
		opt.Goal = *in.Goal
	}
	if in.Discoveries != nil {
		opt.DiscoveriesSet = true
		opt.Discoveries = *in.Discoveries
	}
	if in.Accomplished != nil {
		opt.AccomplishedSet = true
		opt.Accomplished = *in.Accomplished
	}
	if in.NextSteps != nil {
		opt.NextStepsSet = true
		opt.NextSteps = *in.NextSteps
	}
	if in.Files != nil {
		opt.FilesSet = true
		opt.Files = *in.Files
	}
	s, err := memory.SessionSummary(ctx, opt)
	if err != nil {
		return nil, memory.Session{}, err
	}
	return nil, s, nil
}

func handleMemGetObservation(ctx context.Context, _ *mcp.CallToolRequest, in memGetIn) (*mcp.CallToolResult, memory.Observation, error) {
	o, err := memory.Get(ctx, in.ID)
	if err != nil {
		return nil, memory.Observation{}, err
	}
	return nil, o, nil
}

func handleMemSessionStart(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, memory.SessionStartResult, error) {
	r, err := memory.SessionStart(ctx, actorFrom(req))
	if err != nil {
		return nil, memory.SessionStartResult{}, err
	}
	return nil, r, nil
}

func handleMemSessionEnd(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, memory.Session, error) {
	s, err := memory.SessionEnd(ctx, actorFrom(req))
	if err != nil {
		return nil, memory.Session{}, err
	}
	return nil, s, nil
}

type currentProjectOut struct {
	Project string `json:"project"`
	Source  string `json:"source"`
}

func handleMemCurrentProject(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, currentProjectOut, error) {
	wd, _ := os.Getwd()
	r := memory.CurrentProject(wd)
	return nil, currentProjectOut{Project: r.Project, Source: r.Source}, nil
}

func handleMemStats(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, memory.ContextStats, error) {
	st, err := memory.Stats(ctx)
	if err != nil {
		return nil, memory.ContextStats{}, err
	}
	return nil, st, nil
}

func handleMemUpdate(ctx context.Context, _ *mcp.CallToolRequest, in memUpdateIn) (*mcp.CallToolResult, memory.Observation, error) {
	opt := memory.UpdateOptions{ID: in.ID}
	if in.Title != nil {
		opt.Title = in.Title
	}
	if in.Content != nil {
		opt.Body = in.Content
	}
	if in.Kind != nil {
		opt.Kind = in.Kind
	}
	if in.TopicKey != nil {
		opt.TopicKey = in.TopicKey
	}
	if in.SymbolID != nil {
		opt.SymbolID = in.SymbolID
	}
	o, err := memory.Update(ctx, opt)
	if err != nil {
		return nil, memory.Observation{}, err
	}
	return nil, o, nil
}

func handleMemDelete(ctx context.Context, _ *mcp.CallToolRequest, in memDeleteIn) (*mcp.CallToolResult, map[string]string, error) {
	if err := memory.Delete(ctx, in.ID); err != nil {
		return nil, nil, err
	}
	return nil, map[string]string{"id": in.ID, "deleted": "true"}, nil
}

func handleMemTimeline(ctx context.Context, _ *mcp.CallToolRequest, in memTimelineIn) (*mcp.CallToolResult, memory.TimelineResult, error) {
	r, err := memory.Timeline(ctx, memory.TimelineOptions{
		ID: in.ID, Limit: intVal(in.Limit), Project: strVal(in.Project),
	})
	if err != nil {
		return nil, memory.TimelineResult{}, err
	}
	return nil, r, nil
}

func handleMemSavePrompt(ctx context.Context, _ *mcp.CallToolRequest, in memSavePromptIn) (*mcp.CallToolResult, memory.Observation, error) {
	wd, _ := os.Getwd()
	o, err := memory.SavePrompt(ctx, memory.SavePromptOptions{
		Title: in.Title, CapturePrompt: in.CapturePrompt, Cwd: wd,
	})
	if err != nil {
		return nil, memory.Observation{}, err
	}
	return nil, o, nil
}

func handleMemSuggestTopicKey(_ context.Context, _ *mcp.CallToolRequest, in memSuggestTopicKeyIn) (*mcp.CallToolResult, map[string]string, error) {
	key := memory.SuggestTopicKey(memory.SuggestTopicKeyOptions{Title: in.Title})
	return nil, map[string]string{"topic_key": key}, nil
}

func handleIssueReady(ctx context.Context, _ *mcp.CallToolRequest, in issueReadyIn) (*mcp.CallToolResult, []doltstore.IssueSlim, error) {
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer st.Close()
	issues, err := st.Ready(ctx)
	if err != nil {
		return nil, nil, mapStoreErr(err)
	}
	if issues == nil {
		return nil, []doltstore.IssueSlim{}, nil
	}
	return nil, doltstore.SlimIssues(issues), nil
}

func handleIssueShow(ctx context.Context, _ *mcp.CallToolRequest, in issueShowIn) (*mcp.CallToolResult, doltstore.Issue, error) {
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, doltstore.Issue{}, err
	}
	defer st.Close()
	iss, err := st.Show(ctx, in.ID)
	if err != nil {
		return nil, doltstore.Issue{}, mapStoreErr(err)
	}
	return nil, iss, nil
}

func handleIssueClaim(ctx context.Context, req *mcp.CallToolRequest, in issueClaimIn) (*mcp.CallToolResult, doltstore.Issue, error) {
	by := strVal(in.Actor)
	if by == "" {
		by = actorFrom(req)
	}
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, doltstore.Issue{}, err
	}
	defer st.Close()
	iss, err := st.Claim(ctx, in.ID, by)
	if err != nil {
		return nil, doltstore.Issue{}, mapStoreErr(err)
	}
	if err := afterIssueWrite(ctx, st, iss.ID, true); err != nil {
		return nil, doltstore.Issue{}, err
	}
	return nil, iss, nil
}

func handleIssueClose(ctx context.Context, req *mcp.CallToolRequest, in issueCloseIn) (*mcp.CallToolResult, doltstore.Issue, error) {
	force := in.Force != nil && *in.Force
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, doltstore.Issue{}, err
	}
	defer st.Close()
	res, err := st.CloseIssueBy(ctx, in.ID, strVal(in.Reason), actorFrom(req), force)
	if err != nil {
		return nil, doltstore.Issue{}, mapStoreErr(err)
	}
	if err := afterIssueWrite(ctx, st, res.Issue.ID, true); err != nil {
		return nil, doltstore.Issue{}, err
	}
	return nil, res.Issue, nil
}

func handleIssueList(ctx context.Context, _ *mcp.CallToolRequest, in issueListIn) (*mcp.CallToolResult, []doltstore.IssueSlim, error) {
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer st.Close()
	issues, err := st.List(ctx, strVal(in.Status))
	if err != nil {
		return nil, nil, mapStoreErr(err)
	}
	if issues == nil {
		return nil, []doltstore.IssueSlim{}, nil
	}
	return nil, doltstore.SlimIssues(issues), nil
}

func handleIssueCreate(ctx context.Context, _ *mcp.CallToolRequest, in issueCreateIn) (*mcp.CallToolResult, doltstore.Issue, error) {
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, doltstore.Issue{}, err
	}
	defer st.Close()
	iss, err := st.Create(ctx, doltstore.CreateInput{
		Title: in.Title, Body: strVal(in.Body), Acceptance: strVal(in.Acceptance),
		Priority: in.Priority, Kind: strVal(in.Kind),
	})
	if err != nil {
		return nil, doltstore.Issue{}, mapStoreErr(err)
	}
	if err := afterIssueWrite(ctx, st, iss.ID, true); err != nil {
		return nil, doltstore.Issue{}, err
	}
	return nil, iss, nil
}

func handleIssueDep(ctx context.Context, _ *mcp.CallToolRequest, in issueDepIn) (*mcp.CallToolResult, doltstore.DepResult, error) {
	kind := strVal(in.Kind)
	if kind == "" {
		kind = "blocks"
	}
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, doltstore.DepResult{}, err
	}
	defer st.Close()
	res, err := st.Dep(ctx, in.BlockedID, in.BlockerID, kind)
	if err != nil {
		return nil, doltstore.DepResult{}, mapStoreErr(err)
	}
	if err := afterIssueWrite(ctx, st, res.Issue.ID, !res.AlreadyExists); err != nil {
		return nil, doltstore.DepResult{}, err
	}
	return nil, res, nil
}

func handleIssueNote(ctx context.Context, _ *mcp.CallToolRequest, in issueNoteIn) (*mcp.CallToolResult, doltstore.Note, error) {
	st, err := openIssueStore(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, doltstore.Note{}, err
	}
	defer st.Close()
	note, err := st.Note(ctx, in.IssueID, in.Body)
	if err != nil {
		return nil, doltstore.Note{}, mapStoreErr(err)
	}
	if err := afterIssueWrite(ctx, st, in.IssueID, true); err != nil {
		return nil, doltstore.Note{}, err
	}
	return nil, note, nil
}

func handleGraphExplore(ctx context.Context, _ *mcp.CallToolRequest, in graphExploreIn) (*mcp.CallToolResult, any, error) {
	db, err := openGraphDB(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	opts := graph.ExploreOpts{Query: in.Query, Budget: intVal(in.Budget)}
	if in.IssueID != nil && *in.IssueID != "" {
		st, err := openIssueStore(ctx, strVal(in.ProjectPath))
		if err != nil {
			return nil, nil, err
		}
		defer st.Close()
		iss, err := st.Show(ctx, *in.IssueID)
		if err != nil {
			return nil, nil, mapStoreErr(err)
		}
		opts.IssueSeedText = doltstore.IssueExploreSeedText(iss)
	}
	r, err := db.Explore(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	return nil, r, nil
}

func handleGraphContext(ctx context.Context, _ *mcp.CallToolRequest, in graphSymbolIn) (*mcp.CallToolResult, any, error) {
	db, err := openGraphDB(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	r, err := db.Context(ctx, in.Symbol)
	if err != nil {
		return nil, nil, err
	}
	return nil, r, nil
}

func handleGraphImpact(ctx context.Context, _ *mcp.CallToolRequest, in graphImpactIn) (*mcp.CallToolResult, any, error) {
	db, err := openGraphDB(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	r, err := db.Impact(ctx, in.Symbol, intVal(in.Depth))
	if err != nil {
		return nil, nil, err
	}
	return nil, r, nil
}

func handleGraphDetectChanges(ctx context.Context, _ *mcp.CallToolRequest, in graphDetectIn) (*mcp.CallToolResult, any, error) {
	db, err := openGraphDB(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	r, err := db.DetectChanges(ctx, strVal(in.Ref))
	if err != nil {
		return nil, nil, err
	}
	return nil, r, nil
}

func handleGraphPath(ctx context.Context, _ *mcp.CallToolRequest, in graphPathIn) (*mcp.CallToolResult, any, error) {
	db, err := openGraphDB(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	r, err := db.Path(ctx, in.From, in.To)
	if err != nil {
		return nil, nil, err
	}
	return nil, r, nil
}

func handleGraphExplain(ctx context.Context, _ *mcp.CallToolRequest, in graphSymbolIn) (*mcp.CallToolResult, any, error) {
	db, err := openGraphDB(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	r, err := db.Explain(ctx, in.Symbol)
	if err != nil {
		return nil, nil, err
	}
	return nil, r, nil
}

func handleGraphStatus(ctx context.Context, _ *mcp.CallToolRequest, in projectPathIn) (*mcp.CallToolResult, any, error) {
	db, err := openGraphDB(ctx, strVal(in.ProjectPath))
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	r, err := db.Status(ctx)
	if err != nil {
		return nil, nil, err
	}
	return nil, r, nil
}

func actorFrom(req *mcp.CallToolRequest) string {
	if req != nil && req.Session != nil {
		if p := req.Session.InitializeParams(); p != nil && p.ClientInfo != nil && p.ClientInfo.Name != "" {
			return p.ClientInfo.Name
		}
	}
	return "agent"
}
