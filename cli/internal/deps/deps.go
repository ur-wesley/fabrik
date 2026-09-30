// Package deps loads the pinned tool versions (embedded deps.json).
package deps

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ur-wesley/fabrik/cli/internal/platform"
)

//go:embed deps.json
var raw []byte

// Tool is a GitHub-released binary with per-platform assets.
type Tool struct {
	Version string            `json:"version"`
	Tag     string            `json:"tag"`
	Repo    string            `json:"repo"`
	Binary  string            `json:"binary"`
	Assets  map[string]string `json:"assets"`
	// Engram-only: go install fallback module.
	GoModule string `json:"goModule,omitempty"`
}

// Graphify is a PyPI package plus a skill file URL.
type Graphify struct {
	Pypi     string `json:"pypi"`
	Version  string `json:"version"`
	SkillRef string `json:"skillRef"`
	SkillURL string `json:"skillUrl"`
}

// Pi holds Pi package references.
type Pi struct {
	McpAdapter string `json:"mcpAdapter"`
}

// Deps mirrors install/deps.json.
type Deps struct {
	Beads     Tool     `json:"beads"`
	Engram    Tool     `json:"engram"`
	Graphify  Graphify `json:"graphify"`
	Pi        Pi       `json:"pi"`
	Agents    []string `json:"agents"`
	Skills    []string `json:"skills"`
	Subagents []string `json:"subagents"`
}

// Load parses the embedded deps.json.
func Load() (Deps, error) {
	var d Deps
	if err := json.Unmarshal(raw, &d); err != nil {
		return d, fmt.Errorf("parse embedded deps.json: %w", err)
	}
	return d, nil
}

// Asset returns repo/tag/asset/binary for tool on the current platform.
func Asset(d Deps, tool string) (repo, tag, asset, binary string, err error) {
	var t Tool
	switch tool {
	case "beads":
		t = d.Beads
	case "engram":
		t = d.Engram
	default:
		return "", "", "", "", fmt.Errorf("unknown tool: %s", tool)
	}
	key, err := platform.Current()
	if err != nil {
		return "", "", "", "", err
	}
	a, ok := t.Assets[key]
	if !ok || a == "" {
		return "", "", "", "", fmt.Errorf("no %s asset for %s", tool, key)
	}
	return t.Repo, t.Tag, a, t.Binary, nil
}
