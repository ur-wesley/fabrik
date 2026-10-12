// Package flow ports .fabrik/fabrik.sh + loop.sh to Go.
//
// Run mirrors fabrik.sh (setup gate, then PRD-or-TUI, then plan + build).
// Loop mirrors loop.sh (poll ready issues in Dolt, opencode run with the embedded
// PROMPT, opt-in git push via --push). Output goes to stdout only; no state.md.
package flow

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/store"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

// prdPromptPrefix is the fixed PRD-generation prompt from fabrik.sh.
const prdPromptPrefix = "You are an expert product manager. Generate a detailed PRD in .fabrik/docs/PRD.md based on these requirements: "

// Deps are injectable seams (mirrors initer/installer).
type Deps struct {
	Exec  exec.Runner
	Out   io.Writer
	Sleep func(time.Duration)
}

func (d *Deps) defaults() {
	if d.Exec == nil {
		d.Exec = exec.OSRunner{}
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
	if d.Sleep == nil {
		d.Sleep = time.Sleep
	}
}

// RunConfig mirrors the fabrik.sh flags.
type RunConfig struct {
	RepoPath string
	Auto     bool
	Prompt   string
}

// LoopConfig mirrors the loop.sh args.
type LoopConfig struct {
	RepoPath string
	Mode     string // plan|build
	Max      int    // 0 = until no open/ready issues
	// AutoPush enables `git push` after each iteration. Default false:
	// never push without explicit user approval (--push).
	AutoPush bool
}

func resolveRepo(repo string) (string, error) {
	if repo == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		repo = cwd
	}
	return filepath.Abs(repo)
}

// CheckSetup gates on Fabrik v2 repo layout (.fabrik/dolt + config).
func CheckSetup(repo string) error {
	if _, ok := store.RepoRoot(repo); !ok {
		return fmt.Errorf("Fabrik setup incomplete: no store (run fabrik init first)")
	}
	if _, err := os.Stat(filepath.Join(repo, ".fabrik", "config.yaml")); err != nil {
		return fmt.Errorf("Fabrik setup incomplete: missing .fabrik/config.yaml (run fabrik init first)")
	}
	return nil
}

// Run executes the Fabrik workflow: setup gate, PRD-or-TUI, then plan + build.
func Run(ctx context.Context, cfg RunConfig, d Deps) error {
	d.defaults()
	repo, err := resolveRepo(cfg.RepoPath)
	if err != nil {
		return err
	}
	if err := CheckSetup(repo); err != nil {
		return err
	}
	fmt.Fprintln(d.Out, "Starting Fabrik workflow...")
	if cfg.Auto {
		if strings.TrimSpace(cfg.Prompt) == "" {
			return fmt.Errorf("--auto requires -p/--prompt with the initial requirements")
		}
		if _, err := d.Exec.RunIn(ctx, repo, "opencode", "run", prdPromptPrefix+cfg.Prompt); err != nil {
			return fmt.Errorf("opencode run PRD: %w", err)
		}
	} else {
		fmt.Fprintln(d.Out, "Step 1: /grill-with-docs, /to-prd, /exit")
		// TUI needs the real stdio: OSRunner.Run nulls stdin, so a
		// prompting child would fail fast instead of rendering.
		if _, err := d.Exec.RunInteractive(ctx, repo, "opencode"); err != nil {
			return fmt.Errorf("opencode TUI: %w", err)
		}
	}
	fmt.Fprintln(d.Out, "Step 2: Planning...")
	if err := Loop(ctx, LoopConfig{RepoPath: repo, Mode: "plan"}, d); err != nil {
		return err
	}
	// Planner -> builder pause (shared CTA agents/cta.md): never auto-start
	// the build wave. The user replies APPROVE to land + build, REVIEW
	// <note> to amend, STOP to pause.
	fmt.Fprintln(d.Out, "Step 3: Build wave paused for APPROVE (shared CTA agents/cta.md).")
	fmt.Fprintln(d.Out, "Reply APPROVE to land + start the build wave: fabrik loop build (no commit). REVIEW <note> amends. STOP pauses.")
	return nil
}

// validBranch reports whether name is a safe git branch for push.
func validBranch(name string) bool {
	if name == "" || strings.HasPrefix(name, "-") || strings.HasPrefix(name, ".") {
		return false
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, " \t\n\r~^:?*[]\\") {
		return false
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func currentBranch(ctx context.Context, r exec.Runner, repo string) string {
	out, err := r.RunIn(ctx, repo, "git", "branch", "--show-current")
	branch := strings.TrimSpace(out)
	if err != nil || branch == "" || !validBranch(branch) {
		return ""
	}
	return branch
}

// Loop polls ready issues in Dolt and drives opencode runs until no unfinished
// issues remain (build) or one pass completes (plan), capped by Max.
// All commands run in the repo dir (RepoPath, default cwd).
func Loop(ctx context.Context, cfg LoopConfig, d Deps) error {
	d.defaults()
	if cfg.Mode != "plan" && cfg.Mode != "build" {
		return fmt.Errorf("unknown mode %q: want plan|build", cfg.Mode)
	}
	repo, err := resolveRepo(cfg.RepoPath)
	if err != nil {
		return err
	}
	prompt, err := tmpl.Read("hub/PROMPT_" + cfg.Mode + ".md")
	if err != nil {
		return fmt.Errorf("prompt file hub/PROMPT_%s.md: %w", cfg.Mode, err)
	}
	if OpenLoopStore == nil {
		return fmt.Errorf("fabrik: loop store not configured")
	}
	branch := currentBranch(ctx, d.Exec, repo)
	iter := 0
	for {
		if cfg.Max > 0 && iter >= cfg.Max {
			fmt.Fprintf(d.Out, "Reached max iterations: %d\n", cfg.Max)
			break
		}
		issueStore, err := OpenLoopStore(ctx, repo)
		if err != nil {
			if err == store.ErrNoStore {
				return err
			}
			return fmt.Errorf("fabrik: store: %w", err)
		}
		unfinished, done, readyIssues, err := issueStore.Snapshot(ctx)
		if err != nil {
			_ = issueStore.Close()
			return fmt.Errorf("fabrik: store: %w", err)
		}

		nextID, nextDesc := "plan", "PRD gap analysis and issue create"
		if cfg.Mode == "build" {
			if unfinished == 0 {
				_ = issueStore.Close()
				fmt.Fprintln(d.Out, "No open issues.")
				break
			}
			if len(readyIssues) == 0 {
				_ = issueStore.Close()
				fmt.Fprintln(d.Out, "Open issues exist but none are ready.")
				break
			}
			claimed, err := issueStore.ClaimNext(ctx, "fabrik")
			if err != nil {
				_ = issueStore.Close()
				return fmt.Errorf("fabrik: claim: %w", err)
			}
			fmt.Fprintf(d.Out, "claim %s  %s\n", claimed.ID, claimed.Title)
			nextID, nextDesc = claimed.ID, claimed.Title
		}
		if err := issueStore.Close(); err != nil {
			return fmt.Errorf("fabrik: store: %w", err)
		}

		fmt.Fprintln(d.Out, "====================================================")
		fmt.Fprintf(d.Out, "FABRIK %s (iteration %d)\n", cfg.Mode, iter+1)
		fmt.Fprintf(d.Out, "Open: %d | Closed: %d | Ready: %d\n", unfinished, done, len(readyIssues))
		fmt.Fprintf(d.Out, "Target: %s — %s\n", nextID, nextDesc)
		fmt.Fprintln(d.Out, "====================================================")

		if _, err := d.Exec.RunIn(ctx, repo, "opencode", "run", prompt); err != nil {
			return fmt.Errorf("opencode run %s: %w", cfg.Mode, err)
		}
		if cfg.AutoPush {
			if branch == "" {
				fmt.Fprintln(d.Out, "Push skipped (unknown branch).")
			} else {
				// `--` guards against branch names parsed as flags; the
				// branch passed validation in currentBranch.
				_, _ = d.Exec.RunIn(ctx, repo, "git", "push", "origin", "--", branch) // best-effort, explicit opt-in only
			}
		} else {
			fmt.Fprintln(d.Out, "Push skipped (re-run with --push to push explicitly).")
		}

		iter++
		if cfg.Mode == "plan" {
			fmt.Fprintln(d.Out, "Planning complete. Reply APPROVE to land + start the build wave: fabrik loop build (no commit). REVIEW <note> amends. STOP pauses.")
			break
		}
		d.Sleep(3 * time.Second)
	}
	return nil
}
