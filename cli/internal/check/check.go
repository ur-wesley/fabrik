// Package check implements `fabrik check` (port of install/check.sh + check.ps1).
package check

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/exec"
)

// Tools probed in stable order.
var Tools = []string{"bd", "engram", "graphify", "bun", "pi", "uv"}

// OptionalTools never affect OK. Jev is an MCP decision helper, not a PATH
// binary: it counts as configured only when TYPESAFE_API_KEY is set and the
// host app enables the `jev` MCP server (local stdio agents only).
// See skills/jev-loop.md. tools.jev in .fabrik/config.yaml: auto|true|false.
var OptionalTools = []string{"jev"}

// JevStatus reports the optional Jev decision helper status. It never fails
// the overall check: "ok" when TYPESAFE_API_KEY is set (MCP server `jev`
// still required in the host app), "missing (optional)" otherwise.
func JevStatus() string {
	if v := os.Getenv("TYPESAFE_API_KEY"); v != "" {
		return "ok (key set; requires MCP server jev)"
	}
	return "missing (optional; set TYPESAFE_API_KEY + enable MCP server jev)"
}

// Result holds per-tool status.
type Result struct {
	Status map[string]string `json:"-"`
	OK     bool              `json:"ok"`
	// versions for informational display
	Versions map[string]string `json:"-"`
}

// Run probes PATH for each tool.
func Run(ctx context.Context, ex exec.Runner) Result {
	r := Result{Status: map[string]string{}, Versions: map[string]string{}}
	fail := false
	for _, t := range Tools {
		if _, err := ex.LookPath(t); err != nil {
			r.Status[t] = "missing"
			fail = true
		} else {
			r.Status[t] = "ok"
		}
	}
	r.OK = !fail
	r.Status["jev"] = JevStatus()
	if r.Status["bd"] == "ok" {
		if v, err := ex.Run(ctx, "bd", "version"); err == nil {
			r.Versions["bd"] = firstLine(v)
		}
	}
	if r.Status["engram"] == "ok" {
		if v, err := ex.Run(ctx, "engram", "version"); err == nil {
			r.Versions["engram"] = firstLine(v)
		}
	}
	if r.Status["graphify"] == "ok" {
		if v, err := ex.Run(ctx, "graphify", "--version"); err == nil {
			r.Versions["graphify"] = firstLine(v)
		}
	}
	return r
}

// Print renders human output.
func Print(out io.Writer, r Result) {
	for _, t := range Tools {
		s := r.Status[t]
		if v, ok := r.Versions[t]; ok && v != "" {
			s += " (" + v + ")"
		}
		fmt.Fprintf(out, "%s: %s\n", t, s)
	}
	for _, t := range OptionalTools {
		fmt.Fprintf(out, "%s: %s\n", t, r.Status[t])
	}
	if !r.OK {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Missing tools. Run fabrik setup.")
	}
}

// PrintJSON renders machine output (shell parity: required tool keys + optional jev + ok).
func PrintJSON(out io.Writer) error {
	return PrintJSONTo(out, Run(context.Background(), exec.OSRunner{}))
}

// PrintJSONTo renders a given result as JSON.
func PrintJSONTo(out io.Writer, r Result) error {
	m := map[string]any{}
	for _, t := range Tools {
		m[t] = r.Status[t]
	}
	for _, t := range OptionalTools {
		m[t] = r.Status[t]
	}
	m["ok"] = r.OK
	return json.NewEncoder(out).Encode(m)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

var _ = os.Stdout
