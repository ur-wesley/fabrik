// Package paths resolves user-level install locations.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
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

// GraphifySkills are the 3 app skill destinations (Cursor/OpenCode/Pi only).
func GraphifySkills() ([]string, error) {
	h, err := Home()
	if err != nil {
		return nil, err
	}
	if runtime.GOOS == "windows" {
		return []string{
			filepath.Join(h, ".cursor", "rules", "graphify.mdc"),
			filepath.Join(h, ".config", "opencode", "skills", "graphify.md"),
			filepath.Join(h, ".pi", "agent", "skills", "graphify.md"),
		}, nil
	}
	return []string{
		filepath.Join(h, ".cursor", "rules", "graphify.mdc"),
		filepath.Join(h, ".config", "opencode", "skills", "graphify.md"),
		filepath.Join(h, ".pi", "agent", "skills", "graphify.md"),
	}, nil
}

// WorkflowNotes are (path, cursorFrontmatter) destinations for the note.
type WorkflowNote struct {
	Path        string
	Frontmatter bool
}

// WorkflowNoteDests returns the 3 personal workflow-note destinations.
func WorkflowNoteDests() ([]WorkflowNote, error) {
	h, err := Home()
	if err != nil {
		return nil, err
	}
	return []WorkflowNote{
		{Path: filepath.Join(h, ".cursor", "rules", "ai-workflow.mdc"), Frontmatter: true},
		{Path: filepath.Join(h, ".config", "opencode", "AGENTS.md")},
		{Path: filepath.Join(h, ".pi", "agent", "AGENTS.md")},
	}, nil
}

// CursorFrontmatter is the .mdc header for Cursor rule files.
const CursorFrontmatter = "---\ndescription: Fabrik workflow with Beads, Engram, and Graphify\nalwaysApply: true\n---\n"
