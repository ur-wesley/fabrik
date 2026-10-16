package doltstore

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitEnv(dir string) []string {
	return append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
}

func runGitDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(dir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

// initBareRemote returns a bare repo path to use as a sync remote.
func initBareRemote(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	bare := t.TempDir()
	runGitDir(t, t.TempDir(), "init", "--bare", bare)
	return bare
}

// initSeededRepo returns a git repo with one commit and an empty issue
// store, with origin pointing at bare.
func initSeededRepo(t *testing.T, bare string) *Store {
	t.Helper()
	dir := t.TempDir()
	runGitDir(t, dir, "init")
	runGitDir(t, dir, "config", "user.name", "t")
	runGitDir(t, dir, "config", "user.email", "t@t")
	runGitDir(t, dir, "commit", "--allow-empty", "-m", "init")
	st, err := Open(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	_, err = st.git(context.Background(), "remote", "add", "origin", bare)
	require.NoError(t, err)
	return st
}

func TestDoltPushNoRemote(t *testing.T) {
	st := openTestGitStore(t)
	_, err := st.DoltPush(context.Background())
	require.ErrorIs(t, err, ErrNoDoltRemote)
	_, err = st.DoltPull(context.Background())
	require.ErrorIs(t, err, ErrNoDoltRemote)
}

func TestRemoteAddList(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	st := openTestGitStore(t)
	ctx := context.Background()
	remotes, err := st.ListRemotes(ctx)
	require.NoError(t, err)
	assert.Empty(t, remotes)
	require.NoError(t, st.RemoteAdd(ctx, "origin", "https://example.com/r.git"))
	remotes, err = st.ListRemotes(ctx)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	assert.Equal(t, "origin", remotes[0].Name)
}

func TestDoltPushPullRoundTrip(t *testing.T) {
	bare := initBareRemote(t)
	ctx := context.Background()
	a := initSeededRepo(t, bare)
	p2 := 2
	iss, err := a.Create(ctx, CreateInput{Title: "Ship it", Priority: &p2, Kind: "task"})
	require.NoError(t, err)

	push, err := a.DoltPush(ctx)
	require.NoError(t, err)
	assert.True(t, push.Pushed)
	assert.Equal(t, "origin", push.Remote)

	// Machine B: fresh clone + fresh (empty) issue store, like a new
	// checkout. Pull is up-to-date (clone has the snapshot), but the
	// ImportFile merge still lands the issues in the local store.
	cloneParent := t.TempDir()
	runGitDir(t, cloneParent, "clone", bare, "b")
	b, err := Open(ctx, cloneParent+"/b")
	require.NoError(t, err)
	defer b.Close()
	pull, err := b.DoltPull(ctx)
	require.NoError(t, err)
	assert.Equal(t, "origin", pull.Remote)
	got, err := b.Show(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, "Ship it", got.Title)

	// Advance A; B's next pull reports pulled commits.
	_, err = a.Create(ctx, CreateInput{Title: "Follow up", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	_, err = a.DoltPush(ctx)
	require.NoError(t, err)
	pull2, err := b.DoltPull(ctx)
	require.NoError(t, err)
	assert.True(t, pull2.Pulled)
	assert.Equal(t, 1, pull2.Commits)
	list, err := b.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestSyncIdempotent(t *testing.T) {
	bare := initBareRemote(t)
	ctx := context.Background()
	a := initSeededRepo(t, bare)
	p2 := 2
	_, err := a.Create(ctx, CreateInput{Title: "Sync me", Priority: &p2, Kind: "task"})
	require.NoError(t, err)

	first, err := a.Sync(ctx)
	require.NoError(t, err)
	assert.True(t, first.Pushed)

	second, err := a.Sync(ctx)
	require.NoError(t, err)
	assert.False(t, second.Pulled)
	assert.False(t, second.Pushed)
	assert.Empty(t, second.Conflicts)
}
