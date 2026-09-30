// Package updatedeps implements `fabrik update-deps`
// (port of install/update-deps.sh + update-deps.ps1).
package updatedeps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/ui"
)

// Fetcher resolves latest versions. Production uses HTTP; tests stub it.
type Fetcher struct {
	// Client for API calls.
	Client *http.Client
	// LatestTag overrides GitHub tag lookup (tests).
	LatestTag map[string]string
	// PypiVersion overrides PyPI lookup (tests).
	PypiVersion map[string]string
}

func (f *Fetcher) client() *http.Client {
	if f.Client != nil {
		return f.Client
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (f *Fetcher) githubTag(ctx context.Context, repo string) (string, error) {
	if t, ok := f.LatestTag[repo]; ok {
		return t, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github %s: %s", repo, resp.Status)
	}
	var v struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}
	if v.TagName == "" {
		return "", fmt.Errorf("github %s: empty tag_name", repo)
	}
	return v.TagName, nil
}

func (f *Fetcher) pypiVersion(ctx context.Context, pkg string) (string, error) {
	if v, ok := f.PypiVersion[pkg]; ok {
		return v, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://pypi.org/pypi/"+pkg+"/json", nil)
	if err != nil {
		return "", err
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pypi %s: %s", pkg, resp.Status)
	}
	var v struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}
	if v.Info.Version == "" {
		return "", fmt.Errorf("pypi %s: empty version", pkg)
	}
	return v.Info.Version, nil
}

// Pins is the mutable subset of deps.json.
type Pins struct {
	Beads    ToolPin `json:"beads"`
	Engram   ToolPin `json:"engram"`
	Graphify struct {
		Pypi     string `json:"pypi"`
		Version  string `json:"version"`
		SkillRef string `json:"skillRef"`
		SkillURL string `json:"skillUrl"`
	} `json:"graphify"`
	Pi struct {
		McpAdapter string `json:"mcpAdapter"`
	} `json:"pi"`
	Agents    []string `json:"agents"`
	Skills    []string `json:"skills"`
	Subagents []string `json:"subagents"`
}

// ToolPin is one GitHub binary tool.
type ToolPin struct {
	Version  string            `json:"version"`
	Tag      string            `json:"tag"`
	Repo     string            `json:"repo"`
	Binary   string            `json:"binary"`
	Assets   map[string]string `json:"assets"`
	GoModule string            `json:"goModule,omitempty"`
}

func assetName(tool, ver, key string) string {
	ext := ".tar.gz"
	if strings.HasPrefix(key, "windows") {
		ext = ".zip"
	}
	return fmt.Sprintf("%s_%s_%s%s", tool, ver, key, ext)
}

// Refresh resolves latest pins and rewrites depsPath (JSON shape preserved generically).
func Refresh(ctx context.Context, out io.Writer, f *Fetcher, depsPath string, yes, dryRun bool) error {
	raw, err := os.ReadFile(depsPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", depsPath, err)
	}
	var pins Pins
	if err := json.Unmarshal(raw, &pins); err != nil {
		return fmt.Errorf("parse %s: %w", depsPath, err)
	}

	beadsTag, err := f.githubTag(ctx, pins.Beads.Repo)
	if err != nil {
		return err
	}
	engramTag, err := f.githubTag(ctx, pins.Engram.Repo)
	if err != nil {
		return err
	}
	graphifyVer, err := f.pypiVersion(ctx, pins.Graphify.Pypi)
	if err != nil {
		return err
	}
	beadsVer := strings.TrimPrefix(beadsTag, "v")
	engramVer := strings.TrimPrefix(engramTag, "v")

	keys := []string{"windows_amd64", "windows_arm64", "linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64"}

	fmt.Fprintf(out, "Latest pins:\n  beads    %s\n  engram   %s\n  graphify %s\n", beadsTag, engramTag, graphifyVer)
	if !ui.Confirm(fmt.Sprintf("Write pins to %s?", depsPath), yes) {
		return fmt.Errorf("declined")
	}
	if dryRun {
		fmt.Fprintf(out, "dry-run: no write to %s\n", depsPath)
		return nil
	}

	// Rewrite generically to preserve unknown fields and key order as much as
	// encoding/json allows (shell used python json.dump indent=2).
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	setTool(doc, "beads", beadsTag, beadsVer, pins.Beads.Binary, keys)
	setTool(doc, "engram", engramTag, engramVer, pins.Engram.Binary, keys)
	if g, ok := doc["graphify"].(map[string]any); ok {
		g["version"] = graphifyVer
	}
	out2, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out2 = append(out2, '\n')
	if err := os.WriteFile(depsPath, out2, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "Updated deps.json:\n  beads    %s\n  engram   %s\n  graphify %s\n", beadsTag, engramTag, graphifyVer)
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Run fabrik setup to install the new pins.")
	return nil
}

func setTool(doc map[string]any, name, tag, ver, _ string, keys []string) {
	t, ok := doc[name].(map[string]any)
	if !ok {
		return
	}
	t["version"] = ver
	t["tag"] = tag
	assets := map[string]any{}
	for _, k := range keys {
		assets[k] = assetName(name, ver, k)
	}
	t["assets"] = assets
}
