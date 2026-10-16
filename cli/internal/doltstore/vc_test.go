package doltstore

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openTestGitStore opens the SQLite issue store in a fresh git repo, so the
// git-backed VC/sync methods have a backend. The SQLite store itself needs
// no git; only vc status/log/diff/commit/branch/checkout/merge and
// dolt push/pull/remote do.
func openTestGitStore(t *testing.T) *Store {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "fabrik-test")
	runGit(t, dir, "config", "user.email", "fabrik-test@local")
	// An initial commit gives HEAD/branch semantics from the start.
	runGit(t, dir, "commit", "--allow-empty", "-m", "init")
	ctx := context.Background()
	st, err := Open(ctx, dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func TestVCStatusCleanAfterCreate(t *testing.T) {
	st := openTestGitStore(t)
	ctx := context.Background()
	p2 := 2
	_, err := st.Create(ctx, CreateInput{Title: "VC test", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	// issues.db is gitignored: the working tree stays clean until a vc
	// commit exports the snapshot.
	status, err := st.VCStatus(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, status.Branch)
	assert.NotEmpty(t, status.Head)
	assert.False(t, status.Dirty, "unexpected dirty: %+v", status.Changes)
}

func TestVCCommitLogDiff(t *testing.T) {
	st := openTestGitStore(t)
	ctx := context.Background()
	p2 := 2
	_, err := st.Create(ctx, CreateInput{Title: "History row", Priority: &p2, Kind: "task"})
	require.NoError(t, err)

	commit, err := st.VCCommitMessage(ctx, "snapshot issues")
	require.NoError(t, err)
	assert.NotEmpty(t, commit.Hash)

	log, err := st.VCLog(ctx, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, log.Commits)
	assert.Equal(t, "snapshot issues", log.Commits[0].Message)

	status, err := st.VCStatus(ctx)
	require.NoError(t, err)
	assert.False(t, status.Dirty)

	// A second commit with no changes is a no-op error, like Dolt.
	_, err = st.VCCommitMessage(ctx, "nothing new")
	require.Error(t, err)
	assert.True(t, IsNothingToCommit(err))

	diff, err := st.VCDiffWorking(ctx)
	require.NoError(t, err)
	assert.Empty(t, diff.Tables)

	ranged, err := st.VCDiffRange(ctx, commit.Hash+"~1", commit.Hash)
	require.NoError(t, err)
	assert.NotEmpty(t, ranged.Tables)

	_, err = st.VCDiffRange(ctx, "deadbeefdeadbeef", "HEAD")
	var unknown *ErrUnknownCommit
	require.ErrorAs(t, err, &unknown)
}

func TestVCBranchCheckoutMerge(t *testing.T) {
	st := openTestGitStore(t)
	ctx := context.Background()

	list, err := st.VCBranchList(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, list.Current)
	assert.Contains(t, list.Branches, list.Current)

	require.Error(t, st.VCBranchCreate(ctx, "bad name!"))

	require.NoError(t, st.VCBranchCreate(ctx, "feature"))
	require.NoError(t, st.VCCheckout(ctx, "feature"))
	cur, err := st.VCBranchList(ctx)
	require.NoError(t, err)
	assert.Equal(t, "feature", cur.Current)

	var nf *ErrBranchNotFound
	require.ErrorAs(t, st.VCCheckout(ctx, "nope-missing"), &nf)

	require.NoError(t, st.VCCheckout(ctx, list.Current))
	_, err = st.VCMerge(ctx, list.Current, "ours")
	require.Error(t, err, "merging a branch into itself must fail")

	merged, err := st.VCMerge(ctx, "feature", "")
	require.NoError(t, err)
	assert.True(t, merged.Merged)
	assert.NotEmpty(t, merged.Hash)

	require.Error(t, func() error {
		_, err := st.VCMerge(ctx, "feature", "bogus")
		return err
	}())
}

func TestVCHistory(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	p2 := 2
	iss, err := st.Create(ctx, CreateInput{Title: "History row", Priority: &p2, Kind: "task"})
	require.NoError(t, err)

	hist, err := st.History(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, "issues", hist.Table)
	assert.NotEmpty(t, hist.Versions)
}

func TestHistoryBadPrefix(t *testing.T) {
	st := openTestStore(t)
	_, err := st.History(context.Background(), "o-bad00001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expects f- or n-")
}

func TestImportLastWriteWins(t *testing.T) {
	ctx := context.Background()
	p2 := 1
	dirA := t.TempDir()
	a, err := Open(ctx, dirA)
	require.NoError(t, err)
	defer a.Close()
	blocker, err := a.Create(ctx, CreateInput{Title: "Blocker", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	blocked, err := a.Create(ctx, CreateInput{Title: "Blocked v1", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	_, err = a.Dep(ctx, blocked.ID, blocker.ID, "blocks")
	require.NoError(t, err)
	_, err = a.Note(ctx, blocked.ID, "first")
	require.NoError(t, err)
	n, err := a.exportSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	dirB := t.TempDir()
	b, err := Open(ctx, dirB)
	require.NoError(t, err)
	defer b.Close()
	res, err := b.ImportFile(ctx, a.ExportPath())
	require.NoError(t, err)
	assert.Equal(t, 2, res.Issues)
	assert.Equal(t, 1, res.Deps)
	assert.Equal(t, 1, res.Notes)

	got, err := b.Show(ctx, blocked.ID)
	require.NoError(t, err)
	assert.Equal(t, "blocked", got.Status)
	require.Len(t, got.Notes, 1)

	again, err := b.ImportFile(ctx, a.ExportPath())
	require.NoError(t, err)
	assert.Equal(t, 0, again.Issues)
	assert.Equal(t, 2, again.IssuesPresent)
}
