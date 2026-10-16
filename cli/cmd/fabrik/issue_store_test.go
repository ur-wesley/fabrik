package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/issuestore"
)

// initIssueStore creates the SQLite issue store (.fabrik/issues.db) in dir so
// CLI tests have a repo root to resolve. No CGO needed, unlike the legacy
// embedded-Dolt store.
func initIssueStore(t *testing.T, dir string) {
	t.Helper()
	st, err := issuestore.Open(context.Background(), dir)
	require.NoError(t, err)
	require.NoError(t, st.Close())
}
