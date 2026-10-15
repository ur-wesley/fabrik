// Package paths resolves user-level install locations.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Home returns the user home directory.
func Home() (string, error) { return os.UserHomeDir() }

// BinDir is ~/.local/bin on Unix. On Windows it is %APPDATA%\fabrik\bin
// (a location users can add to PATH once), falling back to
// %USERPROFILE%\.local\bin when APPDATA is unset.
func BinDir() (string, error) {
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "fabrik", "bin"), nil
		}
		h, err := Home()
		if err != nil {
			return "", err
		}
		return filepath.Join(h, ".local", "bin"), nil
	}
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "bin"), nil
}

// OnPATH reports whether dir is listed in the PATH environment variable
// (case-insensitive on Windows, where PATH entries vary in case).
func OnPATH(dir string) bool {
	want := filepath.Clean(dir)
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == "" {
			continue
		}
		if runtime.GOOS == "windows" {
			if strings.EqualFold(filepath.Clean(p), want) {
				return true
			}
			continue
		}
		if filepath.Clean(p) == want {
			return true
		}
	}
	return false
}

// WarnIfNotOnPATH returns a warning message when dir is not on PATH, or
// "" when it is. Callers print it so users know to add BinDir to PATH.
func WarnIfNotOnPATH(dir string) string {
	if OnPATH(dir) {
		return ""
	}
	return fmt.Sprintf("warning: %s is not on PATH; add it to use installed binaries", dir)
}

// GraphifySkills returns skill destinations for the specified apps (or all supported apps if empty).
func GraphifySkills(selectedApps ...string) ([]string, error) {
	h, err := Home()
	if err != nil {
		return nil, err
	}
	appFilter := make(map[string]bool)
	for _, a := range selectedApps {
		appFilter[strings.ToLower(strings.TrimSpace(a))] = true
	}
	match := func(app string) bool {
		if len(appFilter) == 0 {
			return true
		}
		return appFilter[app]
	}

	var dests []string
	if match("cursor") {
		dests = append(dests, filepath.Join(h, ".cursor", "rules", "graphify.mdc"))
	}
	if match("opencode") {
		dests = append(dests, filepath.Join(h, ".config", "opencode", "skills", "graphify.md"))
	}
	if match("pi") {
		dests = append(dests, filepath.Join(h, ".pi", "agent", "skills", "graphify.md"))
	}
	if match("antigravity") {
		dests = append(dests, filepath.Join(h, ".gemini", "antigravity", "skills", "graphify", "SKILL.md"))
	}
	if match("gemini") {
		dests = append(dests, filepath.Join(h, ".gemini", "skills", "graphify", "SKILL.md"))
	}
	return dests, nil
}

// WorkflowNotes are (path, cursorFrontmatter) destinations for the note.
type WorkflowNote struct {
	Path        string
	Frontmatter bool
}

// WorkflowNoteDests returns personal workflow-note destinations for the specified apps (or all supported if empty).
func WorkflowNoteDests(selectedApps ...string) ([]WorkflowNote, error) {
	h, err := Home()
	if err != nil {
		return nil, err
	}
	appFilter := make(map[string]bool)
	for _, a := range selectedApps {
		appFilter[strings.ToLower(strings.TrimSpace(a))] = true
	}
	match := func(app string) bool {
		if len(appFilter) == 0 {
			return true
		}
		return appFilter[app]
	}

	var dests []WorkflowNote
	if match("cursor") {
		dests = append(dests, WorkflowNote{Path: filepath.Join(h, ".cursor", "rules", "ai-workflow.mdc"), Frontmatter: true})
	}
	if match("pi") {
		dests = append(dests, WorkflowNote{Path: filepath.Join(h, ".pi", "agent", "AGENTS.md")})
	}
	return dests, nil
}

// CursorFrontmatter is the .mdc header for Cursor rule files.
const CursorFrontmatter = "---\ndescription: Fabrik workflow with Beads, Engram, and Graphify\nalwaysApply: true\n---\n"
