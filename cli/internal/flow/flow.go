// Package flow ports .fabrik/fabrik.sh + loop.sh to Go.
//
// Run mirrors fabrik.sh (setup gate, then PRD-or-TUI, then plan + build).
// Loop mirrors loop.sh (poll bd ready/list, opencode run with the embedded
// PROMPT, opt-in git push via --push). Output goes to stdout only; no state.md.
package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/exec"
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

// CheckSetup mirrors the fabrik.sh gate: .beads dir + .fabrik/config.yaml.
func CheckSetup(repo string) error {
	if st, err := os.Stat(filepath.Join(repo, ".beads")); err != nil || !st.IsDir() {
		return fmt.Errorf("Fabrik setup incomplete: missing .beads (run fabrik init first)")
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

// issue is the bd JSON subset the loop needs.
type issue struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func parseIssues(raw string) []issue {
	var arr []issue
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		return arr
	}
	var wrapped struct {
		Issues []issue `json:"issues"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapped); err == nil {
		return wrapped.Issues
	}
	return nil
}

// bdList shells out to bd in the repo dir. Errors propagate: callers must
// surface them instead of treating a failed query as "no issues".
func bdList(ctx context.Context, r exec.Runner, repo string, args ...string) ([]issue, error) {
	out, err := r.RunIn(ctx, repo, "bd", args...)
	if err != nil {
		return nil, fmt.Errorf("bd %s: %w", strings.Join(args, " "), err)
	}
	return parseIssues(out), nil
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

// Loop polls bd ready/list and drives opencode runs until no open/ready
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
	if _, err := d.Exec.LookPath("bd"); err != nil {
		return fmt.Errorf("bd not found on PATH (install Beads first)")
	}
	branch := currentBranch(ctx, d.Exec, repo)
	iter := 0
	for {
		if cfg.Max > 0 && iter >= cfg.Max {
			fmt.Fprintf(d.Out, "Reached max iterations: %d\n", cfg.Max)
			break
		}
		openIssues, err := bdList(ctx, d.Exec, repo, "list", "--status=open", "--json")
		if err != nil {
			return err
		}
		closedIssues, err := bdList(ctx, d.Exec, repo, "list", "--status=closed", "--json", "--limit", "100")
		if err != nil {
			return err
		}
		readyIssues, err := bdList(ctx, d.Exec, repo, "ready", "--json")
		if err != nil {
			return err
		}

		nextID, nextDesc := "plan", "PRD gap analysis and bd create"
		if cfg.Mode == "build" {
			if len(openIssues) == 0 {
				fmt.Fprintln(d.Out, "No open Beads issues.")
				break
			}
			if len(readyIssues) == 0 {
				fmt.Fprintln(d.Out, "Open issues exist but none are ready.")
				break
			}
			nextID, nextDesc = readyIssues[0].ID, readyIssues[0].Title
		}

		fmt.Fprintln(d.Out, "====================================================")
		fmt.Fprintf(d.Out, "FABRIK %s (iteration %d)\n", cfg.Mode, iter+1)
		fmt.Fprintf(d.Out, "Open: %d | Closed: %d | Ready: %d\n", len(openIssues), len(closedIssues), len(readyIssues))
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
