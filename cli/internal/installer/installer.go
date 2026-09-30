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

	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/download"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/paths"
	"github.com/ur-wesley/fabrik/cli/internal/platform"
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
	fmt.Fprintf(out, "%s\n", ui.OK.Render("Fabrik machine setup"))
	fmt.Fprintf(out, "Deps: beads %s, engram %s, graphify %s\n", pins.Beads.Tag, pins.Engram.Tag, pins.Graphify.Version)

	if err := ensureBinDir(out, cfg.DryRun); err != nil {
		return err
	}
	if err := installBd(ctx, out, d, pins, cfg); err != nil {
		return err
	}
	if err := installEngram(ctx, out, d, pins, cfg); err != nil {
		return err
	}
	if err := installGraphify(ctx, out, d, pins, cfg); err != nil {
		return err
	}
	if err := installPiPackages(ctx, out, d, pins, cfg); err != nil {
		return err
	}
	if err := installWorkflowNote(out, cfg); err != nil {
		return err
	}
	if err := setupEngramAgents(ctx, out, d, cfg); err != nil {
		return err
	}
	if err := setupBeadsAgents(ctx, out, d); err != nil {
		return err
	}
	if cfg.RepoPath != "" {
		header(out, "Initializing repo: "+cfg.RepoPath)
		if cfg.DryRun {
			fmt.Fprintf(out, "dry-run: init %s\n", cfg.RepoPath)
		} else if d.InitRepo != nil {
			if err := d.InitRepo(cfg.RepoPath); err != nil {
				return err
			}
		}
	}
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, ui.OK.Render("Done."))
	fmt.Fprintln(out, "Restart Cursor, OpenCode, and Pi so MCP and rules reload.")
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
	if !ui.Confirm(fmt.Sprintf("Install %s %s?", label, tag), cfg.Yes) {
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
		if !cfg.DryRun && ui.Confirm(fmt.Sprintf("Install Engram %s via go install?", pins.Engram.Tag), cfg.Yes) {
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

func installGraphify(ctx context.Context, out io.Writer, d Deps, pins deps.Deps, cfg Config) error {
	header(out, "Installing Graphify skill (Cursor, OpenCode, Pi only)")
	dests, err := paths.GraphifySkills()
	if err != nil {
		return err
	}
	if cfg.DryRun {
		for _, dst := range dests {
			fmt.Fprintf(out, "dry-run: write %s\n", dst)
		}
	} else {
		if !ui.Confirm("Download Graphify skill file?", cfg.Yes) {
			fmt.Fprintln(out, "Skipped Graphify skill")
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
			for _, dst := range dests {
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
	if !ui.Confirm(fmt.Sprintf("Install %s==%s with uv?", pins.Graphify.Pypi, pins.Graphify.Version), cfg.Yes) {
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

func installPiPackages(ctx context.Context, out io.Writer, d Deps, pins deps.Deps, cfg Config) error {
	if cfg.SkipPiPackages {
		return nil
	}
	if _, err := d.Exec.LookPath("pi"); err != nil {
		fmt.Fprintln(out, "pi not on PATH; skip pi package install")
		return nil
	}
	if !ui.Confirm("Install Pi MCP adapter?", cfg.Yes) {
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

func installWorkflowNote(out io.Writer, cfg Config) error {
	header(out, "Installing personal workflow note")
	dests, err := paths.WorkflowNoteDests()
	if err != nil {
		return err
	}
	note := tmpl.WorkflowNote()
	for _, dst := range dests {
		if cfg.DryRun {
			fmt.Fprintf(out, "dry-run: write %s\n", dst.Path)
			continue
		}
		if !ui.Confirm("Write "+dst.Path+"?", cfg.Yes) {
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

func setupEngramAgents(ctx context.Context, out io.Writer, d Deps, cfg Config) error {
	if cfg.SkipEngramSetup {
		return nil
	}
	if _, err := d.Exec.LookPath("engram"); err != nil {
		fmt.Fprintln(out, "Skipping engram setup")
		return nil
	}
	if !ui.Confirm("Run engram setup for Cursor, OpenCode, Pi?", cfg.Yes) {
		return nil
	}
	header(out, "Running engram setup for Cursor, OpenCode, Pi")
	if cfg.DryRun {
		fmt.Fprintln(out, "dry-run: engram setup cursor|opencode|pi")
		return nil
	}
	for _, app := range []string{"cursor", "opencode", "pi"} {
		if _, err := d.Exec.Run(ctx, "engram", "setup", app); err != nil {
			return fmt.Errorf("engram setup %s: %w", app, err)
		}
	}
	return nil
}

func setupBeadsAgents(ctx context.Context, out io.Writer, d Deps) error {
	if _, err := d.Exec.LookPath("bd"); err != nil {
		return nil
	}
	header(out, "Running bd setup (Cursor, OpenCode only)")
	for _, app := range []string{"cursor", "opencode"} {
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
