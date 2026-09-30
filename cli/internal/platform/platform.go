// Package platform maps runtime OS/arch to deps.json asset keys.
package platform

import (
	"fmt"
	"runtime"
)

// ReleaseTargets are the only OS/arch combos we ship binaries for:
// Windows + Linux on x86_64, macOS on arm64.
var ReleaseTargets = []string{"windows_amd64", "linux_amd64", "darwin_arm64"}

// Key returns the deps.json asset key (e.g. "windows_amd64") for the given
// GOOS/GOARCH pair. All six pinned combos resolve; release only ships three.
func Key(goos, goarch string) (string, error) {
	arch := goarch
	switch goarch {
	case "x86_64":
		arch = "amd64"
	case "aarch64":
		arch = "arm64"
	}
	switch goos {
	case "windows", "linux", "darwin":
		switch arch {
		case "amd64", "arm64":
			return goos + "_" + arch, nil
		}
	}
	return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
}

// Current returns the key for this process.
func Current() (string, error) {
	return Key(runtime.GOOS, runtime.GOARCH)
}

// BinaryName appends .exe on Windows.
func BinaryName(goos, base string) string {
	if goos == "windows" {
		return base + ".exe"
	}
	return base
}

// IsReleaseTarget reports whether key is one we ship.
func IsReleaseTarget(key string) bool {
	for _, t := range ReleaseTargets {
		if t == key {
			return true
		}
	}
	return false
}
