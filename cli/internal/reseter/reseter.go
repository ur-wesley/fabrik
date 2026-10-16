// Package reseter implements `fabrik reset` (un-init / deinit).
//
// It removes Fabrik's configuration and wiring from a project so client
// applications (Cursor, OpenCode, Pi) immediately revert to their default
// behavior. Deletes opencode.json (Fabrik agents), .cursor rules and agents,
// .opencode skills, .pi skills and agents, strips Fabrik workflow from
// AGENTS.md, and removes the entire .fabrik/ directory. Empty directories
// are pruned. Prompts to remove Beads integration as well unless --beads or
// --no-beads is passed.
package reseter

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
	"github.com/ur-wesley/fabrik/cli/internal/migrater"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
	"github.com/ur-wesley/fabrik/cli/internal/ui"
)

// Config mirrors the CLI flags.
type Config struct {
	RepoPath string
	Yes      bool
	DryRun   bool
	Beads    bool
	NoBeads  bool
}

// Deps are injectable seams.
type Deps struct {
	Exec    exec.Runner
	Confirm func(prompt string, defaultYes bool) bool
	Out     io.Writer
}

func (d *Deps) defaults() {
	if d.Exec == nil {
		d.Exec = exec.OSRunner{}
	}
	if d.Confirm == nil {
		d.Confirm = ui.Confirm
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
}

type AgentsMdAction int

const (
	AgentsMdNone AgentsMdAction = iota
	AgentsMdDelete
	AgentsMdUpdate
)

type OpencodeAction int

const (
	OpencodeNone OpencodeAction = iota
	OpencodeDelete
	OpencodeUpdate
)

// PlanResult holds all planned filesystem modifications.
type PlanResult struct {
	RepoDir       string
	FilesToDelete []string
	DirsToDelete  []string

	OpencodeAction  OpencodeAction
	OpencodePath    string
	OpencodeContent string

	AgentsMdAction  AgentsMdAction
	AgentsMdPath    string
	AgentsMdContent string

	RemoveBeads     bool
	BeadsDirPath    string
	BeadsLockPath   string
}

// Shim matching is canonical in migrater (migrater.IsSkillShim /
// migrater.IsAgentShim); reseter only adds filename-stem and marker checks
// around it so the two cannot drift apart.

func thinSkillNames() []string {
	var out []string
	for _, p := range tmpl.List("skills") {
		if base := strings.TrimSuffix(path.Base(p), ".md"); base != "" {
			out = append(out, base)
		}
	}
	return out
}

func fabrikSubagents() []string {
	pins, err := deps.Load()
	if err != nil {
		return nil
	}
	var out []string
	for _, a := range pins.Subagents {
		if a != "cta" {
			out = append(out, a)
		}
	}
	return out
}

func isFabrikSkill(filePath string, body []byte) bool {
	stem := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	for _, s := range thinSkillNames() {
		if stem == s {
			return true
		}
	}
	content := string(body)
	if migrater.IsSkillShim(content) {
		return true
	}
	if strings.Contains(content, "# Fabrik skill ") {
		return true
	}
	if strings.Contains(content, "description: Fabrik skill ") {
		return true
	}
	return false
}

func isFabrikAgent(filePath string, body []byte) bool {
	stem := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	for _, a := range fabrikSubagents() {
		if stem == a {
			return true
		}
	}
	content := string(body)
	if migrater.IsAgentShim(content) {
		return true
	}
	if strings.Contains(content, "# Fabrik subagent ") || strings.Contains(content, "# Fabrik agent ") {
		return true
	}
	if strings.Contains(content, "description: Fabrik subagent ") {
		return true
	}
	return false
}

// newlineOf reports the newline style of body ("\r\n" when it contains CRLF,
// "\n" otherwise) so rewrites preserve the original endings.
func newlineOf(body string) string {
	if strings.Contains(body, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

// splitLines splits body into lines, normalizing CRLF to LF for processing.
// The second return is the original newline for re-joining, so CRLF files
// round-trip byte-identical outside the removed sections.
func splitLines(body string) ([]string, string) {
	nl := newlineOf(body)
	if nl == "\r\n" {
		body = strings.ReplaceAll(body, "\r\n", "\n")
	}
	return strings.Split(body, "\n"), nl
}

func stripFabrikWorkflow(body string) string {
	lines, nl := splitLines(body)
	var result []string
	inFabrik := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if strings.HasPrefix(trimmed, "## Fabrik workflow") {
			inFabrik = true
			continue
		}
		if inFabrik {
			if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "# ") {
				inFabrik = false
				result = append(result, line)
			}
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, nl)
}

func stripBeadsBlock(body string) string {
	lines, nl := splitLines(body)
	var result []string
	inBeads := false
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if strings.Contains(trimmed, "BEGIN BEADS INTEGRATION") {
			inBeads = true
			continue
		}
		if inBeads {
			if strings.Contains(trimmed, "END BEADS INTEGRATION") {
				inBeads = false
			}
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, nl)
}

func stripAgentsMd(body string, removeBeads bool) (action AgentsMdAction, newContent string) {
	modified := body
	hasFabrik := strings.Contains(modified, "Fabrik workflow")
	if hasFabrik {
		modified = stripFabrikWorkflow(modified)
	}

	hasBeads := strings.Contains(modified, "BEGIN BEADS INTEGRATION")
	if removeBeads && hasBeads {
		modified = stripBeadsBlock(modified)
	}

	if !hasFabrik && (!removeBeads || !hasBeads) {
		return AgentsMdNone, body
	}

	trimmed := strings.TrimSpace(modified)
	if trimmed == "" || trimmed == "# Agent instructions" || trimmed == "# Agent Instructions" {
		return AgentsMdDelete, ""
	}

	return AgentsMdUpdate, strings.TrimRight(modified, "\r\n") + newlineOf(modified)
}

func planOpencode(filePath string) (OpencodeAction, string) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return OpencodeNone, ""
	}

	if !looksLikeFabrikOpencode(string(raw)) {
		return OpencodeNone, ""
	}

	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return OpencodeDelete, ""
	}

	agentsRaw, ok := root["agent"].(map[string]interface{})
	if !ok {
		return OpencodeDelete, ""
	}

	fabrikAgents := map[string]bool{}
	for _, a := range fabrikSubagents() {
		fabrikAgents[a] = true
	}

	if !hasCustomOpencodeContent(root, agentsRaw, fabrikAgents) {
		return OpencodeDelete, ""
	}

	return stripFabrikOpencode(root, agentsRaw, fabrikAgents)
}

// looksLikeFabrikOpencode is the cheap marker pre-check before JSON parsing.
func looksLikeFabrikOpencode(content string) bool {
	return strings.Contains(content, "fabrik show agent") ||
		strings.Contains(content, "# Fabrik agent") ||
		strings.Contains(content, "# Fabrik subagent") ||
		strings.Contains(content, `"default_agent": "plan"`)
}

// hasCustomOpencodeContent reports whether the config holds anything beyond
// Fabrik-owned wiring (mcp.fabrik counts as Fabrik-owned; any other mcp
// server counts as custom).
func hasCustomOpencodeContent(root map[string]interface{}, agentsRaw map[string]interface{}, fabrikAgents map[string]bool) bool {
	for name := range agentsRaw {
		if !fabrikAgents[name] {
			return true
		}
	}
	for k, v := range root {
		if k == "$schema" || k == "default_agent" || k == "agent" {
			continue
		}
		if k == "mcp" {
			if mcpMap, ok := v.(map[string]interface{}); ok {
				onlyFabrik := true
				for mk := range mcpMap {
					if mk != "fabrik" {
						onlyFabrik = false
						break
					}
				}
				if onlyFabrik {
					continue
				}
			}
		}
		return true
	}
	return false
}

// stripFabrikOpencode removes Fabrik-owned agents/keys and re-encodes.
func stripFabrikOpencode(root map[string]interface{}, agentsRaw map[string]interface{}, fabrikAgents map[string]bool) (OpencodeAction, string) {
	for a := range fabrikAgents {
		delete(agentsRaw, a)
	}
	if def, ok := root["default_agent"].(string); ok && def == "plan" {
		delete(root, "default_agent")
	}
	// Strip Fabrik-owned mcp.fabrik wiring; drop mcp when empty.
	if mcpMap, ok := root["mcp"].(map[string]interface{}); ok {
		delete(mcpMap, "fabrik")
		if len(mcpMap) == 0 {
			delete(root, "mcp")
		}
	}

	updated, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return OpencodeDelete, ""
	}
	return OpencodeUpdate, string(updated) + "\n"
}

// Plan computes all files to delete, modify, or prune.
func Plan(repo string, cfg Config) (*PlanResult, error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}

	result := &PlanResult{
		RepoDir:       abs,
		RemoveBeads:   cfg.Beads,
		BeadsDirPath:  filepath.Join(abs, ".beads"),
		BeadsLockPath: filepath.Join(abs, ".beads.gate.lock"),
	}

	planOpencodeInto(result, abs)
	planSkillFilesInto(result, abs)
	planAgentFilesInto(result, abs)
	if result.RemoveBeads {
		planBeadsFilesInto(result, abs)
	}
	planAgentsMdInto(result, abs)
	planFabrikStateInto(result, abs)

	sort.Strings(result.FilesToDelete)
	sort.Strings(result.DirsToDelete)
	result.FilesToDelete = slices.Compact(result.FilesToDelete)

	return result, nil
}

func planOpencodeInto(result *PlanResult, abs string) {
	opencodePath := filepath.Join(abs, "opencode.json")
	if _, err := os.Stat(opencodePath); err == nil {
		action, updated := planOpencode(opencodePath)
		result.OpencodeAction = action
		result.OpencodePath = opencodePath
		result.OpencodeContent = updated
	}

	repoMCP := filepath.Join(abs, ".cursor", "mcp.json")
	if _, err := os.Stat(repoMCP); err == nil {
		result.FilesToDelete = append(result.FilesToDelete, repoMCP)
	}
}

func planSkillFilesInto(result *PlanResult, abs string) {
	for _, dir := range []string{
		filepath.Join(abs, ".cursor", "rules"),
		filepath.Join(abs, ".cursor", "rules", "agents"),
		filepath.Join(abs, ".opencode", "skills"),
		filepath.Join(abs, ".pi", "skills"),
	} {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(p))
			if ext != ".md" && ext != ".mdc" {
				return nil
			}
			body, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			if isFabrikSkill(p, body) {
				result.FilesToDelete = append(result.FilesToDelete, p)
			}
			return nil
		})
	}
}

func planAgentFilesInto(result *PlanResult, abs string) {
	for _, dir := range []string{
		filepath.Join(abs, ".cursor", "agents"),
		filepath.Join(abs, ".pi", "agent", "agents"),
		filepath.Join(abs, ".opencode", "agents"),
	} {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(p)) != ".md" {
				return nil
			}
			body, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			if isFabrikAgent(p, body) {
				result.FilesToDelete = append(result.FilesToDelete, p)
			}
			return nil
		})
	}
}

func planBeadsFilesInto(result *PlanResult, abs string) {
	beadsMdc := filepath.Join(abs, ".cursor", "rules", "beads.mdc")
	if _, err := os.Stat(beadsMdc); err == nil {
		result.FilesToDelete = append(result.FilesToDelete, beadsMdc)
	}
	hooksJSON := filepath.Join(abs, ".cursor", "hooks.json")
	if _, err := os.Stat(hooksJSON); err == nil {
		result.FilesToDelete = append(result.FilesToDelete, hooksJSON)
	}
}

func planAgentsMdInto(result *PlanResult, abs string) {
	agentsPath := filepath.Join(abs, "AGENTS.md")
	if body, err := os.ReadFile(agentsPath); err == nil {
		action, updated := stripAgentsMd(string(body), result.RemoveBeads)
		result.AgentsMdAction = action
		result.AgentsMdPath = agentsPath
		result.AgentsMdContent = updated
	}
}

func planFabrikStateInto(result *PlanResult, abs string) {
	fabrikDir := filepath.Join(abs, ".fabrik")
	if st, err := os.Stat(fabrikDir); err == nil && st.IsDir() {
		result.DirsToDelete = append(result.DirsToDelete,
			filepath.Join(fabrikDir, "dolt"),
			filepath.Join(fabrikDir, "export"),
		)
		for _, g := range []string{
			filepath.Join(fabrikDir, "graph.db"),
			filepath.Join(fabrikDir, "graph.db-wal"),
			filepath.Join(fabrikDir, "graph.db-shm"),
			filepath.Join(fabrikDir, "issues.db"),
			filepath.Join(fabrikDir, "issues.db-wal"),
			filepath.Join(fabrikDir, "issues.db-shm"),
		} {
			if _, err := os.Stat(g); err == nil {
				result.FilesToDelete = append(result.FilesToDelete, g)
			}
		}
	}
}

// isNonInteractive mirrors the conditions under which ui.Confirm auto-answers
// yes without prompting (CI, go test, piped stdin). Reset is destructive, so
// Reset requires explicit --yes in that case instead of auto-confirming.
func isNonInteractive() bool {
	if flag.Lookup("test.v") != nil || os.Getenv("CI") != "" {
		return true
	}
	if f, err := os.Stdin.Stat(); err == nil && (f.Mode()&os.ModeCharDevice) == 0 {
		return true
	}
	return false
}

// Reset removes Fabrik configuration from repoPath.
func Reset(ctx context.Context, cfg Config, d Deps) error {
	useDefaultConfirm := d.Confirm == nil
	d.defaults()
	out := d.Out

	abs, err := resolveRepoDir(cfg.RepoPath)
	if err != nil {
		return err
	}

	if err := confirmReset(cfg, d, useDefaultConfirm, abs); err != nil {
		return err
	}

	cfg.Beads = resolveRemoveBeads(cfg, d, abs)
	plan, err := Plan(abs, cfg)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		return executeDryRun(out, plan)
	}

	return executeReset(ctx, out, plan, d)
}

func resolveRepoDir(target string) (string, error) {
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		target = cwd
	}
	return filepath.Abs(target)
}

func confirmReset(cfg Config, d Deps, useDefaultConfirm bool, abs string) error {
	if cfg.DryRun || cfg.Yes {
		return nil
	}
	// Destructive: never rely on ui.Confirm's non-interactive auto-yes
	// (CI, go test, piped stdin). Require explicit --yes instead.
	if useDefaultConfirm && isNonInteractive() {
		return fmt.Errorf("reset requires --yes in non-interactive mode (no TTY); re-run with --yes to confirm deletion in %s", abs)
	}
	if !d.Confirm(fmt.Sprintf("Reset Fabrik configuration in %s?", abs), cfg.Yes) {
		return fmt.Errorf("reset declined")
	}
	return nil
}

func resolveRemoveBeads(cfg Config, d Deps, abs string) bool {
	if cfg.Beads || cfg.NoBeads || cfg.Yes || cfg.DryRun {
		return cfg.Beads
	}
	hasBeadsDir := exists(filepath.Join(abs, ".beads"))
	hasBeadsRule := exists(filepath.Join(abs, ".cursor", "rules", "beads.mdc"))
	if (hasBeadsDir || hasBeadsRule) && d.Confirm("Remove Beads integration and .beads as well?", false) {
		return true
	}
	return false
}

// dryRunPrinter carries the output state through the small dry-run sections.
type dryRunPrinter struct {
	out     io.Writer
	abs     string
	plan    *PlanResult
	actions bool
}

func executeDryRun(out io.Writer, plan *PlanResult) error {
	p := &dryRunPrinter{out: out, abs: plan.RepoDir, plan: plan}
	p.printOpencode()
	p.printFiles()
	p.printAgentsMd()
	p.printDirs()
	p.printBeads()

	if !p.actions {
		fmt.Fprintln(out, "reset: nothing to remove")
		return nil
	}

	fmt.Fprintln(out, "would prune empty directories (.cursor, .opencode, .pi)")
	return nil
}

func (p *dryRunPrinter) printOpencode() {
	if p.plan.OpencodeAction == OpencodeDelete {
		rel, _ := filepath.Rel(p.abs, p.plan.OpencodePath)
		fmt.Fprintf(p.out, "would delete %s\n", filepath.ToSlash(rel))
		p.actions = true
	} else if p.plan.OpencodeAction == OpencodeUpdate {
		rel, _ := filepath.Rel(p.abs, p.plan.OpencodePath)
		fmt.Fprintf(p.out, "would remove Fabrik agents from %s\n", filepath.ToSlash(rel))
		p.actions = true
	}
}

func (p *dryRunPrinter) printFiles() {
	for _, f := range p.plan.FilesToDelete {
		rel, _ := filepath.Rel(p.abs, f)
		fmt.Fprintf(p.out, "would delete %s\n", filepath.ToSlash(rel))
		p.actions = true
	}
}

func (p *dryRunPrinter) printAgentsMd() {
	if p.plan.AgentsMdAction == AgentsMdDelete {
		rel, _ := filepath.Rel(p.abs, p.plan.AgentsMdPath)
		fmt.Fprintf(p.out, "would delete %s\n", filepath.ToSlash(rel))
		p.actions = true
	} else if p.plan.AgentsMdAction == AgentsMdUpdate {
		rel, _ := filepath.Rel(p.abs, p.plan.AgentsMdPath)
		fmt.Fprintf(p.out, "would strip Fabrik workflow from %s\n", filepath.ToSlash(rel))
		p.actions = true
	}
}

func (p *dryRunPrinter) printDirs() {
	for _, dir := range p.plan.DirsToDelete {
		rel, _ := filepath.Rel(p.abs, dir)
		fmt.Fprintf(p.out, "would delete %s\n", filepath.ToSlash(rel))
		p.actions = true
	}
}

func (p *dryRunPrinter) printBeads() {
	if !p.plan.RemoveBeads {
		return
	}
	if exists(p.plan.BeadsDirPath) {
		fmt.Fprintln(p.out, "would delete .beads (entire directory)")
		p.actions = true
	}
	if exists(p.plan.BeadsLockPath) {
		fmt.Fprintln(p.out, "would delete .beads.gate.lock")
		p.actions = true
	}
	fmt.Fprintln(p.out, "would run bd setup cursor --remove and bd setup opencode --remove")
	p.actions = true
}

func executeReset(ctx context.Context, out io.Writer, plan *PlanResult, d Deps) error {
	a := &resetApplier{ctx: ctx, out: out, plan: plan, d: d}
	a.applyOpencode()
	a.deletePlannedFiles()
	a.applyAgentsMd()
	a.removeFabrikState()
	a.removeBeadsWiring()
	a.pruneEmptyDirs()

	if len(a.errs) > 0 {
		return errors.Join(a.errs...)
	}

	if a.deleted == 0 && plan.OpencodeAction == OpencodeNone && plan.AgentsMdAction == AgentsMdNone {
		fmt.Fprintln(out, "reset: nothing to remove")
		return nil
	}

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Done. Fabrik configuration removed. Client apps reverted to default behaviour.")
	return nil
}

// resetApplier carries the mutable execution state (deleted count, errors)
// through the small apply steps below so executeReset stays a flat
// step list with behavior identical to the former monolith.
type resetApplier struct {
	ctx     context.Context
	out     io.Writer
	plan    *PlanResult
	d       Deps
	deleted int
	errs    []error
}

func (a *resetApplier) applyOpencode() {
	plan := a.plan
	if plan.OpencodeAction == OpencodeDelete {
		if err := os.Remove(plan.OpencodePath); err == nil || os.IsNotExist(err) {
			fmt.Fprintln(a.out, "deleted opencode.json")
			a.deleted++
		} else {
			a.errs = append(a.errs, fmt.Errorf("remove opencode.json: %w", err))
		}
	} else if plan.OpencodeAction == OpencodeUpdate {
		if err := os.WriteFile(plan.OpencodePath, []byte(plan.OpencodeContent), fileutil.ConfigPerm); err == nil {
			fmt.Fprintln(a.out, "updated opencode.json (removed Fabrik agents)")
		} else {
			a.errs = append(a.errs, fmt.Errorf("write opencode.json: %w", err))
		}
	}
	if err := stripOpencodeFabrikMCP(plan.OpencodePath); err != nil {
		a.errs = append(a.errs, err)
	}
}

func (a *resetApplier) deletePlannedFiles() {
	abs := a.plan.RepoDir
	for _, f := range a.plan.FilesToDelete {
		rel, _ := filepath.Rel(abs, f)
		if err := os.Remove(f); err == nil || os.IsNotExist(err) {
			fmt.Fprintf(a.out, "deleted %s\n", filepath.ToSlash(rel))
			a.deleted++
		} else {
			a.errs = append(a.errs, fmt.Errorf("remove %s: %w", filepath.ToSlash(rel), err))
		}
	}
}

func (a *resetApplier) applyAgentsMd() {
	plan := a.plan
	if plan.AgentsMdAction == AgentsMdDelete {
		if err := os.Remove(plan.AgentsMdPath); err == nil || os.IsNotExist(err) {
			fmt.Fprintln(a.out, "deleted AGENTS.md")
			a.deleted++
		} else {
			a.errs = append(a.errs, fmt.Errorf("remove AGENTS.md: %w", err))
		}
	} else if plan.AgentsMdAction == AgentsMdUpdate {
		if err := os.WriteFile(plan.AgentsMdPath, []byte(plan.AgentsMdContent), fileutil.ConfigPerm); err == nil {
			fmt.Fprintln(a.out, "updated AGENTS.md (removed Fabrik workflow)")
		} else {
			a.errs = append(a.errs, fmt.Errorf("write AGENTS.md: %w", err))
		}
	}
}

func (a *resetApplier) removeFabrikState() {
	abs := a.plan.RepoDir
	for _, dir := range a.plan.DirsToDelete {
		rel, _ := filepath.Rel(abs, dir)
		if err := os.RemoveAll(dir); err == nil || os.IsNotExist(err) {
			fmt.Fprintf(a.out, "deleted %s\n", filepath.ToSlash(rel))
			a.deleted++
		} else {
			a.errs = append(a.errs, fmt.Errorf("remove %s: %w", filepath.ToSlash(rel), err))
		}
	}
	if err := stripFabrikGitignore(abs, a.out); err != nil {
		a.errs = append(a.errs, err)
	}
}

func (a *resetApplier) removeBeadsWiring() {
	plan := a.plan
	if !plan.RemoveBeads {
		return
	}
	if _, err := a.d.Exec.LookPath("bd"); err == nil {
		if _, err := a.d.Exec.RunIn(a.ctx, plan.RepoDir, "bd", "setup", "cursor", "--remove"); err != nil {
			a.errs = append(a.errs, fmt.Errorf("bd setup cursor --remove: %w", err))
		}
		if _, err := a.d.Exec.RunIn(a.ctx, plan.RepoDir, "bd", "setup", "opencode", "--remove"); err != nil {
			a.errs = append(a.errs, fmt.Errorf("bd setup opencode --remove: %w", err))
		}
	}
	if exists(plan.BeadsDirPath) {
		if err := os.RemoveAll(plan.BeadsDirPath); err == nil {
			fmt.Fprintln(a.out, "deleted .beads/")
			a.deleted++
		} else {
			a.errs = append(a.errs, fmt.Errorf("remove .beads: %w", err))
		}
	}
	if exists(plan.BeadsLockPath) {
		if err := os.Remove(plan.BeadsLockPath); err == nil || os.IsNotExist(err) {
			a.deleted++
		} else {
			a.errs = append(a.errs, fmt.Errorf("remove .beads.gate.lock: %w", err))
		}
	}
}

func (a *resetApplier) pruneEmptyDirs() {
	abs := a.plan.RepoDir
	// Prune empty dirs (including .fabrik itself when only generated
	// state was removed and no hub content remains).
	pruneDirs := []string{
		filepath.Join(abs, ".cursor", "agents"),
		filepath.Join(abs, ".cursor", "rules", "agents"),
		filepath.Join(abs, ".cursor", "rules"),
		filepath.Join(abs, ".cursor"),
		filepath.Join(abs, ".opencode", "skills"),
		filepath.Join(abs, ".opencode", "agents"),
		filepath.Join(abs, ".opencode"),
		filepath.Join(abs, ".pi", "agent", "agents"),
		filepath.Join(abs, ".pi", "agent"),
		filepath.Join(abs, ".pi", "skills"),
		filepath.Join(abs, ".pi"),
		filepath.Join(abs, ".fabrik"),
	}
	for _, dir := range pruneDirs {
		removeIfEmpty(dir)
	}
}

func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) == 0 {
		_ = os.Remove(dir)
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

var fabrikGitignoreLines = []string{
	".fabrik/dolt/",
	".fabrik/graph.db",
	".fabrik/export/",
	".fabrik/issues.db",
	".fabrik/issues.db-wal",
	".fabrik/issues.db-shm",
}

func stripOpencodeFabrikMCP(path string) error {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read opencode.json: %w", err)
	}
	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}
	mcp, ok := root["mcp"].(map[string]interface{})
	if !ok {
		return nil
	}
	delete(mcp, "fabrik")
	if len(mcp) == 0 {
		delete(root, "mcp")
	}
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("encode opencode.json: %w", err)
	}
	if err := os.WriteFile(path, append(out, '\n'), fileutil.ConfigPerm); err != nil {
		return fmt.Errorf("write opencode.json: %w", err)
	}
	return nil
}

func stripFabrikGitignore(abs string, out io.Writer) error {
	path := filepath.Join(abs, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read .gitignore: %w", err)
	}
	lines, nl := splitLines(string(data))
	var keep []string
	removed := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if slices.Contains(fabrikGitignoreLines, trim) {
			removed = true
			continue
		}
		keep = append(keep, line)
	}
	if !removed {
		return nil
	}
	body := strings.TrimRight(strings.Join(keep, nl), "\r\n")
	if body != "" {
		body += nl
	}
	if err := os.WriteFile(path, []byte(body), fileutil.ConfigPerm); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	fmt.Fprintln(out, "updated .gitignore")
	return nil
}
