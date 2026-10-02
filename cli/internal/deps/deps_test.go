package deps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	d, err := Load()
	require.NoError(t, err)
	assert.NotEmpty(t, d.Beads.Tag)
	assert.NotEmpty(t, d.Engram.Tag)
	assert.NotEmpty(t, d.Graphify.Version)
	assert.NotEmpty(t, d.Pi.McpAdapter)
	assert.NotEmpty(t, d.Agents)
	assert.NotEmpty(t, d.Skills)
	assert.NotEmpty(t, d.Subagents)
	// cli-dk8: shared CTA snippet must stay pinned so `fabrik init`
	// installs it (thin: show-shims, full: .fabrik/agents/cta.md).
	assert.Contains(t, d.Subagents, "cta")
	assert.Contains(t, d.Subagents, "plan")
	// six pinned assets per binary tool
	assert.Len(t, d.Beads.Assets, 6)
	assert.Len(t, d.Engram.Assets, 6)
}

func TestAssetCurrent(t *testing.T) {
	d, err := Load()
	require.NoError(t, err)
	repo, tag, asset, binary, err := Asset(d, "beads")
	require.NoError(t, err)
	assert.Contains(t, repo, "/")
	assert.NotEmpty(t, tag)
	assert.NotEmpty(t, asset)
	assert.Equal(t, "bd", binary)
}

func TestAssetUnknown(t *testing.T) {
	d, err := Load()
	require.NoError(t, err)
	_, _, _, _, err = Asset(d, "nope")
	require.Error(t, err)
}

// TestEmbeddedInSyncWithInstall guards the embedded copy against drift.
// Runs only inside the fabrik checkout (skips when install/deps.json is absent,
// e.g. for consumers of the module).
func TestEmbeddedInSyncWithInstall(t *testing.T) {
	// cli/internal/deps -> cli -> fabrik
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "install", "deps.json"))
	require.NoError(t, err)
	rawInstall, err := os.ReadFile(root)
	if err != nil {
		t.Skipf("no install/deps.json nearby: %v", err)
	}
	var a, b map[string]any
	require.NoError(t, json.Unmarshal(rawInstall, &a))
	embedded, err := Load()
	require.NoError(t, err)
	emb, err := json.Marshal(embedded)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(emb, &b))
	assert.Equal(t, a, b, "embedded deps.json drifted from install/deps.json: re-copy it")
}
