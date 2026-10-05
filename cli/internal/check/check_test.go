package check

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
)

func TestRunAllMissing(t *testing.T) {
	r := Run(context.Background(), &exec.Fake{})
	assert.False(t, r.OK)
	for _, tool := range Tools {
		assert.Equal(t, "missing", r.Status[tool])
	}
}

func TestRunAllPresent(t *testing.T) {
	f := &exec.Fake{
		Path: map[string]bool{"bd": true, "engram": true, "graphify": true, "bun": true, "pi": true, "uv": true},
		Outputs: map[string]string{
			"bd version":         "bd version 1.3.0",
			"engram version":     "engram 2.2.1",
			"graphify --version": "graphifyy 0.9.69",
		},
	}
	r := Run(context.Background(), f)
	assert.True(t, r.OK)
	assert.Equal(t, "bd version 1.3.0", r.Versions["bd"])
}

func TestPrintJSONShape(t *testing.T) {
	f := &exec.Fake{Path: map[string]bool{"bd": true}}
	r := Run(context.Background(), f)
	var buf bytes.Buffer
	require.NoError(t, PrintJSONTo(&buf, r))
	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	assert.Equal(t, "ok", m["bd"])
	assert.Equal(t, "missing", m["pi"])
	assert.Equal(t, false, m["ok"])
}

func TestJevOptionalMissingByDefault(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	r := Run(context.Background(), &exec.Fake{})
	assert.Contains(t, r.Status["jev"], "missing (optional")
	assert.False(t, r.OK) // OK still driven by required tools only
}

func TestJevOptionalOkWithKey(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "test-key")
	r := Run(context.Background(), &exec.Fake{})
	assert.Contains(t, r.Status["jev"], "ok")
	assert.False(t, r.OK) // required tools still missing; jev never flips OK
}

func TestJevNeverFailsCheck(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	f := &exec.Fake{
		Path: map[string]bool{"bd": true, "engram": true, "graphify": true, "bun": true, "pi": true, "uv": true},
	}
	r := Run(context.Background(), f)
	assert.True(t, r.OK)
	assert.Contains(t, r.Status["jev"], "missing (optional")
}
