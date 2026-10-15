package flow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckSetupWithoutBeads(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".fabrik", "dolt"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".fabrik", "config.yaml"), []byte("x: 1\n"), 0o644))
	require.NoError(t, CheckSetup(repo))
}
