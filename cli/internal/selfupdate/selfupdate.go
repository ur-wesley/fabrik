// Package selfupdate implements `fabrik self-update`.
//
// Fail-closed: the release must ship a <asset>.sha256 file or the update
// is refused. Swap is rename-based so it works for running .exe files.
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/ur-wesley/fabrik/cli/internal/platform"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
	"github.com/ur-wesley/fabrik/cli/internal/version"
)

// Repo hosting the CLI releases.
const (
	Owner = "ur-wesley"
	Repo  = "fabrik"
)

// Config mirrors the CLI flags.
type Config struct {
	CheckOnly   bool   // --check: report only
	WantVersion string // --version: pin tag (default: latest)
	Yes         bool
	DryRun      bool
}

// Deps are injectable seams.
type Deps struct {
	Out io.Writer
	// APIBase overrides the GitHub API (tests). Default https://api.github.com.
	APIBase string
	// Client for API + download. Defaults to a 5-minute client.
	Client *http.Client
	// ExecPath is the running binary. Defaults to os.Executable.
	ExecPath string
	// PlatformKey resolves the asset suffix. Defaults to platform.Current.
	PlatformKey func() (string, error)
	// Current version. Defaults to version.Get().
	Current string
}

func (d *Deps) defaults() error {
	if d.Out == nil {
		d.Out = os.Stdout
	}
	if d.APIBase == "" {
		d.APIBase = "https://api.github.com"
	}
	if d.Client == nil {
		d.Client = &http.Client{Timeout: 5 * time.Minute}
	}
	if d.ExecPath == "" {
		p, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate executable: %w", err)
		}
		d.ExecPath = p
	}
	if d.PlatformKey == nil {
		d.PlatformKey = platform.Current
	}
	if d.Current == "" {
		d.Current = version.Get()
	}
	return nil
}

// Release is a GitHub release with asset download URLs by name.
type Release struct {
	Tag    string
	Assets map[string]string
}

func (d *Deps) latest(ctx context.Context, want string) (Release, error) {
	url := strings.TrimSuffix(d.APIBase, "/") + "/repos/" + Owner + "/" + Repo + "/releases/"
	if want != "" {
		url += "tags/" + want
	} else {
		url += "latest"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GET %s: status %s", url, resp.Status)
	}
	var v struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return Release{}, err
	}
	if v.TagName == "" {
		return Release{}, fmt.Errorf("release has empty tag_name")
	}
	r := Release{Tag: v.TagName, Assets: map[string]string{}}
	for _, a := range v.Assets {
		r.Assets[a.Name] = a.URL
	}
	return r, nil
}

// assetName maps a platform key to the release asset name.
func assetName(key string) (string, error) {
	switch key {
	case "windows_amd64":
		return "fabrik-windows-amd64.exe", nil
	case "linux_amd64":
		return "fabrik-linux-amd64", nil
	case "darwin_arm64":
		return "fabrik-darwin-arm64", nil
	default:
		return "", fmt.Errorf("no self-update binary for %s — use: go install github.com/ur-wesley/fabrik/cli/cmd/fabrik@latest", key)
	}
}

// newer reports whether want is newer than cur. Non-semver cur (dev builds)
// counts as outdated so dev binaries can self-update.
func newer(cur, want string) bool {
	cur, want = norm(cur), norm(want)
	if !semver.IsValid(cur) {
		return true
	}
	if !semver.IsValid(want) {
		return false
	}
	return semver.Compare(cur, want) < 0
}

func norm(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// Update resolves, verifies, and installs the update.
func Update(ctx context.Context, cfg Config, d Deps) error {
	if err := d.defaults(); err != nil {
		return err
	}
	out := d.Out

	rel, err := d.latest(ctx, cfg.WantVersion)
	if err != nil {
		return err
	}
	if !newer(d.Current, rel.Tag) {
		fmt.Fprintf(out, "fabrik %s is up to date (%s)\n", d.Current, rel.Tag)
		return nil
	}
	fmt.Fprintf(out, "Update available: %s -> %s\n", d.Current, rel.Tag)
	if cfg.CheckOnly {
		return nil
	}

	key, err := d.PlatformKey()
	if err != nil {
		return err
	}
	name, err := assetName(key)
	if err != nil {
		return err
	}
	assetURL, ok := rel.Assets[name]
	if !ok {
		return fmt.Errorf("release %s has no asset %s", rel.Tag, name)
	}
	sumURL, ok := rel.Assets[name+".sha256"]
	if !ok {
		return fmt.Errorf("release %s has no checksum %s — refusing to update", rel.Tag, name+".sha256")
	}

	if !ui.Confirm(fmt.Sprintf("Install %s?", rel.Tag), cfg.Yes) {
		return fmt.Errorf("declined")
	}
	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: download %s -> swap %s\n", assetURL, d.ExecPath)
		return nil
	}

	bin, err := fetch(ctx, d.Client, assetURL)
	if err != nil {
		return err
	}
	defer os.Remove(bin)
	sum, err := fetch(ctx, d.Client, sumURL)
	if err != nil {
		return err
	}
	defer os.Remove(sum)
	if err := verify(bin, sum); err != nil {
		return err
	}
	if err := swap(d.ExecPath, bin); err != nil {
		return err
	}
	fmt.Fprintf(out, "Updated to %s (previous binary kept at %s.old)\n", rel.Tag, d.ExecPath)
	return nil
}

func fetch(ctx context.Context, c *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %s", url, resp.Status)
	}
	f, err := os.CreateTemp("", "fabrik-update-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// verify checks the sha256sum file ("<hex>  <filename>") against bin.
func verify(bin, sumFile string) error {
	raw, err := os.ReadFile(sumFile)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return fmt.Errorf("empty checksum file")
	}
	want, err := hex.DecodeString(fields[0])
	if err != nil || len(want) != sha256.Size {
		return fmt.Errorf("malformed checksum file")
	}
	f, err := os.Open(bin)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if string(h.Sum(nil)) != string(want) {
		return fmt.Errorf("checksum mismatch — refusing to update")
	}
	return nil
}

// swap renames exe to exe.old and moves bin into place.
func swap(exe, bin string) error {
	old := exe + ".old"
	os.Remove(old) // stale rollback from a previous run
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := copyFile(bin, exe, 0o755); err != nil {
		// best-effort rollback
		_ = os.Rename(old, exe)
		return fmt.Errorf("install new binary: %w", err)
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
