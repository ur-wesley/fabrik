// Package migrater implements `fabrik migrate --prune [--dry-run]`.
//
// Default-deny: only exact known legacy generated paths are deleted
// (filename allowlist derived from embedded tmpl names + pinned
// deps.json skills/subagents). Custom user files in the same directories
// are never touched and are reported as kept. It never touches user
// content: config.yaml, docs/, specs/, user PRD, root AGENTS.md,
// opencode.json.
// Skill/agent copies containing an exact `fabrik show skill|agent <name>`
// shim marker line are kept; only full bodies are pruned.
package migrater

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/deps"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

// Config mirrors the CLI flags.
type Config struct {
	RepoPath string
	Prune    bool
	DryRun   bool
}

// Deps are injectable seams.
type Deps struct {
	Out io.Writer
	// BDList reads Beads issues when .beads/issues.jsonl is absent
	// (default: `bd list --json` scoped to the repo). Tests inject fakes
	// so no real `bd` binary is needed.
	BDList func(ctx context.Context, repo string) ([]byte, error)
}

func (d *Deps) defaults() {
	if d.Out == nil {
		d.Out = os.Stdout
	}
}

// Canonical thin-pointer marker lines produced by initer.skillShim /
// initer.agentShim (`Run: fabrik show skill <name>` /
// `Run: fabrik show agent <name>`). This is the single source of truth for
// shim matching, shared with reseter.isFabrikSkill/isFabrikAgent. Anchored
// to the full line so prose that merely mentions the command mid-sentence
// is still treated as a full body (pruned when allowlisted, never when
// custom).
var (
	skillShimRe = regexp.MustCompile(`(?m)^Run: fabrik show skill [A-Za-z0-9_-]+$`)
	agentShimRe = regexp.MustCompile(`(?m)^Run: fabrik show agent [A-Za-z0-9_-]+$`)
)

// IsShim reports whether body is a thin pointer shim (keep, never delete).
func IsShim(body string) bool {
	return IsSkillShim(body) || IsAgentShim(body)
}

// IsSkillShim reports whether body contains a skill shim marker line.
func IsSkillShim(body string) bool {
	return skillShimRe.MatchString(body)
}

// IsAgentShim reports whether body contains an agent shim marker line.
func IsAgentShim(body string) bool {
	return agentShimRe.MatchString(body)
}

// knownStems returns the allowlists of Fabrik-generated skill and agent
// filename stems: the union of embedded tmpl names and pinned deps.json
// names, so newly added skills/agents are covered without code changes.
func knownStems() (skills, agents map[string]bool) {
	skills = map[string]bool{}
	agents = map[string]bool{}
	for _, p := range tmpl.List("skills") {
		if s := strings.TrimSuffix(filepath.Base(p), ".md"); s != "" {
			skills[s] = true
		}
	}
	for _, p := range tmpl.List("agents") {
		if a := strings.TrimSuffix(filepath.Base(p), ".md"); a != "" {
			agents[a] = true
		}
	}
	if d, err := deps.Load(); err == nil {
		for _, s := range d.Skills {
			skills[s] = true
		}
		for _, a := range d.Subagents {
			agents[a] = true
		}
	}
	return skills, agents
}

// stemOf splits a filename into its stem and whether it is a rule/markdown
// file (.md/.mdc, any case).
func stemOf(base string) (stem string, ok bool) {
	lower := strings.ToLower(base)
	switch {
	case strings.HasSuffix(lower, ".mdc"):
		return base[:len(base)-4], true
	case strings.HasSuffix(lower, ".md"):
		return base[:len(base)-3], true
	default:
		return "", false
	}
}

// deletable reports whether rel (slash-separated path relative to the walk
// root) is an exact known legacy generated path. Only flat legacy layouts
// qualify; nested or unknown-stem files are custom user content (keep).
func deletable(root, rel string, skills, agents map[string]bool) bool {
	parts := strings.Split(rel, "/")
	base := parts[len(parts)-1]
	stem, ok := stemOf(base)
	if !ok || stem == "" {
		return false
	}
	switch root {
	case ".fabrik/agents", ".cursor/agents", ".opencode/agents", ".pi/agent/agents", ".agents/agents":
		return len(parts) == 1 && agents[stem]
	case ".opencode/skills", ".pi/skills":
		return len(parts) == 1 && skills[stem]
	case ".cursor/rules":
		if len(parts) == 1 {
			return skills[stem]
		}
		// legacy nested agent copies: .cursor/rules/agents/<agent>.mdc
		return len(parts) == 2 && parts[0] == "agents" && agents[stem]
	case ".agents/skills":
		// legacy layout: .agents/skills/<skill>/SKILL.md
		return len(parts) == 2 && skills[parts[0]] && strings.EqualFold(stem, "SKILL")
	}
	return false
}

// Plan returns the absolute paths of generated files to delete.
func Plan(repo string) ([]string, error) {
	targets, _, err := plan(repo)
	return targets, err
}

// plan returns deletion targets plus the .md/.mdc files that are kept
// because they are custom user content (not an exact known legacy path),
// for dry-run reporting.
func plan(repo string) (targets, kept []string, err error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, nil, err
	}
	skills, agents := knownStems()
	var out []string
	var keep []string
	add := func(rel string) {
		full := filepath.Join(abs, filepath.FromSlash(rel))
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			out = append(out, full)
		}
	}
	addGlob := func(pattern string) {
		matches, _ := filepath.Glob(filepath.Join(abs, filepath.FromSlash(pattern)))
		for _, m := range matches {
			if st, err := os.Stat(m); err == nil && !st.IsDir() {
				out = append(out, m)
			}
		}
	}
	// generated hub runners / dumps
	addGlob(".fabrik/PROMPT_*.md")
	addGlob(".fabrik/loop.*")
	addGlob(".fabrik/fabrik.*")
	addGlob(".fabrik/setup.*")
	for _, rel := range []string{
		".fabrik/state.md",
		".fabrik/README.md",
		".fabrik/CONTEXT.md",
		".fabrik/skills.md",
		".fabrik/AGENTS.md",
		".fabrik/.gitignore",
		".fabrik/styleguide/STYLEGUIDE.md",
	} {
		add(rel)
	}
	// full-body copies only (keep shims); only exact known legacy paths
	// qualify — custom user files in the same dirs are kept. .agents/agents
	// holds Antigravity copies (shims stay, full bodies go). .opencode/agents
	// never received copies (roster lives in opencode.json show-shims) but is
	// walked anyway so a stray full body there is pruned.
	for _, root := range []string{
		".fabrik/agents",
		".cursor/agents",
		".cursor/rules",
		".opencode/skills",
		".opencode/agents",
		".pi/skills",
		".pi/agent/agents",
		".agents/skills",
		".agents/agents",
	} {
		base := filepath.Join(abs, filepath.FromSlash(root))
		_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(p)) {
			case ".md", ".mdc":
			default:
				return nil
			}
			rel, err := filepath.Rel(base, p)
			if err != nil {
				return nil
			}
			if !deletable(root, filepath.ToSlash(rel), skills, agents) {
				keep = append(keep, p)
				return nil
			}
			body, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			if IsShim(string(body)) {
				return nil
			}
			out = append(out, p)
			return nil
		})
	}

	sort.Strings(out)
	// de-dup (overlapping globs)
	out = slices.Compact(out)
	sort.Strings(keep)
	keep = slices.Compact(keep)
	return out, keep, nil
}

// printPrunePreview prints the would-delete / keep lines for a dry run and
// returns how many lines were printed.
func printPrunePreview(d Deps, abs string, targets, kept []string) int {
	for _, t := range targets {
		rel, _ := filepath.Rel(abs, t)
		fmt.Fprintf(d.Out, "would delete %s\n", filepath.ToSlash(rel))
	}
	for _, k := range kept {
		fmt.Fprintln(d.Out, keepMsg(abs, k))
	}
	return len(targets) + len(kept)
}

func keepMsg(abs, p string) string {
	rel, _ := filepath.Rel(abs, p)
	return fmt.Sprintf("keep %s (custom file, not generated)", filepath.ToSlash(rel))
}

// planPrune is plan() renamed for the prune leg (kept separate from the
// import plan).
func planPrune(repo string) (targets, kept []string, err error) {
	return plan(repo)
}

// Migrate imports legacy Beads/Graphify/Engram state into the Fabrik-native
// stack (never deleting sources) and, only with --prune, deletes legacy
// generated dump files. --dry-run prints the would-import list (plus the
// would-delete list with --prune) without touching disk. A run with nothing
// to do prints `migrate: nothing to migrate` and exits 0.
func Migrate(cfg Config, d Deps) error {
	d.defaults()
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
	ctx := context.Background()
	plan, err := PlanImport(abs, d)
	if err != nil {
		return err
	}
	if cfg.DryRun {
		reported := 0
		for _, it := range plan.Items {
			fmt.Fprintf(d.Out, "would import %s (%s -> %s)\n", it.Summary, it.From, it.To)
			reported++
			if it.Source == "beads" && plan.beadsDeps+plan.beadsNotes > 0 {
				fmt.Fprintf(d.Out, "would import %d beads deps and %d beads notes (%s -> .fabrik/dolt)\n",
					plan.beadsDeps, plan.beadsNotes, plan.beadsFrom)
				reported++
			}
		}
		for _, n := range plan.Notices {
			fmt.Fprintln(d.Out, n)
			reported++
		}
		if cfg.Prune {
			targets, kept, err := planPrune(abs)
			if err != nil {
				return err
			}
			reported += printPrunePreview(d, abs, targets, kept)
		}
		if reported == 0 {
			fmt.Fprintln(d.Out, "migrate: nothing to migrate")
		}
		return nil
	}
	imported, notices, err := runImport(ctx, abs, d, plan, d.Out)
	if err != nil {
		return err
	}
	reported := len(imported)
	if cfg.Prune {
		targets, kept, err := planPrune(abs)
		if err != nil {
			return err
		}
		for _, t := range targets {
			rel, _ := filepath.Rel(abs, t)
			if err := os.Remove(t); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", filepath.ToSlash(rel), err)
			}
			fmt.Fprintf(d.Out, "deleted %s\n", filepath.ToSlash(rel))
		}
		for _, k := range kept {
			fmt.Fprintln(d.Out, keepMsg(abs, k))
		}
		// best-effort: drop now-empty generated dirs (never docs/specs)
		for _, dir := range []string{
			filepath.Join(abs, ".fabrik", "agents"),
			filepath.Join(abs, ".fabrik", "styleguide"),
			filepath.Join(abs, ".cursor", "agents"),
			filepath.Join(abs, ".cursor", "rules", "agents"),
			filepath.Join(abs, ".cursor", "rules"),
			filepath.Join(abs, ".opencode", "skills"),
			filepath.Join(abs, ".opencode", "agents"),
			filepath.Join(abs, ".pi", "skills"),
			filepath.Join(abs, ".pi", "agent", "agents"),
			filepath.Join(abs, ".agents", "skills"),
			filepath.Join(abs, ".agents", "agents"),
		} {
			_ = os.Remove(dir) // fails silently when non-empty
		}
		reported += len(targets) + len(kept)
	}
	// runImport already printed import lines and skip notices; count the
	// notices as reported so a skips-only run does not claim emptiness.
	reported += len(notices)
	if reported == 0 {
		fmt.Fprintln(d.Out, "migrate: nothing to migrate")
	}
	return nil
}
