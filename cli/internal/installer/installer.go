// Package installer implements `fabrik setup`.
package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/apps"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/mcpconfig"
	"github.com/ur-wesley/fabrik/cli/internal/repo"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
)

type Config struct {
	SkipToolInstall bool
	SkipEngramSetup bool
	SkipPiPackages  bool
	RepoPath        string
	Yes             bool
	DryRun          bool
	Apps            []string
}

type Deps struct {
	Exec        exec.Runner
	Out         io.Writer
	InitRepo    func(repoPath string) error
	Confirm     func(question string, yes bool) bool
	SelectApps  func(defaults []string, yes bool) ([]string, error)
	EnsureMemory func(ctx context.Context) (string, error)
	RegisterMCP func(apps []string, repo string) error
}

func (d *Deps) defaults() {
	if d.Exec == nil {
		d.Exec = exec.OSRunner{}
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
	if d.Confirm == nil {
		d.Confirm = ui.Confirm
	}
	if d.SelectApps == nil {
		d.SelectApps = ui.SelectApps
	}
	if d.EnsureMemory == nil {
		d.EnsureMemory = ensureMemoryDB
	}
	if d.RegisterMCP == nil {
		d.RegisterMCP = mcpconfig.Register
	}
}

func ensureMemoryDB(ctx context.Context) (string, error) {
	path, err := store.MemoryDBPath()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if _, err := sqlmigrate.ApplySQLite(ctx, "memory", path); err != nil {
		return "", err
	}
	return path, nil
}

func Setup(ctx context.Context, cfg Config, d Deps) error {
	d.defaults()
	out := d.Out

	selectedApps, err := resolveApps(cfg, d)
	if err != nil {
		return err
	}
	cfg.Apps = selectedApps

	var memoryPath string
	if cfg.DryRun {
		p, err := store.MemoryDBPath()
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "dry-run: memory %s\n", p)
	} else {
		memoryPath, err = d.EnsureMemory(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "memory %s\n", memoryPath)
	}

	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: mcp register %s\n", strings.Join(selectedApps, ","))
	} else if err := d.RegisterMCP(selectedApps, ""); err != nil {
		return err
	} else {
		fmt.Fprintf(out, "mcp registered %s\n", strings.Join(selectedApps, ", "))
	}

	initOut, err := initCurrentRepo(out, cfg, d)
	if err != nil {
		return err
	}
	if initOut != "" {
		fmt.Fprint(out, initOut)
	}
	return nil
}

func resolveApps(cfg Config, d Deps) ([]string, error) {
	if len(cfg.Apps) > 0 {
		return apps.Normalize(cfg.Apps)
	}
	if d.SelectApps != nil && !cfg.Yes {
		selected, err := d.SelectApps(apps.All, cfg.Yes)
		if err != nil {
			return nil, err
		}
		return apps.Normalize(selected)
	}
	return apps.Normalize(nil)
}

func initCurrentRepo(out io.Writer, cfg Config, d Deps) (string, error) {
	repoPath := cfg.RepoPath
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		repoPath = cwd
	}
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	if !repo.IsGitRepo(abs) {
		fmt.Fprintf(out, "%s\n", ui.Warn.Render("Not a git repository (no .git found). Run `git init` first — Fabrik works best in git repos."))
	}
	if cfg.DryRun {
		return fmt.Sprintf("dry-run: init %s\n", abs), nil
	}
	if d.InitRepo != nil {
		if err := d.InitRepo(abs); err != nil {
			return "", err
		}
	}
	return "", nil
}
