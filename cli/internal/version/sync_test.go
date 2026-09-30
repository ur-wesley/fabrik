package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageJSONInSync ensures cli/package.json (canonical version file)
// matches the embedded copy.
func TestPackageJSONInSync(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "package.json"))
	require.NoError(t, err)
	raw, err := os.ReadFile(root)
	if err != nil {
		t.Skipf("no cli/package.json nearby: %v", err)
	}
	var outer struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(raw, &outer))
	var inner struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(packageJSON, &inner))
	assert.Equal(t, outer.Version, inner.Version, "re-copy cli/package.json to internal/version/package.json")
	assert.Equal(t, outer.Version, Get())
}
