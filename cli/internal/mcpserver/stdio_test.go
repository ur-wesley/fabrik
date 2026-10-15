package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// runStdioSession feeds input lines through runStdio with immediate EOF
// (like `printf ... | fabrik mcp`) and returns everything written to stdout.
func runStdioSession(t *testing.T, lines ...string) string {
	t.Helper()
	in := strings.Join(lines, "\n")
	if in != "" {
		in += "\n"
	}
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runStdio(context.Background(), NewServer(), strings.NewReader(in), &out)
	}()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(20 * time.Second):
		t.Fatal("runStdio did not exit after EOF")
	}
	return out.String()
}

func responseIDs(t *testing.T, stdout string) map[string]bool {
	t.Helper()
	ids := map[string]bool{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var v map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &v), "stdout line must be JSON: %q", line)
		if id, ok := v["id"]; ok && id != nil {
			ids[idKey(id)] = true
		}
	}
	return ids
}

func initLine(id any) string {
	raw, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": id, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
			"clientInfo":      map[string]any{"name": "t", "version": "1"},
		},
	})
	return string(raw)
}

func reqLine(id any, method string) string {
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method})
	return string(raw)
}

// Piped single initialize + immediate EOF must be answered (cli-6h6:
// piped-EOF exited 0 with 0 bytes).
func TestRunStdioPipedInitAnswers(t *testing.T) {
	out := runStdioSession(t, initLine(1))
	ids := responseIDs(t, out)
	require.True(t, ids[idKey(float64(1))], "expected response to id 1, got stdout %q", out)
	require.Contains(t, out, "fabrik")
}

// Piped full session at once + immediate EOF must answer every request
// (cli-q2r: tools/list never answered).
func TestRunStdioPipedSessionAnswersAll(t *testing.T) {
	out := runStdioSession(t,
		initLine(1),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		reqLine(2, "ping"),
		reqLine(3, "tools/list"),
	)
	ids := responseIDs(t, out)
	require.True(t, ids[idKey(float64(1))], "missing initialize response: %q", out)
	require.True(t, ids[idKey(float64(2))], "missing ping response: %q", out)
	require.True(t, ids[idKey(float64(3))], "missing tools/list response: %q", out)
	require.Contains(t, out, "graph_explore")
}
