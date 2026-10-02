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
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
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
	RepoDir         string
	FilesToDelete   []string
	RemoveFabrikDir bool
	FabrikDirPath   string

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

var (
	skillShimRe = regexp.MustCompile(`Run: fabrik show skill [A-Za-z0-9_-]+`)
	agentShimRe = regexp.MustCompile(`Run: fabrik show agent [A-Za-z0-9_-]+`)
)

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
	if skillShimRe.MatchString(content) {
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
	if agentShimRe.MatchString(content) {
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

func stripFabrikWorkflow(body string) string {
	lines := strings.Split(body, "\n")
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
	return strings.Join(result, "\n")
}

func stripBeadsBlock(body string) string {
	lines := strings.Split(body, "\n")
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
	return strings.Join(result, "\n")
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

	return AgentsMdUpdate, strings.TrimRight(modified, "\r\n") + "\n"
}

func planOpencode(filePath string) (OpencodeAction, string) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return OpencodeNone, ""
	}

	content := string(raw)
	if !strings.Contains(content, "fabrik show agent") &&
		!strings.Contains(content, "# Fabrik agent") &&
		!strings.Contains(content, "# Fabrik subagent") &&
		!strings.Contains(content, `"default_agent": "plan"`) {
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

	hasCustomAgents := false
	for name := range agentsRaw {
		if !fabrikAgents[name] {
			hasCustomAgents = true
			break
		}
	}

	hasCustomTopKeys := false
	for k := range root {
		if k != "$schema" && k != "default_agent" && k != "agent" {
			hasCustomTopKeys = true
			break
		}
	}

	if !hasCustomAgents && !hasCustomTopKeys {
		return OpencodeDelete, ""
	}

	for a := range fabrikAgents {
		delete(agentsRaw, a)
	}
	if def, ok := root["default_agent"].(string); ok && def == "plan" {
		delete(root, "default_agent")
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

	// 1. opencode.json
	opencodePath := filepath.Join(abs, "opencode.json")
	if _, err := os.Stat(opencodePath); err == nil {
		action, updated := planOpencode(opencodePath)
		result.OpencodeAction = action
		result.OpencodePath = opencodePath
		result.OpencodeContent = updated
	}

	// 2. Cursor skills & agents
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

	// 3. Beads wiring (if RemoveBeads)
	if result.RemoveBeads {
		beadsMdc := filepath.Join(abs, ".cursor", "rules", "beads.mdc")
		if _, err := os.Stat(beadsMdc); err == nil {
			result.FilesToDelete = append(result.FilesToDelete, beadsMdc)
		}
		hooksJSON := filepath.Join(abs, ".cursor", "hooks.json")
		if _, err := os.Stat(hooksJSON); err == nil {
			result.FilesToDelete = append(result.FilesToDelete, hooksJSON)
		}
	}

	// 4. AGENTS.md
	agentsPath := filepath.Join(abs, "AGENTS.md")
	if body, err := os.ReadFile(agentsPath); err == nil {
		action, updated := stripAgentsMd(string(body), result.RemoveBeads)
		result.AgentsMdAction = action
		result.AgentsMdPath = agentsPath
		result.AgentsMdContent = updated
	}

	// 5. Entire .fabrik directory
	fabrikDir := filepath.Join(abs, ".fabrik")
	if st, err := os.Stat(fabrikDir); err == nil && st.IsDir() {
		result.RemoveFabrikDir = true
		result.FabrikDirPath = fabrikDir
	}

	sort.Strings(result.FilesToDelete)
	result.FilesToDelete = slices.Compact(result.FilesToDelete)

	return result, nil
}

// Reset removes Fabrik configuration from repoPath.
func Reset(ctx context.Context, cfg Config, d Deps) error {
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

	if !cfg.DryRun && !cfg.Yes {
		if !d.Confirm(fmt.Sprintf("Reset Fabrik configuration in %s?", abs), true) {
			return fmt.Errorf("reset declined")
		}
	}

	removeBeads := cfg.Beads
	if !cfg.Beads && !cfg.NoBeads && !cfg.Yes && !cfg.DryRun {
		hasBeadsDir := exists(filepath.Join(abs, ".beads"))
		hasBeadsRule := exists(filepath.Join(abs, ".cursor", "rules", "beads.mdc"))
		if hasBeadsDir || hasBeadsRule {
			if d.Confirm("Remove Beads integration and .beads as well?", false) {
				removeBeads = true
			}
		}
	}

	cfg.Beads = removeBeads
	plan, err := Plan(abs, cfg)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		return executeDryRun(out, plan)
	}

	return executeReset(ctx, out, plan, d)
}

func executeDryRun(out io.Writer, plan *PlanResult) error {
	abs := plan.RepoDir
	hasActions := false

	if plan.OpencodeAction == OpencodeDelete {
		rel, _ := filepath.Rel(abs, plan.OpencodePath)
		fmt.Fprintf(out, "would delete %s\n", filepath.ToSlash(rel))
		hasActions = true
	} else if plan.OpencodeAction == OpencodeUpdate {
		rel, _ := filepath.Rel(abs, plan.OpencodePath)
		fmt.Fprintf(out, "would remove Fabrik agents from %s\n", filepath.ToSlash(rel))
		hasActions = true
	}

	for _, f := range plan.FilesToDelete {
		rel, _ := filepath.Rel(abs, f)
		fmt.Fprintf(out, "would delete %s\n", filepath.ToSlash(rel))
		hasActions = true
	}

	if plan.AgentsMdAction == AgentsMdDelete {
		rel, _ := filepath.Rel(abs, plan.AgentsMdPath)
		fmt.Fprintf(out, "would delete %s\n", filepath.ToSlash(rel))
		hasActions = true
	} else if plan.AgentsMdAction == AgentsMdUpdate {
		rel, _ := filepath.Rel(abs, plan.AgentsMdPath)
		fmt.Fprintf(out, "would strip Fabrik workflow from %s\n", filepath.ToSlash(rel))
		hasActions = true
	}

	if plan.RemoveFabrikDir {
		fmt.Fprintln(out, "would delete .fabrik (entire directory)")
		hasActions = true
	}

	if plan.RemoveBeads {
		if exists(plan.BeadsDirPath) {
			fmt.Fprintln(out, "would delete .beads (entire directory)")
			hasActions = true
		}
		if exists(plan.BeadsLockPath) {
			fmt.Fprintln(out, "would delete .beads.gate.lock")
			hasActions = true
		}
		fmt.Fprintln(out, "would run bd setup cursor --remove and bd setup opencode --remove")
		hasActions = true
	}

	if !hasActions {
		fmt.Fprintln(out, "reset: nothing to remove")
		return nil
	}

	fmt.Fprintln(out, "would prune empty directories (.cursor, .opencode, .pi)")
	return nil
}

func executeReset(ctx context.Context, out io.Writer, plan *PlanResult, d Deps) error {
	abs := plan.RepoDir
	deletedCount := 0

	// 1. opencode.json
	if plan.OpencodeAction == OpencodeDelete {
		if err := os.Remove(plan.OpencodePath); err == nil {
			fmt.Fprintln(out, "deleted opencode.json")
			deletedCount++
		}
	} else if plan.OpencodeAction == OpencodeUpdate {
		if err := os.WriteFile(plan.OpencodePath, []byte(plan.OpencodeContent), 0o644); err == nil {
			fmt.Fprintln(out, "updated opencode.json (removed Fabrik agents)")
		}
	}

	// 2. files to delete (skills, agents, beads rules)
	for _, f := range plan.FilesToDelete {
		rel, _ := filepath.Rel(abs, f)
		if err := os.Remove(f); err == nil || os.IsNotExist(err) {
			fmt.Fprintf(out, "deleted %s\n", filepath.ToSlash(rel))
			deletedCount++
		}
	}

	// 3. AGENTS.md
	if plan.AgentsMdAction == AgentsMdDelete {
		if err := os.Remove(plan.AgentsMdPath); err == nil {
			fmt.Fprintln(out, "deleted AGENTS.md")
			deletedCount++
		}
	} else if plan.AgentsMdAction == AgentsMdUpdate {
		if err := os.WriteFile(plan.AgentsMdPath, []byte(plan.AgentsMdContent), 0o644); err == nil {
			fmt.Fprintln(out, "updated AGENTS.md (removed Fabrik workflow)")
		}
	}

	// 4. Remove entire .fabrik
	if plan.RemoveFabrikDir {
		if err := os.RemoveAll(plan.FabrikDirPath); err == nil {
			fmt.Fprintln(out, "deleted .fabrik/")
			deletedCount++
		}
	}

	// 5. Beads unwiring and removal
	if plan.RemoveBeads {
		if _, err := d.Exec.LookPath("bd"); err == nil {
			_, _ = d.Exec.RunIn(ctx, abs, "bd", "setup", "cursor", "--remove")
			_, _ = d.Exec.RunIn(ctx, abs, "bd", "setup", "opencode", "--remove")
		}
		if exists(plan.BeadsDirPath) {
			_ = os.RemoveAll(plan.BeadsDirPath)
			fmt.Fprintln(out, "deleted .beads/")
			deletedCount++
		}
		if exists(plan.BeadsLockPath) {
			_ = os.Remove(plan.BeadsLockPath)
			deletedCount++
		}
	}

	// 6. Prune empty dirs
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
	}
	for _, dir := range pruneDirs {
		removeIfEmpty(dir)
	}

	if deletedCount == 0 && plan.OpencodeAction == OpencodeNone && plan.AgentsMdAction == AgentsMdNone {
		fmt.Fprintln(out, "reset: nothing to remove")
		return nil
	}

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Done. Fabrik configuration removed. Client apps reverted to default behaviour.")
	return nil
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
