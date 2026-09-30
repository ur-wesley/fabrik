// Package version reports the CLI version.
//
// Source of truth is cli/package.json (embedded fallback). Release builds
// stamp Version via ldflags from that same file:
//
//	go build -ldflags "-X github.com/ur-wesley/fabrik/cli/internal/version.Version=$(node -p "require('./package.json').version")"
package version

import (
	_ "embed"
	"encoding/json"
)

//go:embed package.json
var packageJSON []byte

// Version is stamped at build time. Empty means dev build.
var Version = ""

// Get returns the effective version: ldflags value, else cli/package.json, else "dev".
func Get() string {
	if Version != "" {
		return Version
	}
	var p struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(packageJSON, &p); err == nil && p.Version != "" {
		return p.Version
	}
	return "dev"
}
