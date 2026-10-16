package issuestore

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSnapshot encodes the store as .fabrik/export/issues.jsonl records
// (same shape export.Issues writes) without importing the export package.
func writeSnapshot(t *testing.T, st *Store, ctx context.Context, path string) {
	t.Helper()
	issues, err := st.List(ctx, "")
	require.NoError(t, err)
	deps, err := st.IssueDeps(ctx)
	require.NoError(t, err)
	byBlocked := map[string][]Dep{}
	for _, d := range deps {
		byBlocked[d.BlockedID] = append(byBlocked[d.BlockedID], d)
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, iss := range issues {
		notes, err := st.IssueNotes(ctx, iss.ID)
		require.NoError(t, err)
		for i := range notes {
			notes[i].IssueID = iss.ID
		}
		if notes == nil {
			notes = []Note{}
		}
		rec := Record{ExportIssue: iss.Export(), Deps: byBlocked[iss.ID], Notes: notes}
		if rec.Deps == nil {
			rec.Deps = []Dep{}
		}
		require.NoError(t, enc.Encode(rec))
	}
}

func withBumpedTitle(rec Record, title, updated string) Record {
	rec.Title = title
	rec.UpdatedAt = updated
	return rec
}

func TestImportLastWriteWinsOrderIndependent(t *testing.T) {
	ctx := context.Background()
	p2 := 1

	build := func(t *testing.T, dir string) (*Store, string, string) {
		st, err := Open(ctx, dir)
		require.NoError(t, err)
		t.Cleanup(func() { _ = st.Close() })
		blocker, err := st.Create(ctx, CreateInput{Title: "Blocker", Priority: &p2, Kind: "task"})
		require.NoError(t, err)
		blocked, err := st.Create(ctx, CreateInput{Title: "Blocked v1", Priority: &p2, Kind: "task"})
		require.NoError(t, err)
		_, err = st.Dep(ctx, blocked.ID, blocker.ID, "blocks")
		require.NoError(t, err)
		note, err := st.Note(ctx, blocked.ID, "first")
		require.NoError(t, err)
		return st, blocked.ID, note.ID
	}

	// Machine A snapshot.
	dirA := t.TempDir()
	stA, blockedID, noteAID := build(t, dirA)
	snapA := filepath.Join(dirA, "a.jsonl")
	writeSnapshot(t, stA, ctx, snapA)

	// Machine B: same base, then a newer title + extra note + extra dep edge.
	// Timestamps are rewritten explicitly so the test is deterministic down
	// to the second (RFC3339 has no sub-second parts here).
	raw, err := os.ReadFile(snapA)
	require.NoError(t, err)
	newer := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	var recs []Record
	for _, line := range splitLines(t, raw) {
		var rec Record
		require.NoError(t, json.Unmarshal([]byte(line), &rec))
		if rec.ID == blockedID {
			rec = withBumpedTitle(rec, "Blocked v2", newer)
			rec.Notes = append(rec.Notes, Note{ID: "n-abcdef01", IssueID: blockedID, Body: "second", CreatedAt: newer})
		}
		recs = append(recs, rec)
	}
	snapB := filepath.Join(dirA, "b.jsonl")
	f, err := os.Create(snapB)
	require.NoError(t, err)
	enc := json.NewEncoder(f)
	for _, r := range recs {
		require.NoError(t, enc.Encode(r))
	}
	require.NoError(t, f.Close())

	merge := func(t *testing.T, first, second string) Issue {
		dir := t.TempDir()
		st, err := Open(ctx, dir)
		require.NoError(t, err)
		defer st.Close()
		r1, err := st.ImportFile(ctx, first)
		require.NoError(t, err)
		require.Equal(t, 2, r1.Issues)
		r2, err := st.ImportFile(ctx, second)
		require.NoError(t, err)
		_ = r2
		got, err := st.Show(ctx, blockedID)
		require.NoError(t, err)
		return got
	}

	// Both orders converge: newer title wins, notes/deps union.
	for _, order := range [][2]string{{snapA, snapB}, {snapB, snapA}} {
		got := merge(t, order[0], order[1])
		assert.Equal(t, "Blocked v2", got.Title)
		assert.Equal(t, newer, got.UpdatedAt)
		bodies := []string{got.Notes[0].Body, got.Notes[1].Body}
		assert.ElementsMatch(t, []string{"first", "second"}, bodies)
		ids := []string{got.Notes[0].ID, got.Notes[1].ID}
		assert.Contains(t, ids, noteAID)
		assert.Contains(t, ids, "n-abcdef01")
		require.Len(t, got.Blockers, 1)
	}

	// Re-import is idempotent: everything already present.
	dir := t.TempDir()
	st, err := Open(ctx, dir)
	require.NoError(t, err)
	defer st.Close()
	_, err = st.ImportFile(ctx, snapB)
	require.NoError(t, err)
	r, err := st.ImportFile(ctx, snapB)
	require.NoError(t, err)
	assert.Equal(t, 0, r.Issues)
	assert.Equal(t, 2, r.IssuesPresent)
	assert.Equal(t, 0, r.Deps)
	assert.Equal(t, 1, r.DepsPresent)
	assert.Equal(t, 0, r.Notes)
	assert.Equal(t, 2, r.NotesPresent)
}

func TestImportViaGitPlusJSONL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	ctx := context.Background()
	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}

	// Machine A: store + snapshot, snapshot committed to git.
	repoA := t.TempDir()
	runGit(repoA, "init")
	stA, err := Open(ctx, repoA)
	require.NoError(t, err)
	p1 := 1
	blocker, err := stA.Create(ctx, CreateInput{Title: "Ship it", Priority: &p1, Kind: "task"})
	require.NoError(t, err)
	blocked, err := stA.Create(ctx, CreateInput{Title: "Follow up", Priority: &p1, Kind: "task"})
	require.NoError(t, err)
	_, err = stA.Dep(ctx, blocked.ID, blocker.ID, "blocks")
	require.NoError(t, err)
	_, err = stA.Note(ctx, blocked.ID, "context")
	require.NoError(t, err)
	snapPath := filepath.Join(repoA, ".fabrik", "export", "issues.jsonl")
	writeSnapshot(t, stA, ctx, snapPath)
	require.NoError(t, stA.Close())
	runGit(repoA, "add", ".fabrik/export/issues.jsonl")
	runGit(repoA, "commit", "-m", "sync issues")

	// Machine B: fresh clone, empty local store, merge from the tracked file.
	repoB := t.TempDir()
	runGit(t.TempDir(), "clone", repoA, repoB)
	stB, err := Open(ctx, repoB)
	require.NoError(t, err)
	defer stB.Close()
	res, err := stB.ImportFile(ctx, filepath.Join(repoB, ".fabrik", "export", "issues.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, 2, res.Issues)
	assert.Equal(t, 1, res.Deps)
	assert.Equal(t, 1, res.Notes)

	got, err := stB.Show(ctx, blocked.ID)
	require.NoError(t, err)
	assert.Equal(t, "Follow up", got.Title)
	assert.Equal(t, "blocked", got.Status)
	require.Len(t, got.Blockers, 1)
	assert.Equal(t, blocker.ID, got.Blockers[0].ID)
	require.Len(t, got.Notes, 1)
	assert.Equal(t, "context", got.Notes[0].Body)
}

func splitLines(t *testing.T, raw []byte) []string {
	t.Helper()
	var out []string
	for _, line := range splitBytes(raw) {
		if len(line) > 0 {
			out = append(out, string(line))
		}
	}
	require.NotEmpty(t, out)
	return out
}

func splitBytes(raw []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, b := range raw {
		if b == '\n' {
			out = append(out, raw[start:i])
			start = i + 1
		}
	}
	if start < len(raw) {
		out = append(out, raw[start:])
	}
	return out
}
