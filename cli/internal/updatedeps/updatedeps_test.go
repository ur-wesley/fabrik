package updatedeps

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedDeps(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "deps.json")
	raw := `{"beads":{"version":"1.3.0","tag":"v1.3.0","repo":"gastownhall/beads","binary":"bd","assets":{}},"engram":{"version":"2.2.1","tag":"v2.2.1","repo":"Gentleman-Programming/engram","goModule":"github.com/Gentleman-Programming/engram/v2/cmd/engram","binary":"engram","assets":{}},"graphify":{"pypi":"graphifyy","version":"0.9.69","skillRef":"v3","skillUrl":"https://example/x"},"pi":{"mcpAdapter":"npm:@piarium/pi-mcp-adapter"},"agents":["cursor"],"skills":["tdd"],"subagents":["explore"]}`
	require.NoError(t, os.WriteFile(p, []byte(raw), 0o644))
	return p
}

func TestRefreshRewritesPins(t *testing.T) {
	p := seedDeps(t)
	var out bytes.Buffer
	f := &Fetcher{
		LatestTag:   map[string]string{"gastownhall/beads": "v1.4.0", "Gentleman-Programming/engram": "v2.3.0"},
		PypiVersion: map[string]string{"graphifyy": "0.9.70"},
	}
	require.NoError(t, Refresh(context.Background(), &out, f, p, true, false))
	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	assert.Equal(t, "v1.4.0", doc["beads"].(map[string]any)["tag"])
	assert.Equal(t, "1.4.0", doc["beads"].(map[string]any)["version"])
	assets := doc["beads"].(map[string]any)["assets"].(map[string]any)
	assert.Equal(t, "beads_1.4.0_windows_amd64.zip", assets["windows_amd64"])
	assert.Equal(t, "beads_1.4.0_linux_arm64.tar.gz", assets["linux_arm64"])
	assert.Equal(t, "0.9.70", doc["graphify"].(map[string]any)["version"])
	// untouched fields preserved
	assert.Equal(t, "npm:@piarium/pi-mcp-adapter", doc["pi"].(map[string]any)["mcpAdapter"])
	assert.Contains(t, out.String(), "Updated deps.json")
}

func TestRefreshDryRun(t *testing.T) {
	p := seedDeps(t)
	before, _ := os.ReadFile(p)
	var out bytes.Buffer
	f := &Fetcher{
		LatestTag:   map[string]string{"gastownhall/beads": "v9.0.0", "Gentleman-Programming/engram": "v9.0.0"},
		PypiVersion: map[string]string{"graphifyy": "9.0.0"},
	}
	require.NoError(t, Refresh(context.Background(), &out, f, p, true, true))
	after, _ := os.ReadFile(p)
	assert.Equal(t, string(before), string(after))
	assert.Contains(t, out.String(), "dry-run")
}
