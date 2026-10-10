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
	"path"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/apps"
	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/repo"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
)

// Config mirrors the shell flags.
type Config struct {
	RepoPath   string
	SkipChecks bool
	Yes        bool
	DryRun     bool
	// Full restores the legacy dump (one-release rollback). Default is thin.
	Full bool
	// Apps specifies which AI applications to configure (cursor, pi, antigravity, opencode).
	Apps []string
}

// Deps are injectable seams.
type Deps struct {
	Exec       exec.Runner
	Check      func(ctx context.Context) error // advisory tool check; nil skips
	Out        io.Writer
	SelectApps func(defaults []string, yes bool) ([]string, error)
}

func (d *Deps) defaults() {
	if d.Exec == nil {
		d.Exec = exec.OSRunner{}
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
	if d.SelectApps == nil {
		d.SelectApps = ui.SelectApps
	}
}

// Init initializes repoPath.
func Init(ctx context.Context, cfg Config, d Deps) error {
	d.defaults()
	out := d.Out
	target := cfg.RepoPath
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		target = cwd
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if !repo.IsGitRepo(abs) {
		fmt.Fprintf(out, "%s\n", ui.Warn.Render("Not a git repository (no .git found). Run `git init` first — Fabrik/Beads work best in git repos."))
	}

	if !cfg.SkipChecks && d.Check != nil {
		if err := d.Check(ctx); err != nil {
			fmt.Fprintf(out, "%s\n", ui.Warn.Render("(advisory: optional tools missing — continuing, bd is required)"))
		}
	}
	if _, err := d.Exec.LookPath("bd"); err != nil {
		return fmt.Errorf("bd not found on PATH (install Beads first)")
	}

	pins, err := deps.Load()
	if err != nil {
		return err
	}

	selectedApps, err := resolveApps(cfg, d, abs)
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

	// 2. Hub + skills + subagents: thin default, --full restores legacy dump.
	if cfg.Full {
		if err := initFull(out, cfg.DryRun, abs, pins.Subagents, selectedApps); err != nil {
			return err
		}
	} else {
		if err := initThin(out, cfg.DryRun, abs, pins.Subagents, selectedApps); err != nil {
			return err
		}
	}

	// 3. Per-repo agent wiring (best-effort), then strip side-effect dirs.
	hadAgents := exists(filepath.Join(abs, ".agents"))
	hadClaude := exists(filepath.Join(abs, ".claude"))
	hadCodex := exists(filepath.Join(abs, ".codex"))
	if cfg.DryRun {
		fmt.Fprintf(out, "dry-run: bd/engram setup for %s\n", strings.Join(selectedApps, "|"))
	} else {
		if _, err := d.Exec.LookPath("bd"); err == nil {
			if apps.Contains(selectedApps, apps.Cursor) {
				_, _ = d.Exec.RunIn(ctx, abs, "bd", "setup", "cursor")
			}
			if apps.Contains(selectedApps, apps.OpenCode) {
				_, _ = d.Exec.RunIn(ctx, abs, "bd", "setup", "opencode")
			}
		}
		if _, err := d.Exec.LookPath("engram"); err == nil {
			for _, app := range selectedApps {
				target := app
				if app == apps.Antigravity {
					target = "antigravity-cli"
				}
				_, _ = d.Exec.RunIn(ctx, abs, "engram", "setup", target)
			}
		}
	}
	if !cfg.DryRun {
		if !hadAgents && !apps.Contains(selectedApps, apps.Antigravity) {
			os.RemoveAll(filepath.Join(abs, ".agents"))
		}
		if !hadClaude {
			os.RemoveAll(filepath.Join(abs, ".claude"))
		}
		if !hadCodex {
			os.RemoveAll(filepath.Join(abs, ".codex"))
		}
	}

	// 4. Root AGENTS.md: thin pointer by default, full workflow note with --full.
	if cfg.Full {
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
	} else {
		agentsPath := filepath.Join(abs, "AGENTS.md")
		if body, err := os.ReadFile(agentsPath); err == nil {
			if !strings.Contains(string(body), "Fabrik workflow") {
				if err := appendFile(out, cfg.DryRun, agentsPath, agentsPointerAppend()); err != nil {
					return err
				}
				fmt.Fprintln(out, "Appended Fabrik workflow pointer to AGENTS.md")
			}
		} else {
			if err := write(out, cfg.DryRun, agentsPath, rootAgentsPointer()); err != nil {
				return err
			}
			fmt.Fprintln(out, "Created AGENTS.md")
		}
	}

	fmt.Fprintln(out, "")
	hub := ".fabrik/README.md"
	if !cfg.Full {
		hub = ".fabrik/config.yaml (content via `fabrik show <key>`)"
	}
	fmt.Fprintf(out, "Done. Apps: %s. Hub: %s\n", apps.FormatList(selectedApps), hub)
	fmt.Fprintln(out, "Next: grill, bd create issues, bd ready, claim, close.")
	return nil
}

func resolveApps(cfg Config, d Deps, abs string) ([]string, error) {
	if len(cfg.Apps) > 0 {
		return apps.Normalize(cfg.Apps)
	}
	configPath := filepath.Join(abs, ".fabrik", "config.yaml")
	if configApps := readConfigApps(configPath); len(configApps) > 0 {
		return apps.Normalize(configApps)
	}
	selected, err := d.SelectApps(apps.All, cfg.Yes)
	if err != nil {
		return nil, err
	}
	return apps.Normalize(selected)
}

func readConfigApps(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	lines := strings.Split(string(data), "\n")
	inApps := false
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "apps:") {
			inApps = true
			continue
		}
		if inApps {
			if strings.HasPrefix(trimmed, "- ") {
				app := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				if app != "" {
					out = append(out, app)
				}
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				break
			}
		}
	}
	return out
}

func writeConfigYaml(out io.Writer, dry bool, abs string, selectedApps []string) error {
	dst := filepath.Join(abs, ".fabrik", "config.yaml")
	if exists(dst) {
		return nil
	}
	baseTmpl, err := tmpl.Read("fabrik/config.yaml")
	if err != nil {
		return fmt.Errorf("template fabrik/config.yaml: %w", err)
	}
	rendered := renderConfigYaml(baseTmpl, selectedApps)
	return write(out, dry, dst, rendered)
}

func renderConfigYaml(baseTmpl string, selectedApps []string) string {
	var appLines []string
	for _, a := range selectedApps {
		appLines = append(appLines, fmt.Sprintf("  - %s", a))
	}
	appsBlock := "apps:\n" + strings.Join(appLines, "\n")

	if strings.Contains(baseTmpl, "apps:") {
		lines := strings.Split(baseTmpl, "\n")
		var newLines []string
		inApps := false
		replaced := false
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, "apps:") {
				inApps = true
				if !replaced {
					newLines = append(newLines, appsBlock)
					replaced = true
				}
				continue
			}
			if inApps {
				if strings.HasPrefix(trimmed, "- ") {
					continue
				}
				inApps = false
			}
			newLines = append(newLines, l)
		}
		return strings.Join(newLines, "\n")
	}
	return appsBlock + "\n\n" + baseTmpl
}

func antigravitySkillShim(s string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: Fabrik skill %s\n---\n\n# Fabrik skill %s\n\nRun: fabrik show skill %s\n", s, s, s, s)
}

// thinSkillNames derives the shimmed skills from the embedded templates
// (single source: tmpl.List), so new skills are picked up automatically.
func thinSkillNames() []string {
	var out []string
	for _, p := range tmpl.List("skills") {
		if base := strings.TrimSuffix(path.Base(p), ".md"); base != "" {
			out = append(out, base)
		}
	}
	return out
}

// skillShim is the 3-line shim body pointing at `fabrik show skill`.
func skillShim(s string) string {
	return "# Fabrik skill " + s + "\n\nRun: fabrik show skill " + s + "\n"
}

// agentDisplayName returns the uniform display name: first letter
// capitalized only, with style-smells shortened to Style.
func agentDisplayName(a string) string {
	if a == "style-smells" {
		return "Style"
	}
	if a == "" {
		return a
	}
	return strings.ToUpper(a[:1]) + a[1:]
}

// agentShim is the 3-line shim body pointing at `fabrik show agent`.
func agentShim(a string) string {
	return "# Fabrik subagent " + agentDisplayName(a) + "\n\nRun: fabrik show agent " + a + "\n"
}

type cursorAgentSpec struct {
	description string
	readonly    bool
}

// cursorAgentSpecs mirrors cli/internal/tmpl/opencode.json agent descriptions
// and readonly flags for Cursor subagent frontmatter.
var cursorAgentSpecs = map[string]cursorAgentSpec{
	"plan":          {"Explore and write discussable plan to .fabrik/specs/. No Beads, no code.", false},
	"orchestrator":  {"Fabrik execute router. After APPROVE: land + builder + tester.", true},
	"build":         {"Direct build approved spec or ready issue. No orchestrator overhead.", false},
	"explore":       {"Read-only codebase study. No edits.", true},
	"researcher":    {"External research: web, memory, docs. Read-only.", true},
	"briefer":       {"Writes docs/IDEA.md, docs/STACK.md, docs/BRAND.md. Stops before code.", false},
	"planner":       {"Post-APPROVE: land approved spec as Beads issues. No code.", false},
	"builder":       {"Implements one Beads issue. TDD, minimal diff.", false},
	"tester":        {"Runs tests, reports pass/fail. Minimal test fixes only.", false},
	"reviewer":      {"Diff review against spec. No implementation.", true},
	"style-smells":  {"Style and smells: YAGNI, KISS, DRY, SOLID. No implementation.", true},
	"security":      {"Security audit. Read-only, no fixes.", true},
	"backlog":       {"Backlog discover+plan gated land no build.", false},
}

func selectableAgents(subagents []string) []string {
	var out []string
	for _, a := range subagents {
		if a == "cta" {
			continue
		}
		out = append(out, a)
	}
	return out
}

func cursorAgentFile(name, body string) string {
	spec, ok := cursorAgentSpecs[name]
	if !ok {
		return fmt.Sprintf("---\nname: %s\ndescription: Fabrik subagent %s\nmodel: inherit\nreadonly: false\n---\n\n%s", name, agentDisplayName(name), body)
	}
	readonly := "false"
	if spec.readonly {
		readonly = "true"
	}
	return fmt.Sprintf("---\nname: %s\ndescription: %s\nmodel: inherit\nreadonly: %s\n---\n\n%s", name, spec.description, readonly, body)
}

func antigravityAgentFile(name, body string) string {
	spec, ok := cursorAgentSpecs[name]
	if !ok {
		return fmt.Sprintf("---\nname: %q\ndescription: %q\nmainAgent: true\nsubagent: true\n---\n\n%s", name, "Fabrik subagent "+agentDisplayName(name), body)
	}
	return fmt.Sprintf("---\nname: %q\ndescription: %q\nmainAgent: true\nsubagent: true\n---\n\n%s", name, spec.description, body)
}

// rootAgentsPointer is the 5-line AGENTS.md pointer written by thin init.
func rootAgentsPointer() string {
	return "# Agent instructions\n\n## Fabrik workflow\n\nRun `fabrik show workflow` for the full workflow.\n"
}

// agentsPointerAppend adds the pointer to an existing AGENTS.md.
func agentsPointerAppend() string {
	return "\n## Fabrik workflow\n\nRun `fabrik show workflow` for the full workflow.\n"
}

// initThin writes the thin default: config.yaml + docs/specs dirs + pointer
// AGENTS.md + 3-line shims for selected apps only.
func initThin(out io.Writer, dry bool, abs string, subagents []string, selectedApps []string) error {
	for _, dir := range []string{
		filepath.Join(abs, ".fabrik", "docs"),
		filepath.Join(abs, ".fabrik", "specs"),
	} {
		if err := mkdir(out, dry, dir); err != nil {
			return err
		}
	}
	if err := writeConfigYaml(out, dry, abs, selectedApps); err != nil {
		return err
	}
	if apps.Contains(selectedApps, apps.OpenCode) {
		if err := copyMissing(out, dry, "opencode.json", filepath.Join(abs, "opencode.json")); err != nil {
			return err
		}
	}
	for _, s := range thinSkillNames() {
		shim := skillShim(s)
		if apps.Contains(selectedApps, apps.Cursor) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".cursor", "rules", s+".mdc"), shim); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.OpenCode) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".opencode", "skills", s+".md"), shim); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.Pi) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".pi", "skills", s+".md"), shim); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.Antigravity) {
			agShim := antigravitySkillShim(s)
			if err := writeMissing(out, dry, filepath.Join(abs, ".agents", "skills", s, "SKILL.md"), agShim); err != nil {
				return err
			}
		}
	}
	for _, a := range selectableAgents(subagents) {
		shim := agentShim(a)
		if apps.Contains(selectedApps, apps.Pi) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".pi", "agent", "agents", a+".md"), shim); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.Cursor) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".cursor", "agents", a+".md"), cursorAgentFile(a, shim)); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.Antigravity) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".agents", "agents", a+".md"), antigravityAgentFile(a, shim)); err != nil {
				return err
			}
		}
	}
	return nil
}

// initFull restores the legacy dump (one-release rollback for --full).
func initFull(out io.Writer, dry bool, abs string, subagents []string, selectedApps []string) error {
	// 2. .fabrik hub
	for _, dir := range []string{"docs", "specs", "styleguide", "agents"} {
		if err := mkdir(out, dry, filepath.Join(abs, ".fabrik", dir)); err != nil {
			return err
		}
	}
	runners := []string{"PROMPT_plan.md", "PROMPT_build.md", "AGENTS.md"}
	for _, f := range runners {
		if err := copyMissing(out, dry, "hub/"+f, filepath.Join(abs, ".fabrik", f)); err != nil {
			return err
		}
	}
	for _, f := range []string{"README.md", "CONTEXT.md", "skills.md"} {
		if err := copyMissing(out, dry, "fabrik/"+f, filepath.Join(abs, ".fabrik", f)); err != nil {
			return err
		}
	}
	if err := writeConfigYaml(out, dry, abs, selectedApps); err != nil {
		return err
	}
	if err := copyMissing(out, dry, "fabrik/gitignore", filepath.Join(abs, ".fabrik", ".gitignore")); err != nil {
		return err
	}
	if err := copyMissing(out, dry, "hub/STYLEGUIDE.md", filepath.Join(abs, ".fabrik", "styleguide", "STYLEGUIDE.md")); err != nil {
		return err
	}
	// OpenCode opencode.json carries show-shims (no .opencode/agents copies).
	if apps.Contains(selectedApps, apps.OpenCode) {
		if err := copyMissing(out, dry, "opencode.json", filepath.Join(abs, "opencode.json")); err != nil {
			return err
		}
	}
	for _, a := range subagents {
		if _, err := tmpl.Read("agents/" + a + ".md"); err != nil {
			continue // shell parity: skip missing silently
		}
		if err := copyMissing(out, dry, "agents/"+a+".md", filepath.Join(abs, ".fabrik", "agents", a+".md")); err != nil {
			return err
		}
	}

	// 3. Skills -> selected apps
	if apps.Contains(selectedApps, apps.Cursor) {
		if err := mkdir(out, dry, filepath.Join(abs, ".cursor", "rules")); err != nil {
			return err
		}
	}
	if apps.Contains(selectedApps, apps.OpenCode) {
		if err := mkdir(out, dry, filepath.Join(abs, ".opencode", "skills")); err != nil {
			return err
		}
	}
	if apps.Contains(selectedApps, apps.Pi) {
		if err := mkdir(out, dry, filepath.Join(abs, ".pi", "skills")); err != nil {
			return err
		}
	}
	if apps.Contains(selectedApps, apps.Antigravity) {
		if err := mkdir(out, dry, filepath.Join(abs, ".agents", "skills")); err != nil {
			return err
		}
	}

	for _, s := range thinSkillNames() {
		body, err := tmpl.Read("skills/" + s + ".md")
		if err != nil {
			continue // shell parity: skip missing silently
		}
		if apps.Contains(selectedApps, apps.Cursor) {
			mdc := filepath.Join(abs, ".cursor", "rules", s+".mdc")
			if !exists(mdc) {
				wrapped := fmt.Sprintf("---\ndescription: Fabrik skill %s\nalwaysApply: false\n---\n\n%s", s, body)
				if err := write(out, dry, mdc, wrapped); err != nil {
					return err
				}
			}
		}
		if apps.Contains(selectedApps, apps.OpenCode) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".opencode", "skills", s+".md"), body); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.Pi) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".pi", "skills", s+".md"), body); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.Antigravity) {
			wrapped := fmt.Sprintf("---\nname: %s\ndescription: Fabrik skill %s\n---\n\n%s", s, s, body)
			if err := writeMissing(out, dry, filepath.Join(abs, ".agents", "skills", s, "SKILL.md"), wrapped); err != nil {
				return err
			}
		}
	}

	// 4. Subagents -> Pi + Cursor copies (OpenCode uses opencode.json show-shims)
	if apps.Contains(selectedApps, apps.Pi) {
		if err := mkdir(out, dry, filepath.Join(abs, ".pi", "agent", "agents")); err != nil {
			return err
		}
	}
	if apps.Contains(selectedApps, apps.Cursor) {
		if err := mkdir(out, dry, filepath.Join(abs, ".cursor", "agents")); err != nil {
			return err
		}
	}
	if apps.Contains(selectedApps, apps.Antigravity) {
		if err := mkdir(out, dry, filepath.Join(abs, ".agents", "agents")); err != nil {
			return err
		}
	}
	for _, a := range selectableAgents(subagents) {
		src, err := tmpl.Read("agents/" + a + ".md")
		if err != nil {
			continue
		}
		if apps.Contains(selectedApps, apps.Pi) {
			if err := writeMissing(out, dry, filepath.Join(abs, ".pi", "agent", "agents", a+".md"), src); err != nil {
				return err
			}
		}
		if apps.Contains(selectedApps, apps.Cursor) {
			cursorPath := filepath.Join(abs, ".cursor", "agents", a+".md")
			if !exists(cursorPath) {
				if err := write(out, dry, cursorPath, cursorAgentFile(a, src)); err != nil {
					return err
				}
			}
		}
		if apps.Contains(selectedApps, apps.Antigravity) {
			agPath := filepath.Join(abs, ".agents", "agents", a+".md")
			if !exists(agPath) {
				if err := write(out, dry, agPath, antigravityAgentFile(a, src)); err != nil {
					return err
				}
			}
		}
	}
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
	body, err := tmpl.Read(template)
	if err != nil {
		return fmt.Errorf("template %s: %w", template, err)
	}
	return writeMissing(out, dry, dst, body)
}

// writeMissing writes body to dst unless dst already exists (idempotent).
func writeMissing(out io.Writer, dry bool, dst, body string) error {
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
