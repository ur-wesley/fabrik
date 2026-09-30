// Package initer implements `fabrik init` (port of install/init.sh + init.ps1).
//
// Sets up the .fabrik overview hub plus Cursor/OpenCode/Pi wiring only.
// Never touches .claude/.codex/.agents except removing side-effects the
// bd/engram setup commands may create.
package initer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
)

// Config mirrors the shell flags.
type Config struct {
	RepoPath   string
	SkipChecks bool
	Yes        bool
	DryRun     bool
}

// Deps are injectable seams.
type Deps struct {
	Exec  exec.Runner
	Check func(ctx context.Context) error // advisory tool check; nil skips
	Out   io.Writer
}

func (d *Deps) defaults() {
	if d.Exec == nil {
		d.Exec = exec.OSRunner{}
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
}

func isWindows() bool { return runtime.GOOS == "windows" }

// Init initializes repoPath.
func Init(ctx context.Context, cfg Config, d Deps) error {
	d.defaults()
	out := d.Out
	repo := cfg.RepoPath
	if repo == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repo = cwd
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return err
	}

	if !cfg.SkipChecks && d.Check != nil {
		if err := d.Check(ctx); err != nil {
			fmt.Fprintf(out, "%s\n", ui.Warn.Render("(advisory: optional tools missing — continuing, bd is required)"))
		}
	}
	if _, err := d.Exec.LookPath("bd"); err != nil {
		return fmt.Errorf("bd not found. Run fabrik setup first")
	}

	pins, err := deps.Load()
	if err != nil {
		return err
	}

	// 1. Beads
	if st, err := os.Stat(filepath.Join(abs, ".beads")); err == nil && st.IsDir() {
		fmt.Fprintf(out, "Beads already initialized in %s\n", abs)
	} else {
		if !ui.Confirm("Run bd init here?", cfg.Yes) {
			return fmt.Errorf("bd init declined")
		}
		if cfg.DryRun {
			fmt.Fprintln(out, "dry-run: bd init --non-interactive --skip-agents --skip-hooks -q")
		} else {
			if _, err := d.Exec.RunIn(ctx, abs, "bd", "init", "--non-interactive", "--skip-agents", "--skip-hooks", "-q"); err != nil {
				return fmt.Errorf("bd init: %w", err)
			}
			fmt.Fprintln(out, "Beads initialized")
		}
	}

	// 2. .fabrik hub
	for _, dir := range []string{"docs", "specs", "styleguide", "agents"} {
		if err := mkdir(out, cfg.DryRun, filepath.Join(abs, ".fabrik", dir)); err != nil {
			return err
		}
	}
	runners := []string{"PROMPT_plan.md", "PROMPT_build.md", "loop.sh", "fabrik.sh", "AGENTS.md"}
	if isWindows() {
		runners = []string{"PROMPT_plan.md", "PROMPT_build.md", "loop.ps1", "fabrik.ps1", "AGENTS.md"}
	}
	for _, f := range runners {
		if err := copyMissing(out, cfg.DryRun, "hub/"+f, filepath.Join(abs, ".fabrik", f)); err != nil {
			return err
		}
	}
	for _, f := range []string{"README.md", "CONTEXT.md", "config.yaml", "skills.md"} {
		if err := copyMissing(out, cfg.DryRun, "fabrik/"+f, filepath.Join(abs, ".fabrik", f)); err != nil {
			return err
		}
	}
	if err := copyMissing(out, cfg.DryRun, "fabrik/gitignore", filepath.Join(abs, ".fabrik", ".gitignore")); err != nil {
		return err
	}
	if err := copyMissing(out, cfg.DryRun, "hub/STYLEGUIDE.md", filepath.Join(abs, ".fabrik", "styleguide", "STYLEGUIDE.md")); err != nil {
		return err
	}
	// OpenCode config replaces built-in build with Fabrik roster (file refs, no .opencode/agents copies).
	if err := copyMissing(out, cfg.DryRun, "opencode.json", filepath.Join(abs, "opencode.json")); err != nil {
		return err
	}
	for _, a := range pins.Subagents {
		if _, err := tmpl.Read("agents/" + a + ".md"); err != nil {
			continue // shell parity: skip missing silently
		}
		if err := copyMissing(out, cfg.DryRun, "agents/"+a+".md", filepath.Join(abs, ".fabrik", "agents", a+".md")); err != nil {
			return err
		}
	}

	// 3. Skills -> 3 apps only
	for _, dir := range []string{".cursor/rules", ".opencode/skills", ".pi/skills"} {
		if err := mkdir(out, cfg.DryRun, filepath.Join(abs, dir)); err != nil {
			return err
		}
	}
	for _, s := range []string{"i-have-adhd", "caveman", "ponytail", "rtk-usage", "tdd", "diagnose", "guardrails"} {
		body, err := tmpl.Read("skills/" + s + ".md")
		if err != nil {
			continue // shell parity: skip missing silently
		}
		mdc := filepath.Join(abs, ".cursor", "rules", s+".mdc")
		if !exists(mdc) {
			wrapped := fmt.Sprintf("---\ndescription: Fabrik skill %s\nalwaysApply: false\n---\n\n%s", s, body)
			if err := write(out, cfg.DryRun, mdc, wrapped); err != nil {
				return err
			}
		}
		if err := copyBodyMissing(out, cfg.DryRun, body, filepath.Join(abs, ".opencode", "skills", s+".md")); err != nil {
			return err
		}
		if err := copyBodyMissing(out, cfg.DryRun, body, filepath.Join(abs, ".pi", "skills", s+".md")); err != nil {
			return err
		}
	}

	// 4. Subagents -> Pi + Cursor copies (OpenCode uses opencode.json file refs)
	for _, dir := range []string{".pi/agent/agents", ".cursor/rules/agents"} {
		if err := mkdir(out, cfg.DryRun, filepath.Join(abs, dir)); err != nil {
			return err
		}
	}
	for _, a := range pins.Subagents {
		src, err := tmpl.Read("agents/" + a + ".md")
		if err != nil {
			continue
		}
		if err := copyBodyMissing(out, cfg.DryRun, src, filepath.Join(abs, ".pi", "agent", "agents", a+".md")); err != nil {
			return err
		}
		mdc := filepath.Join(abs, ".cursor", "rules", "agents", a+".mdc")
		if !exists(mdc) {
			wrapped := fmt.Sprintf("---\ndescription: Fabrik subagent %s\nalwaysApply: false\n---\n\n%s", a, src)
			if err := write(out, cfg.DryRun, mdc, wrapped); err != nil {
				return err
			}
		}
	}

	// 5. Per-repo agent wiring (best-effort), then strip side-effect dirs.
	hadAgents := exists(filepath.Join(abs, ".agents"))
	hadClaude := exists(filepath.Join(abs, ".claude"))
	hadCodex := exists(filepath.Join(abs, ".codex"))
	if cfg.DryRun {
		fmt.Fprintln(out, "dry-run: bd setup cursor|opencode, engram setup cursor|opencode|pi")
	} else {
		if _, err := d.Exec.LookPath("bd"); err == nil {
			_, _ = d.Exec.RunIn(ctx, abs, "bd", "setup", "cursor")
			_, _ = d.Exec.RunIn(ctx, abs, "bd", "setup", "opencode")
		}
		if _, err := d.Exec.LookPath("engram"); err == nil {
			for _, app := range []string{"cursor", "opencode", "pi"} {
				_, _ = d.Exec.RunIn(ctx, abs, "engram", "setup", app)
			}
		}
	}
	if !cfg.DryRun {
		if !hadAgents {
			os.RemoveAll(filepath.Join(abs, ".agents"))
		}
		if !hadClaude {
			os.RemoveAll(filepath.Join(abs, ".claude"))
		}
		if !hadCodex {
			os.RemoveAll(filepath.Join(abs, ".codex"))
		}
	}

	// 6. Root AGENTS.md workflow block.
	note := tmpl.WorkflowNote()
	agentsPath := filepath.Join(abs, "AGENTS.md")
	if body, err := os.ReadFile(agentsPath); err == nil {
		if !strings.Contains(string(body), "Fabrik workflow") {
			if err := appendFile(out, cfg.DryRun, agentsPath, "\n## Fabrik workflow\n\n"+note+"\n"); err != nil {
				return err
			}
			fmt.Fprintln(out, "Appended Fabrik workflow to AGENTS.md")
		}
	} else {
		if err := write(out, cfg.DryRun, agentsPath, "# Agent instructions\n\n## Fabrik workflow\n\n"+note+"\n"); err != nil {
			return err
		}
		fmt.Fprintln(out, "Created AGENTS.md")
	}

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Done. Apps: Cursor, OpenCode, Pi. Hub: .fabrik/README.md")
	fmt.Fprintln(out, "Next: grill, bd create issues, bd ready, claim, close.")
	return nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func mkdir(out io.Writer, dry bool, dir string) error {
	if dry {
		fmt.Fprintf(out, "dry-run: mkdir %s\n", dir)
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func copyMissing(out io.Writer, dry bool, template, dst string) error {
	if exists(dst) {
		return nil
	}
	body, err := tmpl.Read(template)
	if err != nil {
		return fmt.Errorf("template %s: %w", template, err)
	}
	return write(out, dry, dst, body)
}

func copyBodyMissing(out io.Writer, dry bool, body, dst string) error {
	if exists(dst) {
		return nil
	}
	return write(out, dry, dst, body)
}

func write(out io.Writer, dry bool, dst, body string) error {
	if dry {
		fmt.Fprintf(out, "dry-run: write %s\n", dst)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, []byte(body), 0o644)
}

func appendFile(out io.Writer, dry bool, dst, body string) error {
	if dry {
		fmt.Fprintf(out, "dry-run: append %s\n", dst)
		return nil
	}
	f, err := os.OpenFile(dst, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, body)
	return err
}
