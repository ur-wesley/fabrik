package issuestore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// No CGO gate here: the SQLite store works in every build, which is the
// point of cli-x3a (issue CRUD with no Dolt dependency).

func openTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	ctx := context.Background()
	st, err := Open(ctx, t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return st, ctx
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	st, ctx := openTestStore(t)
	_, err := st.Create(ctx, CreateInput{Title: "   ", Kind: "task"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestCreateDefaultPriority(t *testing.T) {
	st, ctx := openTestStore(t)
	iss, err := st.Create(ctx, CreateInput{Title: "Default priority", Kind: "task"})
	require.NoError(t, err)
	assert.Equal(t, 2, iss.Priority)
}

func TestOpenListCreateShow(t *testing.T) {
	st, ctx := openTestStore(t)

	list, err := st.List(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, list)

	p2 := 2
	iss, err := st.Create(ctx, CreateInput{
		Title: "Add issue store", Acceptance: "list works", Priority: &p2, Kind: "task",
	})
	require.NoError(t, err)
	assert.True(t, iss.Ready)
	assert.Regexp(t, `^f-[0-9a-f]{8}$`, iss.ID)

	list, err = st.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.True(t, list[0].Ready)

	shown, err := st.Show(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, iss.Title, shown.Title)
	assert.Empty(t, shown.Blockers)

	// Schema 001+002 applied: meta + seq column present.
	var ver string
	require.NoError(t, st.DB.QueryRowContext(ctx, `SELECT value FROM meta WHERE name='schema_version'`).Scan(&ver))
	assert.Equal(t, "2", ver)
	var seqName string
	require.NoError(t, st.DB.QueryRowContext(ctx,
		`SELECT name FROM pragma_table_info('issue_notes') WHERE name='seq'`).Scan(&seqName))
	assert.Equal(t, "seq", seqName)
}

func TestReadyClaimCloseDepNote(t *testing.T) {
	st, ctx := openTestStore(t)

	p2, p1 := 2, 1
	blocker, err := st.Create(ctx, CreateInput{Title: "Blocker", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	blocked, err := st.Create(ctx, CreateInput{Title: "Blocked", Priority: &p1, Kind: "bug"})
	require.NoError(t, err)

	ready, err := st.Ready(ctx)
	require.NoError(t, err)
	require.Len(t, ready, 2)

	dep, err := st.Dep(ctx, blocked.ID, blocker.ID, "blocks")
	require.NoError(t, err)
	assert.Equal(t, "blocked", dep.Issue.Status)
	assert.False(t, dep.Issue.Ready)

	ready, err = st.Ready(ctx)
	require.NoError(t, err)
	require.Len(t, ready, 1)
	assert.Equal(t, blocker.ID, ready[0].ID)

	_, err = st.Claim(ctx, blocked.ID, "agent")
	var blk *ErrIssueBlocked
	require.ErrorAs(t, err, &blk)
	assert.Equal(t, blocker.ID, blk.BlockerID)

	claimed, err := st.Claim(ctx, blocker.ID, "agent")
	require.NoError(t, err)
	assert.Equal(t, "active", claimed.Status)

	p3 := 3
	relates, err := st.Create(ctx, CreateInput{Title: "Related", Priority: &p3, Kind: "task"})
	require.NoError(t, err)
	_, err = st.Dep(ctx, relates.ID, blocked.ID, "relates")
	require.NoError(t, err)
	ready, err = st.Ready(ctx)
	require.NoError(t, err)
	require.Len(t, ready, 1)
	assert.Equal(t, relates.ID, ready[0].ID)

	res, err := st.CloseIssue(ctx, blocker.ID, "done")
	require.NoError(t, err)
	assert.Equal(t, "done", res.Issue.Status)
	assert.NotEmpty(t, res.NoteID)

	unblocked, err := st.Get(ctx, blocked.ID)
	require.NoError(t, err)
	assert.Equal(t, "open", unblocked.Status)
	assert.True(t, unblocked.Ready)

	note, err := st.Note(ctx, blocked.ID, "progress")
	require.NoError(t, err)
	assert.Regexp(t, `^n-[0-9a-f]{8}$`, note.ID)

	shown, err := st.Show(ctx, blocked.ID)
	require.NoError(t, err)
	require.Len(t, shown.Notes, 1)
	assert.Equal(t, "progress", shown.Notes[0].Body)

	blockerShown, err := st.Show(ctx, blocker.ID)
	require.NoError(t, err)
	require.Len(t, blockerShown.Notes, 1)
	assert.Equal(t, "done", blockerShown.Notes[0].Body)
}

func TestClaimUnclaimCycle(t *testing.T) {
	st, ctx := openTestStore(t)
	p2 := 2
	iss, err := st.Create(ctx, CreateInput{Title: "Cycle", Priority: &p2, Kind: "task"})
	require.NoError(t, err)

	claimed, err := st.Claim(ctx, iss.ID, "agent")
	require.NoError(t, err)
	assert.Equal(t, "active", claimed.Status)

	_, err = st.Unclaim(ctx, iss.ID, "other", false)
	var notAssignee *ErrNotAssignee
	require.ErrorAs(t, err, &notAssignee)

	back, err := st.Unclaim(ctx, iss.ID, "agent", false)
	require.NoError(t, err)
	assert.Equal(t, "open", back.Status)
	assert.True(t, back.Ready)
}

func TestCloseByAssignee(t *testing.T) {
	st, ctx := openTestStore(t)
	p2 := 2
	iss, err := st.Create(ctx, CreateInput{Title: "Guard", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	_, err = st.Claim(ctx, iss.ID, "alice")
	require.NoError(t, err)
	_, err = st.CloseIssueBy(ctx, iss.ID, "x", "bob", false)
	require.Error(t, err)
	res, err := st.CloseIssueBy(ctx, iss.ID, "ok", "bob", true)
	require.NoError(t, err)
	assert.Equal(t, "done", res.Issue.Status)
}

func TestHistoryCurrentRow(t *testing.T) {
	st, ctx := openTestStore(t)
	p2 := 2
	iss, err := st.Create(ctx, CreateInput{Title: "Hist", Priority: &p2, Kind: "task"})
	require.NoError(t, err)

	hist, err := st.History(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, "issues", hist.Table)
	require.Len(t, hist.Versions, 1)
	assert.Equal(t, "Hist", hist.Versions[0]["title"])

	note, err := st.Note(ctx, iss.ID, "hello")
	require.NoError(t, err)
	nhist, err := st.History(ctx, note.ID)
	require.NoError(t, err)
	assert.Equal(t, "issue_notes", nhist.Table)

	_, err = st.History(ctx, "o-bad00001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expects f- or n-")
}

func TestReopenPersists(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	st, err := Open(ctx, dir)
	require.NoError(t, err)
	p2 := 2
	iss, err := st.Create(ctx, CreateInput{Title: "Persist", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	require.NoError(t, st.Close())

	st2, err := Open(ctx, dir)
	require.NoError(t, err)
	defer st2.Close()
	got, err := st2.Get(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, "Persist", got.Title)
}
