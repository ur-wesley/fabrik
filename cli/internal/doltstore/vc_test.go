//go:build cgo

package doltstore

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func openTestStore(t *testing.T) *Store {
	if os.Getenv("CGO_ENABLED") == "0" {
		t.Skip("requires CGO_ENABLED=1")
	}
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(store.DoltDir(dir), 0o755))
	ctx := context.Background()
	st, err := Open(ctx, dir)
	require.NoError(t, err)
	return st
}

func TestVCStatusAfterCreate(t *testing.T) {
	st := openTestStore(t)
	defer st.Close()
	ctx := context.Background()
	p2 := 2
	_, err := st.Create(ctx, CreateInput{Title: "VC test", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	status, err := st.VCStatus(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, status.Branch)
	assert.NotEmpty(t, status.Head)
	if status.Dirty {
		t.Logf("dirty changes: %+v", status.Changes)
	}
	assert.False(t, status.Dirty, "unexpected dirty: %+v", status.Changes)
}

func TestVCLogAndHistory(t *testing.T) {
	st := openTestStore(t)
	defer st.Close()
	ctx := context.Background()
	p2 := 2
	iss, err := st.Create(ctx, CreateInput{Title: "History row", Priority: &p2, Kind: "task"})
	require.NoError(t, err)
	log, err := st.VCLog(ctx, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, log.Commits)

	hist, err := st.History(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, "issues", hist.Table)
	assert.NotEmpty(t, hist.Versions)
}

func TestHistoryBadPrefix(t *testing.T) {
	st := openTestStore(t)
	defer st.Close()
	_, err := st.History(context.Background(), "o-bad00001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expects f- or n-")
}
