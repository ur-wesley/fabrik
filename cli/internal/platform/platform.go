// Package platform maps runtime OS/arch to deps.json asset keys and release
// asset names. It is the single source of truth for platform variants:
// keySuffix holds all 6 supported keys, ReleaseTargets the 5 fabrik binaries
// the release workflow builds (see .github/workflows/release-cli.yml matrix;
// windows_arm64 is a documented exclusion — no Windows/arm64 runner — not
// drift), and the asset-name helpers keep selfupdate and updatedeps from
// re-implementing the mapping.
package platform

import (
	"fmt"
	"runtime"
	"strings"
)

// keySuffix maps every supported platform key to its release asset suffix.
// It backs Key (all 6 resolve — third-party dep archives ship all 6),
// ReleaseTargets (the 5 fabrik binaries we build) and both asset-name
// helpers below.
var keySuffix = map[string]string{
	"windows_amd64": "windows-amd64",
	"windows_arm64": "windows-arm64",
	"linux_amd64":   "linux-amd64",
	"linux_arm64":   "linux-arm64",
	"darwin_amd64":  "darwin-amd64",
	"darwin_arm64":  "darwin-arm64",
}

// noFabrikBinary documents keys in keySuffix with no fabrik release binary.
// Third-party dep assets may still cover these keys.
var noFabrikBinary = map[string]bool{
	"windows_arm64": true,
}

// AllKeys lists every supported platform key, sorted for determinism.
var AllKeys = []string{"darwin_amd64", "darwin_arm64", "linux_amd64", "linux_arm64", "windows_amd64", "windows_arm64"}

// ReleaseTargets are the only OS/arch combos we ship fabrik binaries for:
// windows_amd64, linux_amd64, linux_arm64, darwin_amd64, darwin_arm64
// (see .github/workflows/release-cli.yml matrix).
var ReleaseTargets = []string{"windows_amd64", "linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64"}

// Key returns the deps.json asset key (e.g. "windows_amd64") for the given
// GOOS/GOARCH pair. All six supported combos resolve (third-party dep assets
// cover windows_arm64 even though fabrik ships no binary for it); anything
// else errors.
func Key(goos, goarch string) (string, error) {
	arch := goarch
	switch goarch {
	case "x86_64":
		arch = "amd64"
	case "aarch64":
		arch = "arm64"
	}
	key := goos + "_" + arch
	if _, ok := keySuffix[key]; ok {
		return key, nil
	}
	return "", fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
}

// Current returns the key for this process.
func Current() (string, error) {
	return Key(runtime.GOOS, runtime.GOARCH)
}

// Suffix returns the release asset suffix (e.g. "windows-amd64") for key.
func Suffix(key string) (string, error) {
	s, ok := keySuffix[key]
	if !ok {
		return "", fmt.Errorf("unsupported platform: %s", key)
	}
	return s, nil
}

// ArchiveExt is ".zip" for Windows keys, ".tar.gz" elsewhere.
func ArchiveExt(key string) string {
	if strings.HasPrefix(key, "windows") {
		return ".zip"
	}
	return ".tar.gz"
}

// DepAssetName builds a third-party dep archive name:
// <tool>_<ver>_<key>(.zip|.tar.gz).
func DepAssetName(tool, ver, key string) string {
	return fmt.Sprintf("%s_%s_%s%s", tool, ver, key, ArchiveExt(key))
}

// SelfUpdateAssetName maps a platform key to the fabrik release asset name.
// Keys with no fabrik binary (windows_arm64) error with the go-install
// fallback hint.
func SelfUpdateAssetName(key string) (string, error) {
	s, ok := keySuffix[key]
	if !ok || noFabrikBinary[key] {
		return "", fmt.Errorf("no self-update binary for %s — use: go install github.com/ur-wesley/fabrik/cli/cmd/fabrik@latest", key)
	}
	if strings.HasPrefix(key, "windows") {
		return "fabrik-" + s + ".exe", nil
	}
	return "fabrik-" + s, nil
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
