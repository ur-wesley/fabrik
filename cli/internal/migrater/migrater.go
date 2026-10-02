// Package migrater implements `fabrik migrate --prune [--dry-run]`.
//
// It deletes legacy generated dumps only and never touches user content:
// config.yaml, docs/, specs/, user PRD, root AGENTS.md, opencode.json.
// Skill/agent copies containing an exact `fabrik show skill|agent <name>`
// shim marker are kept; only full bodies are pruned.
package migrater

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
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
}

func (d *Deps) defaults() {
	if d.Out == nil {
		d.Out = os.Stdout
	}
}

// shimRe matches the canonical thin-pointer markers produced by
// initer.skillShim / initer.agentShim (`Run: fabrik show skill <name>` /
// `Run: fabrik show agent <name>`), so prose that merely mentions
// "skillset" or similar is still pruned as a full body.
var shimRe = regexp.MustCompile(`Run: fabrik show (skill|agent) [A-Za-z0-9_-]+`)

// IsShim reports whether body is a thin pointer shim (keep, never delete).
func IsShim(body string) bool {
	return shimRe.MatchString(body)
}

// Plan returns the absolute paths of generated files to delete.
func Plan(repo string) ([]string, error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	var out []string
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
	// full-body copies only (keep shims); walk recursively so nested
	// skill dirs and both .md/.mdc Cursor rules are covered.
	// .opencode/agents never received copies (roster lives in opencode.json
	// show-shims) but is walked anyway so a stray full body there is pruned.
	for _, root := range []string{
		".fabrik/agents",
		".cursor/agents",
		".cursor/rules",
		".opencode/skills",
		".opencode/agents",
		".pi/skills",
		".pi/agent/agents",
		".agents/skills",
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
	return out, nil
}

// Migrate lists (dry-run) or deletes (prune) generated files.
func Migrate(cfg Config, d Deps) error {
	d.defaults()
	if !cfg.Prune {
		return fmt.Errorf("nothing to do: pass --prune to delete generated files (use --dry-run with --prune to preview)")
	}
	repo := cfg.RepoPath
	if repo == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		repo = cwd
	}
	targets, err := Plan(repo)
	if err != nil {
		return err
	}
	abs, _ := filepath.Abs(repo)
	if cfg.DryRun {
		if len(targets) == 0 {
			fmt.Fprintln(d.Out, "migrate: nothing generated to prune")
			return nil
		}
		for _, t := range targets {
			rel, _ := filepath.Rel(abs, t)
			fmt.Fprintf(d.Out, "would delete %s\n", filepath.ToSlash(rel))
		}
		return nil
	}
	for _, t := range targets {
		rel, _ := filepath.Rel(abs, t)
		if err := os.Remove(t); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", filepath.ToSlash(rel), err)
		}
		fmt.Fprintf(d.Out, "deleted %s\n", filepath.ToSlash(rel))
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
	} {
		_ = os.Remove(dir) // fails silently when non-empty
	}
	if len(targets) == 0 {
		fmt.Fprintln(d.Out, "migrate: nothing generated to prune")
	}
	return nil
}
