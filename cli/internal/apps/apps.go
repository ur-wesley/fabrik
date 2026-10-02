// Package apps manages supported AI coding applications and selection logic.
package apps

import (
	"fmt"
	"strings"
)

const (
	Cursor      = "cursor"
	Pi          = "pi"
	Antigravity = "antigravity"
	OpenCode    = "opencode"
)

// All lists all supported applications in canonical display order.
var All = []string{Cursor, Pi, Antigravity, OpenCode}

// DisplayName returns a human-friendly display name for an app.
func DisplayName(app string) string {
	switch strings.ToLower(strings.TrimSpace(app)) {
	case Cursor:
		return "Cursor"
	case Pi:
		return "Pi"
	case Antigravity:
		return "Antigravity"
	case OpenCode:
		return "OpenCode"
	default:
		return app
	}
}

// IsValid reports whether app is one of the supported applications.
func IsValid(app string) bool {
	switch strings.ToLower(strings.TrimSpace(app)) {
	case Cursor, Pi, Antigravity, OpenCode:
		return true
	default:
		return false
	}
}

// Normalize parses, cleans, validates, and deduplicates a slice of apps
// (which may include comma-separated values or "all").
// If raw is empty, all supported apps are returned.
func Normalize(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return append([]string(nil), All...), nil
	}
	var out []string
	seen := make(map[string]bool)
	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			p := strings.ToLower(strings.TrimSpace(part))
			if p == "" {
				continue
			}
			if p == "all" {
				return append([]string(nil), All...), nil
			}
			if !IsValid(p) {
				return nil, fmt.Errorf("invalid app %q: supported apps are %s", p, strings.Join(All, ", "))
			}
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	if len(out) == 0 {
		return append([]string(nil), All...), nil
	}
	return out, nil
}

// Contains reports whether app is present in apps.
func Contains(apps []string, app string) bool {
	target := strings.ToLower(strings.TrimSpace(app))
	for _, a := range apps {
		if strings.ToLower(strings.TrimSpace(a)) == target {
			return true
		}
	}
	return false
}

// FormatList formats a slice of app names into a friendly comma-separated string.
func FormatList(apps []string) string {
	var names []string
	for _, a := range apps {
		names = append(names, DisplayName(a))
	}
	return strings.Join(names, ", ")
}
