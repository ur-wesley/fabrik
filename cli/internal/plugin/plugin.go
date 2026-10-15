// Package plugin syncs the global Antigravity/Gemini CLI plugin copy at
// ~/.gemini/config/plugins/fabrik from the embedded repo source.
//
// Background: nothing in the repo ever installed that global copy (no
// installer or doc references config/plugins; legacy shell installers only
// targeted Cursor/OpenCode/Pi), so it goes stale — e.g. v0.1.0 with the old
// bun/pi/uv stack while the repo source plugins/fabrik is v0.2.0. Antigravity
// prefers the global copy over local .agents/skills shims, so a stale copy
// shadows current docs.
//
// The source of truth is the repo tree plugins/fabrik. Because go:embed
// rejects ".." patterns, this package embeds a checked-in mirror under
// fabrik/; plugin_drift_test.go fails on any drift (file list or contents)
// between the mirror and the repo tree — re-copy on change:
//
//	cp -rf plugins/fabrik cli/internal/plugin/fabrik
package plugin

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
)

//go:embed fabrik
var dist embed.FS

// PluginName is the directory/file name of the plugin on both ends.
const PluginName = "fabrik"

// Dir returns the global plugin directory for a home dir:
// ~/.gemini/config/plugins/fabrik.
func Dir(home string) string {
	return filepath.Join(home, ".gemini", "config", "plugins", PluginName)
}

// Version parses the version from the embedded plugin.json.
// Returns "" when the embedded manifest is unparsable (never in practice;
// the drift test guards the mirror).
func Version() string {
	raw, err := dist.ReadFile("fabrik/plugin.json")
	if err != nil {
		return ""
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return ""
	}
	return strings.TrimSpace(manifest.Version)
}

// InstalledVersion reads the version of the global plugin installed for
// home. Returns "" when missing or unparsable.
func InstalledVersion(home string) string {
	raw, err := os.ReadFile(filepath.Join(Dir(home), "plugin.json"))
	if err != nil {
		return ""
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return ""
	}
	return strings.TrimSpace(manifest.Version)
}

// cmpVersion compares dot-separated versions numerically per part
// (missing parts count as 0, leading "v" ignored). Returns -1, 0, +1.
func cmpVersion(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(strings.TrimSpace(a), "v"), ".")
	pb := strings.Split(strings.TrimPrefix(strings.TrimSpace(b), "v"), ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		na, nb := 0, 0
		if i < len(pa) {
			na = leadingInt(pa[i])
		}
		if i < len(pb) {
			nb = leadingInt(pb[i])
		}
		if na < nb {
			return -1
		}
		if na > nb {
			return 1
		}
	}
	return 0
}

// leadingInt parses the leading decimal digits of s; 0 when none.
func leadingInt(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	n, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0
	}
	return n
}

// NeedsSync reports whether the global copy for home is missing,
// unparsable, or semantically older than the embedded source.
// A newer installed copy is left alone.
func NeedsSync(home string) bool {
	installed := InstalledVersion(home)
	if installed == "" {
		return true
	}
	embedded := Version()
	if embedded == "" {
		return false
	}
	return cmpVersion(installed, embedded) < 0
}

// Files lists embedded plugin paths (slash-separated, "fabrik/..." prefix).
func Files() []string {
	var out []string
	_ = fs.WalkDir(dist, "fabrik", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// Sync copies the embedded plugin tree over the global copy for home,
// overwriting files and creating dirs. When already up to date it is a
// no-op printing `plugin fabrik <v> up to date`. With dryRun nothing is
// written; it prints the would-copy count instead. Always prints one
// `plugin fabrik ...` or `dry-run: sync fabrik plugin ...` line to out.
func Sync(home string, dryRun bool, out io.Writer) error {
	embedded := Version()
	installed := InstalledVersion(home)
	if !NeedsSync(home) {
		fmt.Fprintf(out, "plugin %s %s up to date\n", PluginName, embedded)
		return nil
	}
	old := installed
	if old == "" {
		old = "missing"
	}
	files := Files()
	if dryRun {
		fmt.Fprintf(out, "dry-run: sync fabrik plugin %s -> %s (%d files would be copied)\n", old, embedded, len(files))
		return nil
	}
	dest := Dir(home)
	for _, f := range files {
		data, err := dist.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", f, err)
		}
		rel := strings.TrimPrefix(filepath.FromSlash(f), "fabrik"+string(filepath.Separator))
		dst := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(dst), fileutil.DirPerm); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, fileutil.ConfigPerm); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "plugin %s %s -> %s\n", PluginName, old, embedded)
	return nil
}

// Advisory returns a machine-level freshness warning for home, or "" when
// the global copy is current. Advisory only — callers must not fail on it.
func Advisory(home string) string {
	if !NeedsSync(home) {
		return ""
	}
	if installed := InstalledVersion(home); installed != "" {
		return fmt.Sprintf("warning: antigravity plugin %s %s is stale (embedded %s); re-run 'fabrik setup --apps antigravity' to refresh", PluginName, installed, Version())
	}
	return fmt.Sprintf("warning: antigravity plugin %s is not installed (embedded %s); run 'fabrik setup --apps antigravity' to install", PluginName, Version())
}
