// Package paths resolves user-level install locations.
package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// Home returns the user home directory.
func Home() (string, error) { return os.UserHomeDir() }

// BinDir is ~/.local/bin on all platforms (Windows included, like setup.ps1).
func BinDir() (string, error) {
	h, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "bin"), nil
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
