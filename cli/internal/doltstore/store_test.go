package doltstore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	st, err := Open(ctx, dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	_, err := st.Create(ctx, CreateInput{Title: "   ", Kind: "task"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestDepAfterCloseReturnsError(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	p2 := 2
	a, err := st.Create(ctx, CreateInput{Title: "A", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	b, err := st.Create(ctx, CreateInput{Title: "B", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	require.NoError(t, st.Close())
	_, err = st.Dep(ctx, b.ID, a.ID, "blocks")
	require.Error(t, err)
}

func TestCreateDefaultPriority(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	iss, err := st.Create(ctx, CreateInput{Title: "Default priority", Kind: "task"})
	require.NoError(t, err)
	assert.Equal(t, 2, iss.Priority)
}

func TestOpenListCreateShow(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

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
}

func TestReadyClaimCloseDepNote(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

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
}
