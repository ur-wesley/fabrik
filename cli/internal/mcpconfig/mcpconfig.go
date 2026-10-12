package mcpconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/apps"
)

func fabrikEntry() map[string]any {
	return map[string]any{
		"command": "fabrik",
		"args":    []any{"mcp"},
	}
}

func Register(selected []string, repoRoot string) error {
	for _, app := range selected {
		switch app {
		case apps.Cursor:
			if err := registerCursor(repoRoot); err != nil {
				return err
			}
		case apps.OpenCode:
			if err := registerOpenCode(repoRoot); err != nil {
				return err
			}
		case apps.Pi:
			if err := registerPi(); err != nil {
				return err
			}
		case apps.Antigravity:
			if err := registerAntigravity(); err != nil {
				return err
			}
		}
	}
	return nil
}

func registerCursor(repoRoot string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := patchCursorMCP(filepath.Join(home, ".cursor", "mcp.json")); err != nil {
		return err
	}
	if repoRoot != "" {
		return patchCursorMCP(filepath.Join(repoRoot, ".cursor", "mcp.json"))
	}
	return nil
}

func patchCursorMCP(path string) error {
	root, err := readJSONObject(path)
	if err != nil {
		if os.IsNotExist(err) {
			root = map[string]any{}
		} else {
			return err
		}
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	removeEngram(servers)
	servers["fabrik"] = fabrikEntry()
	root["mcpServers"] = servers
	return writeJSON(path, root)
}

func registerOpenCode(repoRoot string) error {
	path := filepath.Join(repoRoot, "opencode.json")
	root, err := readJSONObject(path)
	if err != nil {
		if os.IsNotExist(err) {
			root = map[string]any{}
		} else {
			return err
		}
	}
	mcp, _ := root["mcp"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
	}
	removeEngram(mcp)
	mcp["fabrik"] = fabrikEntry()
	root["mcp"] = mcp
	return writeJSON(path, root)
}

func registerPi() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return patchCursorMCP(filepath.Join(home, ".pi", "mcp.json"))
}

func registerAntigravity() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return patchCursorMCP(filepath.Join(home, ".gemini", "antigravity", "mcp.json"))
}

func removeEngram(servers map[string]any) {
	for k := range servers {
		if strings.EqualFold(k, "engram") {
			delete(servers, k)
		}
	}
}

func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, nil
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if root == nil {
		root = map[string]any{}
	}
	return root, nil
}

func writeJSON(path string, root map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}
