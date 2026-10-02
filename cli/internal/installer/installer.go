// Package installer implements `fabrik setup` (port of install/setup.sh + setup.ps1).
package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/apps"
	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/download"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/paths"
	"github.com/ur-wesley/fabrik/cli/internal/platform"
	"github.com/ur-wesley/fabrik/cli/internal/repo"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
)

// Config mirrors the shell flags.
type Config struct {
	SkipToolInstall bool
	SkipEngramSetup bool
	SkipPiPackages  bool
	RepoPath        string
	Yes             bool // --yes: skip interactive confirms
	DryRun          bool // print actions only
	Apps            []string
}

// Deps are the injectable seams (fakes in tests).
type Deps struct {
	Exec exec.Runner
	Out  io.Writer
	// Download fetches a URL to a local file. Defaults to download.Get.
	Download func(ctx context.Context, url string) (string, error)
	// CopyGoBin copies the go-installed engram into BinDir. Defaults to real copy.
	CopyGoBin func(goExe, dest string) error
	// InitRepo runs per-repo init for --repo. Defaults to initer via exec of init logic.
	InitRepo func(repoPath string) error
	// Confirm asks a yes/no question. Defaults to ui.Confirm.
	Confirm func(question string, yes bool) bool
	// SelectApps prompts user for apps. Defaults to ui.SelectApps.
	SelectApps func(defaults []string, yes bool) ([]string, error)
}

func (d *Deps) defaults() {
	if d.Exec == nil {
		d.Exec = exec.OSRunner{}
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
	if d.Download == nil {
		d.Download = download.Get
	}
	if d.CopyGoBin == nil {
		d.CopyGoBin = copyFile
	}
	if d.Confirm == nil {
		d.Confirm = ui.Confirm
	}
	if d.SelectApps == nil {
		d.SelectApps = ui.SelectApps
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// Setup runs the machine setup.
func Setup(ctx context.Context, cfg Config, d Deps) error {
	d.defaults()
	pins, err := deps.Load()
	if err != nil {
		return err
	}
	out := d.Out

	var selectedApps []string
	if len(cfg.Apps) > 0 {
		var err error
		selectedApps, err = apps.Normalize(cfg.Apps)
		if err != nil {
			return err
		}
	} else if d.SelectApps != nil && !cfg.Yes {
		var err error
		selectedApps, err = d.SelectApps(apps.All, cfg.Yes)
		if err != nil {
			return err
		}
		selectedApps, err = apps.Normalize(selectedApps)
		if err != nil {
			return err
		}
	} else {
		selectedApps = append([]string(nil), apps.All...)
	}
	cfg.Apps = selectedApps

	fmt.Fprintf(out, "%s\n", ui.OK.Render("Fabrik machine setup"))
	fmt.Fprintf(out, "Deps: beads %s, engram %s, graphify %s\n", pins.Beads.Tag, pins.Engram.Tag, pins.Graphify.Version)
	fmt.Fprintf(out, "Apps: %s\n", apps.FormatList(selectedApps))

	if err := ensureBinDir(out, cfg.DryRun); err != nil {
		return err
	}
	if err := installBd(ctx, out, d, pins, cfg); err != nil {
		return err
	}
	if err := installEngram(ctx, out, d, pins, cfg); err != nil {
		return err
	}
	if err := initCurrentRepo(out, cfg, d); err != nil {
		return err
	}
	if err := installGraphify(ctx, out, d, pins, cfg, selectedApps); err != nil {
		return err
	}
	if err := installPiPackages(ctx, out, d, pins, cfg, selectedApps); err != nil {
		return err
	}
	if err := installWorkflowNote(out, cfg, selectedApps); err != nil {
		return err
	}
	if err := setupEngramAgents(ctx, out, d, cfg, selectedApps); err != nil {
		return err
	}
	if err := setupBeadsAgents(ctx, out, d, selectedApps); err != nil {
		return err
	}
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, ui.OK.Render("Done."))
	fmt.Fprintf(out, "Restart %s so MCP and rules reload.\n", apps.FormatList(selectedApps))
	return nil
}

func initCurrentRepo(out io.Writer, cfg Config, d Deps) error {
	repoPath := cfg.RepoPath
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repoPath = cwd
	}
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	if !repo.IsGitRepo(abs) {
		fmt.Fprintf(out, "%s\n", ui.Warn.Render("Not a git repository (no .git found). Run `git init` first — Fabrik/Beads work best in git repos."))
	}
	header(out, "Initializing repo: "+abs)
	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: init %s\n", abs)
		return nil
	}
	if d.InitRepo != nil {
		return d.InitRepo(abs)
	}
	return nil
}

func header(out io.Writer, text string) {
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, ui.Step.Render("==> "+text))
}

func ensureBinDir(out io.Writer, dry bool) error {
	bin, err := paths.BinDir()
	if err != nil {
		return err
	}
	if dry {
		fmt.Fprintf(out, "dry-run: mkdir %s\n", bin)
		return nil
	}
	return os.MkdirAll(bin, 0o755)
}

func hasBinary(d Deps, dest, name string) bool {
	if _, err := os.Stat(dest); err == nil {
		return true
	}
	_, err := d.Exec.LookPath(name)
	return err == nil
}

func installGithubArchive(ctx context.Context, out io.Writer, d Deps, repo, tag, asset, binary, label string, cfg Config) error {
	bin, err := paths.BinDir()
	if err != nil {
		return err
	}
	dest := filepath.Join(bin, platform.BinaryName(runtime.GOOS, binary))
	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(out, "%s already at %s\n", label, dest)
		return nil
	}
	if cfg.SkipToolInstall {
		return fmt.Errorf("%s missing and --skip-tool-install set", label)
	}
	if !d.Confirm(fmt.Sprintf("Install %s %s?", label, tag), cfg.Yes) {
		fmt.Fprintf(out, "Skipped %s\n", label)
		return nil
	}
	header(out, fmt.Sprintf("Installing %s %s", label, tag))
	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: download %s/releases/download/%s/%s -> %s\n", repo, tag, asset, dest)
		return nil
	}
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, asset)
	arc, err := d.Download(ctx, url)
	if err != nil {
		return err
	}
	defer os.Remove(arc)
	found, err := download.ExtractBinary(arc, platform.BinaryName(runtime.GOOS, binary))
	if err != nil {
		// archives sometimes omit .exe suffix on Windows payloads
		if alt, aerr := download.ExtractBinary(arc, binary); aerr == nil {
			found, err = alt, nil
		} else {
			return err
		}
	}
	defer os.RemoveAll(filepath.Dir(found))
	if err := copyFile(found, dest); err != nil {
		return err
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return err
	}
	fmt.Fprintf(out, "Installed %s\n", dest)
	return nil
}

func installBd(ctx context.Context, out io.Writer, d Deps, pins deps.Deps, cfg Config) error {
	if _, err := d.Exec.LookPath("bd"); err == nil {
		ver, _ := d.Exec.Run(ctx, "bd", "version")
		fmt.Fprintf(out, "bd already on PATH: %s\n", firstLine(ver))
		return nil
	}
	repo, tag, asset, binary, err := deps.Asset(pins, "beads")
	if err != nil {
		return err
	}
	return installGithubArchive(ctx, out, d, repo, tag, asset, binary, "Beads", cfg)
}

func installEngram(ctx context.Context, out io.Writer, d Deps, pins deps.Deps, cfg Config) error {
	if _, err := d.Exec.LookPath("engram"); err == nil {
		ver, _ := d.Exec.Run(ctx, "engram", "version")
		fmt.Fprintf(out, "engram already on PATH: %s\n", firstLine(ver))
		return nil
	}
	if cfg.SkipToolInstall {
		return fmt.Errorf("engram missing and --skip-tool-install set")
	}
	// Prefer `go install` when Go is available (shell parity).
	if _, err := d.Exec.LookPath("go"); err == nil {
		if !cfg.DryRun && d.Confirm(fmt.Sprintf("Install Engram %s via go install?", pins.Engram.Tag), cfg.Yes) {
			header(out, fmt.Sprintf("Installing Engram %s via go install", pins.Engram.Tag))
			module := fmt.Sprintf("%s@%s", pins.Engram.GoModule, pins.Engram.Tag)
			if _, err := d.Exec.Run(ctx, "go", "install", module); err == nil {
				gopath, gerr := d.Exec.Run(ctx, "go", "env", "GOPATH")
				if gerr == nil {
					goExe := filepath.Join(strings.TrimSpace(gopath), "bin", platform.BinaryName(runtime.GOOS, pins.Engram.Binary))
					bin, berr := paths.BinDir()
					if berr == nil {
						dest := filepath.Join(bin, platform.BinaryName(runtime.GOOS, pins.Engram.Binary))
						if _, serr := os.Stat(goExe); serr == nil {
							if cerr := d.CopyGoBin(goExe, dest); cerr == nil {
								fmt.Fprintf(out, "Copied engram to %s\n", bin)
								return nil
							}
						}
					}
				}
			}
			// go install failed: fall through to archive install.
		} else if cfg.DryRun {
			header(out, fmt.Sprintf("Installing Engram %s via go install", pins.Engram.Tag))
			fmt.Fprintf(out, "dry-run: go install %s@%s\n", pins.Engram.GoModule, pins.Engram.Tag)
			return nil
		}
	}
	repo, tag, asset, binary, err := deps.Asset(pins, "engram")
	if err != nil {
		return err
	}
	return installGithubArchive(ctx, out, d, repo, tag, asset, binary, "Engram", cfg)
}

func installGraphify(ctx context.Context, out io.Writer, d Deps, pins deps.Deps, cfg Config, selectedApps []string) error {
	header(out, fmt.Sprintf("Installing Graphify skill (%s only)", apps.FormatList(selectedApps)))
	dests, err := paths.GraphifySkills(selectedApps...)
	if err != nil {
		return err
	}
	var missing []string
	for _, dst := range dests {
		if fileExists(dst) {
			fmt.Fprintf(out, "%s already present\n", dst)
			continue
		}
		missing = append(missing, dst)
	}
	if len(missing) > 0 {
		if cfg.DryRun {
			for _, dst := range missing {
				fmt.Fprintf(out, "dry-run: write %s\n", dst)
			}
		} else if !cfg.Yes {
			fmt.Fprintln(out, "Graphify skill files skipped (pass --yes to install missing)")
		} else {
			tmp, err := d.Download(ctx, pins.Graphify.SkillURL)
			if err != nil {
				return err
			}
			defer os.Remove(tmp)
			body, err := os.ReadFile(tmp)
			if err != nil {
				return err
			}
			for _, dst := range missing {
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(dst, body, 0o644); err != nil {
					return err
				}
				fmt.Fprintf(out, "Wrote %s\n", dst)
			}
		}
	}
	if _, err := d.Exec.LookPath("graphify"); err == nil {
		fmt.Fprintln(out, "graphify CLI already on PATH")
		return nil
	}
	if cfg.SkipToolInstall {
		return nil
	}
	if _, err := d.Exec.LookPath("uv"); err != nil {
		fmt.Fprintln(out, ui.Warn.Render(fmt.Sprintf("uv not found; install graphify CLI manually: uv tool install %s==%s", pins.Graphify.Pypi, pins.Graphify.Version)))
		return nil
	}
	if !d.Confirm(fmt.Sprintf("Install %s==%s with uv?", pins.Graphify.Pypi, pins.Graphify.Version), cfg.Yes) {
		return nil
	}
	header(out, fmt.Sprintf("Installing %s==%s with uv", pins.Graphify.Pypi, pins.Graphify.Version))
	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: uv tool install %s==%s\n", pins.Graphify.Pypi, pins.Graphify.Version)
		return nil
	}
	_, err = d.Exec.Run(ctx, "uv", "tool", "install", fmt.Sprintf("%s==%s", pins.Graphify.Pypi, pins.Graphify.Version))
	return err
}

func installPiPackages(ctx context.Context, out io.Writer, d Deps, pins deps.Deps, cfg Config, selectedApps []string) error {
	if cfg.SkipPiPackages || !apps.Contains(selectedApps, apps.Pi) {
		return nil
	}
	if _, err := d.Exec.LookPath("pi"); err != nil {
		fmt.Fprintln(out, "pi not on PATH; skip pi package install")
		return nil
	}
	if !d.Confirm("Install Pi MCP adapter?", cfg.Yes) {
		return nil
	}
	header(out, "Installing Pi MCP adapter")
	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: pi install %s\n", pins.Pi.McpAdapter)
		return nil
	}
	_, err := d.Exec.Run(ctx, "pi", "install", pins.Pi.McpAdapter)
	return err
}

func installWorkflowNote(out io.Writer, cfg Config, selectedApps []string) error {
	header(out, "Installing personal workflow note")
	dests, err := paths.WorkflowNoteDests(selectedApps...)
	if err != nil {
		return err
	}
	note := tmpl.WorkflowNote()
	skippedHint := false
	for _, dst := range dests {
		if fileExists(dst.Path) {
			fmt.Fprintf(out, "%s already present\n", dst.Path)
			continue
		}
		if cfg.DryRun {
			fmt.Fprintf(out, "dry-run: write %s\n", dst.Path)
			continue
		}
		if !cfg.Yes {
			if !skippedHint {
				fmt.Fprintln(out, "Personal workflow notes skipped (pass --yes to write missing)")
				skippedHint = true
			}
			continue
		}
		body := note
		if dst.Frontmatter {
			body = paths.CursorFrontmatter + note
		}
		if err := os.MkdirAll(filepath.Dir(dst.Path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst.Path, []byte(body), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(out, "Wrote %s\n", dst.Path)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func setupEngramAgents(ctx context.Context, out io.Writer, d Deps, cfg Config, selectedApps []string) error {
	if cfg.SkipEngramSetup {
		return nil
	}
	if _, err := d.Exec.LookPath("engram"); err != nil {
		fmt.Fprintln(out, "Skipping engram setup")
		return nil
	}
	var targetApps []string
	for _, a := range selectedApps {
		if a == apps.Cursor || a == apps.Pi || a == apps.Antigravity {
			targetApps = append(targetApps, a)
		}
	}
	if len(targetApps) == 0 {
		return nil
	}
	if !d.Confirm(fmt.Sprintf("Run engram setup for %s?", apps.FormatList(targetApps)), cfg.Yes) {
		return nil
	}
	header(out, fmt.Sprintf("Running engram setup for %s (OpenCode: per-repo via fabrik init)", apps.FormatList(targetApps)))
	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: engram setup %s\n", strings.Join(targetApps, "|"))
		return nil
	}
	for _, app := range targetApps {
		binaryToCheck := app
		targetArg := app
		if app == apps.Antigravity {
			binaryToCheck = "agy"
			targetArg = "antigravity-cli"
		}
		if _, err := d.Exec.LookPath(binaryToCheck); err != nil {
			if app == apps.Antigravity {
				if _, err2 := d.Exec.LookPath("antigravity"); err2 != nil {
					fmt.Fprintf(out, "%s not on PATH; skipping engram setup for %s\n", binaryToCheck, app)
					continue
				}
			} else {
				fmt.Fprintf(out, "%s not on PATH; skipping engram setup for %s\n", binaryToCheck, app)
				continue
			}
		}
		// Best-effort like the shell scripts (|| true) and setupBeadsAgents.
		if _, err := d.Exec.Run(ctx, "engram", "setup", targetArg); err != nil {
			fmt.Fprintln(out, ui.Warn.Render(fmt.Sprintf("engram setup %s: %v", targetArg, err)))
		}
	}
	return nil
}

func setupBeadsAgents(ctx context.Context, out io.Writer, d Deps, selectedApps []string) error {
	if _, err := d.Exec.LookPath("bd"); err != nil {
		return nil
	}
	if !apps.Contains(selectedApps, apps.Cursor) {
		return nil
	}
	header(out, "Running bd setup (Cursor only; OpenCode: per-repo via fabrik init)")
	for _, app := range []string{"cursor"} {
		// Best-effort like the shell scripts (|| true).
		_, _ = d.Exec.Run(ctx, "bd", "setup", app)
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
