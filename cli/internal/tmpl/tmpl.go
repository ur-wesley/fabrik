// Package tmpl exposes embedded setup/init file templates.
//
// Sources are copies of install/templates/*, .fabrik runners, and skills/*.md.
// Sync tests guard against drift; re-copy on change.
package tmpl

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed workflow-note.md opencode.json fabrik/* agents/*.md hub/* skills/*.md
var files embed.FS

// Read returns the embedded file at path (e.g. "fabrik/config.yaml").
func Read(path string) (string, error) {
	b, err := files.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", path, err)
	}
	return string(b), nil
}

// MustRead panics on missing template (programmer error).
func MustRead(path string) string {
	s, err := Read(path)
	if err != nil {
		panic(err)
	}
	return s
}

// List returns embedded paths under dir.
func List(dir string) []string {
	var out []string
	_ = fs.WalkDir(files, dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// WorkflowNote is the personal workflow note for Cursor/OpenCode/Pi.
func WorkflowNote() string { return MustRead("workflow-note.md") }
