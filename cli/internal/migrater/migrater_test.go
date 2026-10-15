package migrater

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

// fileRefRe matches opencode.json {file:<rel>} refs.
var fileRefRe = regexp.MustCompile(`\{file:([^}]+)\}`)

func seed(t *testing.T, repo, rel, body string) {
	t.Helper()
	full := filepath.Join(repo, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
}

func seedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	full := "FULL BODY generated dump without any shim marker\n# line\n"
	// generated hub files
	seed(t, repo, ".fabrik/PROMPT_plan.md", full)
	seed(t, repo, ".fabrik/PROMPT_build.md", full)
	seed(t, repo, ".fabrik/loop.sh", full)
	seed(t, repo, ".fabrik/fabrik.sh", full)
	seed(t, repo, ".fabrik/setup.sh", full)
	seed(t, repo, ".fabrik/state.md", full)
	seed(t, repo, ".fabrik/README.md", full)
	seed(t, repo, ".fabrik/CONTEXT.md", full)
	seed(t, repo, ".fabrik/skills.md", full)
	seed(t, repo, ".fabrik/AGENTS.md", full)
	seed(t, repo, ".fabrik/styleguide/STYLEGUIDE.md", full)
	seed(t, repo, ".fabrik/agents/builder.md", full)
	// generated skill/agent copies (full bodies)
	seed(t, repo, ".cursor/rules/tdd.mdc", full)
	seed(t, repo, ".cursor/rules/agents/builder.mdc", full)
	seed(t, repo, ".cursor/agents/builder.md", full)
	seed(t, repo, ".opencode/skills/tdd.md", full)
	seed(t, repo, ".pi/skills/tdd.md", full)
	seed(t, repo, ".pi/agent/agents/builder.md", full)
	// shims that must be kept (canonical skillShim/agentShim bodies)
	seed(t, repo, ".cursor/rules/ponytail.mdc", "# Fabrik skill ponytail\n\nRun: fabrik show skill ponytail\n")
	seed(t, repo, ".opencode/skills/ponytail.md", "# Fabrik skill ponytail\n\nRun: fabrik show skill ponytail\n")
	seed(t, repo, ".pi/skills/ponytail.md", "# Fabrik skill ponytail\n\nRun: fabrik show skill ponytail\n")
	seed(t, repo, ".pi/agent/agents/planner.md", "# Fabrik subagent planner\n\nRun: fabrik show agent planner\n")
	seed(t, repo, ".cursor/agents/planner.md", "---\nname: planner\ndescription: Post-APPROVE land\nmodel: inherit\nreadonly: false\n---\n\n# Fabrik subagent planner\n\nRun: fabrik show agent planner\n")
	// user files that must never be touched
	seed(t, repo, ".fabrik/config.yaml", "version: 1\n")
	seed(t, repo, ".fabrik/docs/PRD.md", "# user PRD\n")
	seed(t, repo, ".fabrik/docs/extra.md", "docs\n")
	seed(t, repo, ".fabrik/specs/001.md", "spec\n")
	seed(t, repo, "AGENTS.md", "# user root agents\n")
	seed(t, repo, "opencode.json", "{}\n")
	return repo
}

func TestPlanListsGeneratedOnly(t *testing.T) {
	repo := seedRepo(t)
	got, err := Plan(repo)
	require.NoError(t, err)
	set := map[string]bool{}
	for _, p := range got {
		rel, _ := filepath.Rel(repo, p)
		set[filepath.ToSlash(rel)] = true
	}
	joined := ""
	for k := range set {
		joined += k + "\n"
	}
	for _, want := range []string{
		".fabrik/PROMPT_plan.md", ".fabrik/loop.sh", ".fabrik/fabrik.sh",
		".fabrik/setup.sh", ".fabrik/state.md", ".fabrik/README.md",
		".fabrik/CONTEXT.md", ".fabrik/skills.md", ".fabrik/AGENTS.md",
		".fabrik/styleguide/STYLEGUIDE.md", ".fabrik/agents/builder.md",
		".cursor/rules/tdd.mdc", ".opencode/skills/tdd.md",
		".pi/skills/tdd.md", ".pi/agent/agents/builder.md",
		".cursor/rules/agents/builder.mdc", ".cursor/agents/builder.md",
	} {
		assert.True(t, set[want], "plan must include generated file %s (got:\n%s)", want, joined)
	}
	for _, keep := range []string{
		".fabrik/config.yaml", ".fabrik/docs/PRD.md", ".fabrik/specs/001.md",
		"AGENTS.md", "opencode.json",
		".cursor/rules/ponytail.mdc", ".opencode/skills/ponytail.md",
		".pi/skills/ponytail.md", ".pi/agent/agents/planner.md",
		".cursor/agents/planner.md",
	} {
		assert.False(t, set[keep], "plan must not include user/shim file %s (got:\n%s)", keep, joined)
	}
}

func TestDryRunTouchesNothing(t *testing.T) {
	isolateImportEnv(t)
	repo := seedRepo(t)
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true, DryRun: true}, testDeps(&out)))
	assert.Contains(t, out.String(), "would delete")
	// generated file still on disk after dry-run
	assert.FileExists(t, filepath.Join(repo, ".fabrik", "README.md"))
	assert.FileExists(t, filepath.Join(repo, ".cursor", "rules", "tdd.mdc"))
}

func TestPruneDeletesGeneratedOnly(t *testing.T) {
	isolateImportEnv(t)
	repo := seedRepo(t)
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true}, testDeps(&out)))
	for _, gone := range []string{
		".fabrik/PROMPT_plan.md", ".fabrik/loop.sh", ".fabrik/README.md",
		".fabrik/CONTEXT.md", ".fabrik/skills.md", ".fabrik/AGENTS.md",
		".fabrik/styleguide/STYLEGUIDE.md", ".fabrik/agents/builder.md",
		".cursor/rules/tdd.mdc", ".opencode/skills/tdd.md",
		".pi/skills/tdd.md", ".pi/agent/agents/builder.md",
	} {
		assert.NoFileExists(t, filepath.Join(repo, filepath.FromSlash(gone)), gone)
	}
	// user files untouched
	for _, keep := range []string{
		".fabrik/config.yaml", ".fabrik/docs/PRD.md", ".fabrik/specs/001.md",
		"AGENTS.md", "opencode.json",
		".cursor/rules/ponytail.mdc", ".pi/agent/agents/planner.md",
	} {
		assert.FileExists(t, filepath.Join(repo, filepath.FromSlash(keep)), keep)
	}
}

// cli-s34: init output (embedded opencode.json template) followed by
// migrate --prune must leave zero dangling {file:} refs. Seeds the real
// template plus the full bodies the old refs pointed at, prunes, then
// resolves every remaining ref against disk.
func TestInitTemplatePruneLeavesNoDanglingFileRefs(t *testing.T) {
	isolateImportEnv(t)
	repo := t.TempDir()
	seed(t, repo, "opencode.json", tmpl.MustRead("opencode.json"))
	for _, a := range []string{"orchestrator", "builder"} {
		body, err := tmpl.Read("agents/" + a + ".md")
		require.NoError(t, err)
		seed(t, repo, ".fabrik/agents/"+a+".md", body)
	}
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true}, testDeps(&out)))
	assert.FileExists(t, filepath.Join(repo, "opencode.json"), "opencode.json is user config, never pruned")
	raw, err := os.ReadFile(filepath.Join(repo, "opencode.json"))
	require.NoError(t, err)
	// explicit non-vacuous guard: the template must carry zero {file:} refs
	// (a pure FindAll loop would pass vacuously when no refs exist).
	assert.NotContains(t, string(raw), "{file:")
	assert.Empty(t, fileRefRe.FindAllString(string(raw), -1), "opencode.json must carry zero {file:} refs")
	for _, m := range fileRefRe.FindAllStringSubmatch(string(raw), -1) {
		target := filepath.Join(repo, filepath.FromSlash(m[1]))
		assert.FileExists(t, target, "dangling ref %s after prune", m[0])
	}
}

func TestPruneRequiresFlag(t *testing.T) {
	// cli-hrq: plain `migrate` runs the legacy import; the empty case exits
	// 0 with "migrate: nothing to migrate" instead of demanding --prune.
	isolateImportEnv(t)
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: t.TempDir()}, testDeps(&out)))
	assert.Contains(t, out.String(), "migrate: nothing to migrate")
}

func TestPlanGitignoreNestedAndFalseShim(t *testing.T) {
	repo := t.TempDir()
	full := "FULL BODY generated dump without any shim marker\n"
	// generated .gitignore from init --full
	seed(t, repo, ".fabrik/.gitignore", full)
	// exact known legacy paths (flat, allowlisted stems) are still pruned
	seed(t, repo, ".opencode/skills/tdd.md", full)
	seed(t, repo, ".cursor/rules/tdd.md", full)
	seed(t, repo, ".cursor/rules/agents/builder.mdc", full)
	seed(t, repo, ".agents/skills/tdd/SKILL.md", full)
	seed(t, repo, ".agents/agents/builder.md", full)
	// custom user files: unknown stems or nested layouts are kept
	seed(t, repo, ".opencode/skills/nested/deep.md", full)
	seed(t, repo, ".opencode/agents/stray.md", full)
	seed(t, repo, ".pi/skills/team/custom.md", full)
	seed(t, repo, ".cursor/rules/extra/nested.mdc", full)
	// false shim: mentions the words but is not an exact pointer marker
	seed(t, repo, ".opencode/skills/false.md", "notes about fabrik show skillset ideas\n")
	// nested known stem is not an exact legacy path: kept
	seed(t, repo, ".opencode/skills/nested/tdd.md", full)
	seed(t, repo, ".agents/skills/custom/SKILL.md", full)
	// true nested shim must be kept
	seed(t, repo, ".pi/skills/nested/keep.md", "# Fabrik skill tdd\n\nRun: fabrik show skill tdd\n")

	got, err := Plan(repo)
	require.NoError(t, err)
	set := map[string]bool{}
	for _, p := range got {
		rel, _ := filepath.Rel(repo, p)
		set[filepath.ToSlash(rel)] = true
	}
	for _, want := range []string{
		".fabrik/.gitignore",
		".opencode/skills/tdd.md",
		".cursor/rules/tdd.md",
		".cursor/rules/agents/builder.mdc",
		".agents/skills/tdd/SKILL.md",
		".agents/agents/builder.md",
	} {
		assert.True(t, set[want], "plan must include %s (got %v)", want, set)
	}
	for _, keep := range []string{
		".opencode/skills/nested/deep.md",
		".opencode/agents/stray.md",
		".pi/skills/team/custom.md",
		".cursor/rules/extra/nested.mdc",
		".opencode/skills/false.md",
		".opencode/skills/nested/tdd.md",
		".agents/skills/custom/SKILL.md",
		".pi/skills/nested/keep.md",
	} {
		assert.False(t, set[keep], "plan must keep custom file %s (got %v)", keep, set)
	}
}

func TestIsShimRequiresExactMarkerLine(t *testing.T) {
	assert.True(t, IsShim("# Fabrik skill tdd\n\nRun: fabrik show skill tdd\n"))
	assert.True(t, IsShim("---\nname: planner\n---\n\n# Fabrik subagent Planner\n\nRun: fabrik show agent planner\n"))
	// prose mentioning the command mid-sentence is a full body, not a shim
	assert.False(t, IsShim("see Run: fabrik show skill tdd for details\n"))
	assert.False(t, IsShim("notes about fabrik show skillset ideas\n"))
}

// cli-mip: the canonical matchers distinguish skill vs agent markers —
// reseter relies on the specific predicates, so pin the split here.
func TestSkillAgentShimPredicates(t *testing.T) {
	skill := "Run: fabrik show skill tdd\n"
	agent := "Run: fabrik show agent planner\n"
	assert.True(t, IsSkillShim(skill))
	assert.False(t, IsAgentShim(skill))
	assert.True(t, IsShim(skill))
	assert.True(t, IsAgentShim(agent))
	assert.False(t, IsSkillShim(agent))
	assert.True(t, IsShim(agent))
	assert.False(t, IsSkillShim("no marker here\n"))
	assert.False(t, IsAgentShim("no marker here\n"))
}

// cli-19h: custom user rules outside the known legacy set must survive
// migration; known legacy full bodies are still removed.
func TestCustomRuleFilesSurviveMigration(t *testing.T) {
	isolateImportEnv(t)
	repo := t.TempDir()
	full := "FULL BODY generated dump without any shim marker\n"
	// known legacy full bodies (removed)
	for _, gone := range []string{
		".cursor/rules/tdd.mdc",
		".opencode/skills/tdd.md",
		".pi/agent/agents/builder.md",
	} {
		seed(t, repo, gone, full)
	}
	// custom user rules (kept)
	custom := []string{
		".cursor/rules/my-team-rule.mdc",
		".cursor/rules/my-team-rule.md",
		".cursor/agents/my-helper.md",
		".opencode/skills/my-notes.md",
		".opencode/agents/my-helper.md",
		".pi/skills/custom.md",
		".pi/agent/agents/my-helper.md",
		".fabrik/agents/my-helper.md",
		".agents/skills/custom/SKILL.md",
	}
	for _, c := range custom {
		seed(t, repo, c, "# my own rule\n\nsome custom content\n")
	}
	var dry bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true, DryRun: true}, testDeps(&dry)))
	assert.Contains(t, dry.String(), "would delete .cursor/rules/tdd.mdc")
	for _, c := range custom {
		assert.Contains(t, dry.String(), "keep "+c, "dry-run must log kept custom file")
	}
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true}, testDeps(&out)))
	for _, gone := range []string{
		".cursor/rules/tdd.mdc",
		".opencode/skills/tdd.md",
		".pi/agent/agents/builder.md",
	} {
		assert.NoFileExists(t, filepath.Join(repo, filepath.FromSlash(gone)), gone)
	}
	for _, c := range custom {
		assert.FileExists(t, filepath.Join(repo, filepath.FromSlash(c)), c)
	}
}

// ---- cli-hrq: legacy import (Beads/Graphify/Engram), no deletes ----

// isolateImportEnv seals legacy-import tests: the memory target goes to a
// temp dir and the Engram source points at a nonexistent file, so no test
// can read the real ~/.engram/engram.db or write the real
// ~/.fabrik/memory.db. Returns the temp memory dir.
func isolateImportEnv(t *testing.T) string {
	t.Helper()
	memdir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", memdir)
	t.Setenv("FABRIK_ENGRAM_DB", filepath.Join(t.TempDir(), "no-engram.db"))
	return memdir
}

var errTestBD = errors.New("test: bd fallback must not run")

// testDeps wires output plus a bd seam that must never fire in fixture
// tests (.beads/issues.jsonl covers detection, so production's `bd list`
// fallback stays unused). Tests that exercise the fallback pass their own
// Deps instead.
func testDeps(out *bytes.Buffer) Deps {
	return Deps{Out: out, BDList: func(ctx context.Context, repo string) ([]byte, error) {
		return nil, errTestBD
	}}
}

func seedBeadsJSONL(t *testing.T, repo string) string {
	t.Helper()
	body := `{"id":"cli-hrq","title":"migrate import","description":"desc body","status":"open","priority":1,"issue_type":"task","created_at":"2026-10-14T07:19:11Z","updated_at":"2026-10-14T07:19:35Z"}` + "\n" +
		`{"id":"cli-done","title":"finished thing","description":"done body","status":"closed","priority":0,"issue_type":"bug","assignee":"ur-wesley","created_at":"2026-10-01T00:00:00Z","updated_at":"2026-10-02T00:00:00Z","closed_at":"2026-10-02T00:00:00Z"}` + "\n"
	seed(t, repo, ".beads/issues.jsonl", body)
	return body
}

func seedGraphify(t *testing.T, repo string) string {
	t.Helper()
	body := `{"nodes":[{"id":"n1","kind":"function","name":"Migrate","path":"cli/internal/migrater/migrater.go","line":243,"lang":"go"},{"id":"n2","kind":"file","name":"migrater.go","path":"cli/internal/migrater/migrater.go","lang":"go"}],"edges":[{"src":"n1","dst":"n2","kind":"defined_in","label":""}]}`
	seed(t, repo, "graphify-out/graph.json", body)
	return body
}

// seedEngramDB creates a minimal Engram-shaped source database scoped to
// project: two live observations (one with an unknown kind, one with
// topic/session) plus one soft-deleted row that must never be imported, one
// live row for a foreign project that the project filter must exclude, and a
// sessions table with one matching session (markdown summary) plus one
// foreign session that must also be excluded.
func seedEngramDB(t *testing.T, path, project string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE observations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		project TEXT,
		topic_key TEXT,
		created_at TEXT NOT NULL,
		deleted_at TEXT
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO observations
		(session_id, type, title, content, project, topic_key, created_at, deleted_at) VALUES
		('s-1', 'decision', 'Keep Dolt', 'Dolt stays the source of truth', ?, 'arch/store', '2026-10-01T10:00:00Z', NULL),
		('s-1', 'field-note', 'Unknown kind', 'kind falls back to discovery', ?, NULL, '2026-10-02T10:00:00Z', NULL),
		('s-2', 'discovery', 'Old deleted', 'must not be imported', ?, NULL, '2026-09-01T10:00:00Z', '2026-09-02T10:00:00Z'),
		('s-9', 'decision', 'Foreign project', 'must not be imported', 'other-project-zzz', NULL, '2026-10-03T10:00:00Z', NULL)`,
		project, project, project)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		project TEXT NOT NULL,
		directory TEXT NOT NULL,
		started_at TEXT NOT NULL DEFAULT (datetime('now')),
		ended_at TEXT,
		summary TEXT
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO sessions (id, project, directory, started_at, ended_at, summary) VALUES
		('sess-1', ?, '/repo', '2026-10-01T09:00:00Z', NULL,
		'## Goal' || char(10) || 'Ship migrate' || char(10) || char(10) ||
		'## Instructions' || char(10) || 'Do not delete sources' || char(10) || char(10) ||
		'## Discoveries' || char(10) || 'Found the seam' || char(10) || char(10) ||
		'## Accomplished' || char(10) || 'Wrote legs' || char(10) || char(10) ||
		'## Next Steps' || char(10) || 'Verify live' || char(10) || char(10) ||
		'## Files' || char(10) || 'cli/internal/migrater/import.go'),
		('sess-foreign', 'other-project-zzz', '/elsewhere', '2026-10-01T09:00:00Z', NULL, 'foreign summary')`,
		project)
	require.NoError(t, err)
}

// queryStrings runs a read query against a SQLite file and returns all rows
// as strings (NULL reads as "").
func queryStrings(t *testing.T, dbPath, query string, args ...any) [][]string {
	t.Helper()
	require.FileExists(t, dbPath)
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query(query, args...)
	require.NoError(t, err)
	defer rows.Close()
	cols, err := rows.Columns()
	require.NoError(t, err)
	var out [][]string
	for rows.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		row := make([]string, len(cols))
		for i, v := range vals {
			row[i] = v.String
		}
		out = append(out, row)
	}
	require.NoError(t, rows.Err())
	return out
}

func seedAllSources(t *testing.T, repo string) (beadsBody, graphBody, engramPath string) {
	t.Helper()
	beadsBody = seedBeadsJSONL(t, repo)
	graphBody = seedGraphify(t, repo)
	engramPath = filepath.Join(t.TempDir(), "engram.db")
	seedEngramDB(t, engramPath, filepath.Base(repo))
	t.Setenv("FABRIK_ENGRAM_DB", engramPath)
	return beadsBody, graphBody, engramPath
}

func TestImportEmptyPrintsNothingToMigrate(t *testing.T) {
	memdir := isolateImportEnv(t)
	repo := t.TempDir()
	for name, cfg := range map[string]Config{
		"plain":      {RepoPath: repo},
		"dry-run":    {RepoPath: repo, DryRun: true},
		"prune":      {RepoPath: repo, Prune: true},
		"prune-dry":  {RepoPath: repo, Prune: true, DryRun: true},
	} {
		var out bytes.Buffer
		require.NoError(t, Migrate(cfg, testDeps(&out)), name)
		assert.Contains(t, out.String(), "migrate: nothing to migrate", name)
	}
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik"))
	assert.NoFileExists(t, filepath.Join(memdir, "memory.db"))
}

func TestPlanImportListsSources(t *testing.T) {
	isolateImportEnv(t)
	repo := t.TempDir()
	_, _, engramPath := seedAllSources(t, repo)
	plan, err := PlanImport(repo, testDeps(nil))
	require.NoError(t, err)
	require.Len(t, plan.Items, 3)
	assert.Empty(t, plan.Notices)
	bySource := map[string]ImportItem{}
	for _, it := range plan.Items {
		bySource[it.Source] = it
	}
	beads := bySource["beads"]
	assert.Equal(t, ".beads/issues.jsonl", beads.From)
	assert.Equal(t, ".fabrik/dolt", beads.To)
	assert.Equal(t, "2 beads issues", beads.Summary)
	graph := bySource["graphify"]
	assert.Equal(t, "graphify-out/graph.json", graph.From)
	assert.Equal(t, ".fabrik/graph.db", graph.To)
	assert.Equal(t, "2 nodes, 1 edges", graph.Summary)
	engram := bySource["engram"]
	assert.Equal(t, engramPath, engram.From)
	assert.True(t, strings.HasSuffix(engram.To, "memory.db"), engram.To)
	assert.Equal(t, "2 engram observations, 1 session", engram.Summary)
}

func TestPlanImportEmbeddedOnlyNotice(t *testing.T) {
	isolateImportEnv(t)
	repo := t.TempDir()
	seed(t, repo, ".beads/embeddeddolt/.keep", "")
	plan, err := PlanImport(repo, testDeps(nil))
	require.NoError(t, err)
	assert.Empty(t, plan.Items)
	require.Len(t, plan.Notices, 1)
	assert.Contains(t, plan.Notices[0], "embeddeddolt")
	assert.Contains(t, plan.Notices[0], "test: bd fallback must not run")
}

func TestImportDryRunTouchesNothing(t *testing.T) {
	memdir := isolateImportEnv(t)
	repo := t.TempDir()
	beadsBody, graphBody, engramPath := seedAllSources(t, repo)
	engramBefore, err := os.ReadFile(engramPath)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, DryRun: true}, testDeps(&out)))
	text := out.String()
	assert.Contains(t, text, "would import 2 beads issues (.beads/issues.jsonl -> .fabrik/dolt)")
	assert.Contains(t, text, "would import 2 nodes, 1 edges (graphify-out/graph.json -> .fabrik/graph.db)")
	assert.Contains(t, text, "would import 2 engram observations, 1 session")
	assert.NotContains(t, text, "would delete")

	assert.NoDirExists(t, filepath.Join(repo, ".fabrik"), "dry-run must not create .fabrik")
	assert.NoFileExists(t, filepath.Join(memdir, "memory.db"), "dry-run must not create memory.db")
	beadsAfter, err := os.ReadFile(filepath.Join(repo, ".beads", "issues.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, beadsBody, string(beadsAfter))
	graphAfter, err := os.ReadFile(filepath.Join(repo, "graphify-out", "graph.json"))
	require.NoError(t, err)
	assert.Equal(t, graphBody, string(graphAfter))
	engramAfter, err := os.ReadFile(engramPath)
	require.NoError(t, err)
	assert.Equal(t, engramBefore, engramAfter, "dry-run must not modify the Engram source")
}

func TestImportDryRunWithPrunePreviewsBoth(t *testing.T) {
	isolateImportEnv(t)
	repo := t.TempDir()
	seedAllSources(t, repo)
	seed(t, repo, ".fabrik/README.md", "FULL BODY generated dump without any shim marker\n")
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true, DryRun: true}, testDeps(&out)))
	assert.Contains(t, out.String(), "would import 2 beads issues")
	assert.Contains(t, out.String(), "would delete .fabrik/README.md")
	assert.FileExists(t, filepath.Join(repo, ".fabrik", "README.md"), "dry-run must not delete")
	assert.FileExists(t, filepath.Join(repo, ".beads", "issues.jsonl"))
	assert.FileExists(t, filepath.Join(repo, "graphify-out", "graph.json"))
}

func TestImportGraphAndMemory(t *testing.T) {
	memdir := isolateImportEnv(t)
	repo := t.TempDir()
	// no .beads here: the beads leg is covered by its own tests below.
	_, graphBody, engramPath := seedAllSources(t, repo)
	require.NoError(t, os.RemoveAll(filepath.Join(repo, ".beads")))
	project := filepath.Base(repo)

	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out)))
	text := out.String()
	assert.Contains(t, text, "imported 2 graphify nodes, 1 edges to .fabrik/graph.db")
	assert.Contains(t, text, "imported 2 engram observations to ")
	assert.Contains(t, text, "imported 1 engram session to ")
	assert.Contains(t, text, "(0 already present)")

	graphDB := filepath.Join(repo, ".fabrik", "graph.db")
	nodes := queryStrings(t, graphDB, `SELECT id, kind, name, path, line, lang FROM graph_nodes ORDER BY id`)
	require.Len(t, nodes, 2)
	assert.Equal(t, []string{"n1", "function", "Migrate", "cli/internal/migrater/migrater.go", "243", "go"}, nodes[0])
	assert.Equal(t, []string{"n2", "file", "migrater.go", "cli/internal/migrater/migrater.go", "0", "go"}, nodes[1])
	edges := queryStrings(t, graphDB, `SELECT src, dst, kind, label FROM graph_edges`)
	require.Len(t, edges, 1)
	assert.Equal(t, []string{"n1", "n2", "defined_in", ""}, edges[0])
	files := queryStrings(t, graphDB, `SELECT path, ok FROM graph_files`)
	require.Len(t, files, 1)
	assert.Equal(t, "cli/internal/migrater/migrater.go", files[0][0])

	memDB := filepath.Join(memdir, "memory.db")
	obs := queryStrings(t, memDB, `SELECT id, session_id, project, kind, title, body, topic_key, created_at FROM observations ORDER BY id`)
	require.Len(t, obs, 2, "soft-deleted and foreign-project Engram rows must not be imported")
	assert.Equal(t, []string{"1", "s-1", project, "decision", "Keep Dolt", "Dolt stays the source of truth", "arch/store", "2026-10-01T10:00:00Z"}, obs[0])
	assert.Equal(t, []string{"2", "s-1", project, "discovery", "Unknown kind", "kind falls back to discovery", "", "2026-10-02T10:00:00Z"}, obs[1])
	assert.Len(t, queryStrings(t, memDB, `SELECT rowid FROM observations_fts`), 2, "FTS rows must mirror observations")

	sess := queryStrings(t, memDB, `SELECT id, started_at, ended_at, goal, discoveries, accomplished, next_steps, files FROM sessions ORDER BY id`)
	require.Len(t, sess, 1, "foreign-project Engram sessions must not be imported")
	assert.Equal(t, "sess-1", sess[0][0])
	assert.Equal(t, "2026-10-01T09:00:00Z", sess[0][1])
	assert.Equal(t, "", sess[0][2], "open session has no ended_at")
	assert.Equal(t, "Ship migrate", sess[0][3])
	assert.Equal(t, "Do not delete sources\nFound the seam", sess[0][4], "## Instructions folds into discoveries")
	assert.Equal(t, "Wrote legs", sess[0][5])
	assert.Equal(t, "Verify live", sess[0][6])
	assert.Equal(t, "cli/internal/migrater/import.go", sess[0][7])

	// sources survive the import
	graphAfter, err := os.ReadFile(filepath.Join(repo, "graphify-out", "graph.json"))
	require.NoError(t, err)
	assert.Equal(t, graphBody, string(graphAfter))
	assert.Len(t, queryStrings(t, engramPath, `SELECT id FROM observations`), 4)

	// re-run is idempotent: same ids, nothing new
	var out2 bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out2)))
	assert.Contains(t, out2.String(), "imported 0 graphify nodes, 0 edges to .fabrik/graph.db")
	assert.Contains(t, out2.String(), "(2 already present)")
	assert.Contains(t, out2.String(), "imported 0 engram sessions to ")
	assert.Contains(t, out2.String(), "(1 already present)")
	assert.Len(t, queryStrings(t, graphDB, `SELECT id FROM graph_nodes`), 2)
	assert.Len(t, queryStrings(t, memDB, `SELECT id FROM observations`), 2)
	assert.Len(t, queryStrings(t, memDB, `SELECT id FROM sessions`), 1)
}

func TestImportBeadsSkippedWithoutEngine(t *testing.T) {
	if doltstore.HasCGO {
		t.Skip("engine present: covered by TestImportBeadsToDolt")
	}
	memdir := isolateImportEnv(t)
	repo := t.TempDir()
	beadsBody := seedBeadsJSONL(t, repo)

	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out)))
	assert.Contains(t, out.String(), "dolt engine unavailable")
	assert.Contains(t, out.String(), "found 2 issues")
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik", "dolt"))
	assert.NoFileExists(t, filepath.Join(memdir, "memory.db"))
	after, err := os.ReadFile(filepath.Join(repo, ".beads", "issues.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, beadsBody, string(after))
}

func TestImportBeadsToDolt(t *testing.T) {
	if !doltstore.HasCGO {
		t.Skip("requires CGO_ENABLED=1 for the embedded Dolt engine")
	}
	isolateImportEnv(t)
	repo := t.TempDir()
	seed(t, repo, ".beads/issues.jsonl",
		`{"id":"cli-a","title":"open task","description":"body a","status":"open","priority":2,"issue_type":"task","created_at":"2026-10-01T00:00:00Z","updated_at":"2026-10-01T00:00:00Z"}`+"\n"+
			`{"id":"cli-b","title":"active bug","description":"body b","status":"in_progress","priority":1,"issue_type":"bug","assignee":"dev","created_at":"2026-10-02T00:00:00Z","updated_at":"2026-10-03T00:00:00Z"}`+"\n"+
			`{"id":"cli-c","title":"closed feature","description":"body c","acceptance_criteria":"must work","status":"closed","priority":0,"issue_type":"feature","created_at":"2026-10-04T00:00:00Z","updated_at":"2026-10-05T00:00:00Z","closed_at":"2026-10-05T00:00:00Z"}`+"\n")

	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out)))
	assert.Contains(t, out.String(), "imported 3 beads issues to .fabrik/dolt (0 already present)")

	ctx := context.Background()
	st, err := doltstore.Open(ctx, repo)
	require.NoError(t, err)
	defer st.Close()
	a, err := st.Get(ctx, beadsImportID("cli-a"))
	require.NoError(t, err)
	assert.Equal(t, "open task", a.Title)
	assert.Equal(t, "body a", a.Body)
	assert.Equal(t, "open", a.Status)
	assert.Equal(t, 2, a.Priority)
	assert.Equal(t, "task", a.Kind)
	b, err := st.Get(ctx, beadsImportID("cli-b"))
	require.NoError(t, err)
	assert.Equal(t, "active", b.Status)
	assert.Equal(t, "bug", b.Kind)
	require.NotNil(t, b.Assignee)
	assert.Equal(t, "dev", *b.Assignee)
	c, err := st.Get(ctx, beadsImportID("cli-c"))
	require.NoError(t, err)
	assert.Equal(t, "done", c.Status)
	assert.Equal(t, "feature", c.Kind)
	assert.Equal(t, "must work", c.Acceptance)
	assert.Equal(t, 0, c.Priority)

	// Release the exclusive Dolt lock before the second Migrate opens the
	// same directory again (open_cgo.go gives up after 500ms).
	require.NoError(t, st.Close())

	var out2 bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out2)))
	assert.Contains(t, out2.String(), "imported 0 beads issues to .fabrik/dolt (3 already present)")

	after, err := os.ReadFile(filepath.Join(repo, ".beads", "issues.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(after), `"id":"cli-a"`)
}

func TestImportNeverDeletesSources(t *testing.T) {
	isolateImportEnv(t)
	repo := t.TempDir()
	beadsBody, graphBody, engramPath := seedAllSources(t, repo)
	engramBefore, err := os.ReadFile(engramPath)
	require.NoError(t, err)
	seed(t, repo, ".fabrik/README.md", "FULL BODY generated dump without any shim marker\n")
	seed(t, repo, ".fabrik/config.yaml", "version: 1\n")

	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, Prune: true}, testDeps(&out)))
	// prune still works on generated dumps ...
	assert.NoFileExists(t, filepath.Join(repo, ".fabrik", "README.md"))
	assert.FileExists(t, filepath.Join(repo, ".fabrik", "config.yaml"))
	// ... but legacy sources are never removed, even with --prune.
	beadsAfter, err := os.ReadFile(filepath.Join(repo, ".beads", "issues.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, beadsBody, string(beadsAfter))
	graphAfter, err := os.ReadFile(filepath.Join(repo, "graphify-out", "graph.json"))
	require.NoError(t, err)
	assert.Equal(t, graphBody, string(graphAfter))
	engramAfter, err := os.ReadFile(engramPath)
	require.NoError(t, err)
	assert.Equal(t, engramBefore, engramAfter)
}

func TestMigrateCorruptSourceFails(t *testing.T) {
	isolateImportEnv(t)
	repo := t.TempDir()
	seed(t, repo, "graphify-out/graph.json", "{not json")
	seed(t, repo, ".beads/issues.jsonl", "{also not json\n")
	var out bytes.Buffer
	err := Migrate(Config{RepoPath: repo}, testDeps(&out))
	require.Error(t, err)
	assert.Contains(t, err.Error(), ".beads/issues.jsonl")
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik"), "failed import must not create stores")
}

func TestParseBeadsShapes(t *testing.T) {
	// `bd list --json` array shape with Beads field names
	arr := `[{"id":"x-1","title":"T","description":"D","acceptance_criteria":"A","status":"in_progress","priority":1,"issue_type":"bug","assignee":"me","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z","started_at":"2026-01-02T00:00:00Z"}]`
	issues, err := parseBeadsBytes([]byte(arr), "test")
	require.NoError(t, err)
	require.Len(t, issues, 1)
	assert.Equal(t, "active", issues[0].Status)
	assert.Equal(t, "bug", issues[0].Kind)
	assert.Equal(t, "D", issues[0].Body)
	assert.Equal(t, "A", issues[0].Acceptance)
	assert.Equal(t, "me", issues[0].Assignee)
	assert.Equal(t, 1, issues[0].Priority)

	// JSONL shape with Fabrik-native aliases, priority clamp, kind default
	lines := "{\"id\":\"x-2\",\"title\":\"U\",\"body\":\"B\",\"status\":\"deferred\",\"priority\":9}\n\n{\"id\":\"x-3\",\"title\":\"V\",\"status\":\"closed\",\"priority\":\"high\"}\n"
	issues, err = parseBeadsBytes([]byte(lines), "test")
	require.NoError(t, err)
	require.Len(t, issues, 2)
	assert.Equal(t, "open", issues[0].Status, "unknown status falls back to open")
	assert.Equal(t, 4, issues[0].Priority, "priority clamps to 0..4")
	assert.Equal(t, "task", issues[0].Kind)
	assert.Equal(t, "done", issues[1].Status)
	assert.Equal(t, 2, issues[1].Priority, "non-numeric priority defaults to 2")

	_, err = parseBeadsBytes([]byte("{\"title\":\"no id\"}\n"), "test")
	require.Error(t, err)
	_, err = parseBeadsBytes([]byte("{\"id\":\"x\",\"title\":\"\"}\n"), "test")
	require.Error(t, err)
	_, err = parseBeadsBytes([]byte("{broken\n"), "test")
	require.Error(t, err)

	idRe := regexp.MustCompile(`^f-[0-9a-f]{8}$`)
	a, b := beadsImportID("cli-x"), beadsImportID("cli-x")
	assert.True(t, idRe.MatchString(a), a)
	assert.Equal(t, a, b, "beads ids map deterministically for idempotent re-imports")
	assert.NotEqual(t, a, beadsImportID("cli-y"))
}

func TestParseGraphifyShapes(t *testing.T) {
	doc := `{"graph":{"nodes":[{"id":"a","type":"class","title":"Widget","file":"w.py","lineno":"10","language":"py"},{"no-id":"skip"}],"edges":[{"from":"a","to":"b","type":"uses","name":"e1"},{"src":"a"}]}}`
	payload, err := parseGraphifyBytes([]byte(doc), "test")
	require.NoError(t, err)
	require.Len(t, payload.nodes, 1)
	assert.Equal(t, graphNode{ID: "a", Kind: "class", Name: "Widget", Path: "w.py", Line: 10, Lang: "py"}, payload.nodes[0])
	require.Len(t, payload.edges, 1, "edges without both endpoints are skipped")
	assert.Equal(t, graphEdge{Src: "a", Dst: "b", Kind: "uses", Label: "e1"}, payload.edges[0])

	_, err = parseGraphifyBytes([]byte("nope"), "test")
	require.Error(t, err)
	empty, err := parseGraphifyBytes([]byte(`{"nodes":[],"edges":[]}`), "test")
	require.NoError(t, err)
	assert.Empty(t, empty.nodes)
	assert.Empty(t, empty.edges)
}

// cli-hrq: full-fidelity Beads parsing — `bd export` shape with dependencies,
// comments, notes string, close reason, and design folded into the body.
func TestParseBeadsDepsNotes(t *testing.T) {
	doc := `{"_type":"issue","id":"x-1","title":"Blocked task","description":"body one","design":"the plan","status":"open","priority":1,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-02T00:00:00Z",` +
		`"dependencies":[{"issue_id":"x-1","depends_on_id":"x-0","type":"blocks"},{"issue_id":"x-1","depends_on_id":"x-9","type":"relates-to"},{"issue_id":"x-1","depends_on_id":"x-8","type":"discovered-from"},{"issue_id":"x-1","depends_on_id":"x-1","type":"blocks"},{"issue_id":"","depends_on_id":"x-0","type":"blocks"}],` +
		`"comments":[{"id":"c1","issue_id":"x-1","author":"dev","text":"first comment","created_at":"2026-01-03T00:00:00Z"},{"text":"no timestamp","created_at":null}],` +
		`"notes":"line one\nline two","close_reason":"Done"}` + "\n" +
		`{"id":"x-0","title":"Blocker","status":"open","priority":2}` + "\n"
	issues, err := parseBeadsBytes([]byte(doc), "test")
	require.NoError(t, err)
	require.Len(t, issues, 2)

	a := issues[0]
	assert.Equal(t, "body one\n\n## Design\nthe plan", a.Body)
	require.Len(t, a.Deps, 3, "self-edge and empty-side rows are skipped")
	assert.Equal(t, beadsDep{BlockerBeadsID: "x-0", BlockedBeadsID: "x-1", Kind: "blocks"}, a.Deps[0])
	assert.Equal(t, beadsDep{BlockerBeadsID: "x-9", BlockedBeadsID: "x-1", Kind: "relates"}, a.Deps[1])
	assert.Equal(t, beadsDep{BlockerBeadsID: "x-8", BlockedBeadsID: "x-1", Kind: "relates"}, a.Deps[2])

	require.Len(t, a.Notes, 4, "2 comments + notes string + close reason")
	assert.Equal(t, "dev: first comment", a.Notes[0].Body)
	assert.Equal(t, "2026-01-03T00:00:00Z", a.Notes[0].CreatedAt)
	assert.Equal(t, "no timestamp", a.Notes[1].Body)
	assert.Equal(t, "2026-01-02T00:00:00Z", a.Notes[1].CreatedAt, "missing comment timestamp falls back to issue updated_at")
	assert.Equal(t, "line one\nline two", a.Notes[2].Body, "notes string imports whole to preserve text exactly")
	assert.Equal(t, "Close reason: Done", a.Notes[3].Body)

	n1, n2 := beadsNoteID("x-1", 0, a.Notes[0].Body, a.Notes[0].CreatedAt), beadsNoteID("x-1", 0, a.Notes[0].Body, a.Notes[0].CreatedAt)
	assert.Equal(t, n1, n2, "note ids are deterministic for idempotent re-imports")
	assert.NotEqual(t, n1, beadsNoteID("x-1", 1, a.Notes[0].Body, a.Notes[0].CreatedAt))
	assert.Regexp(t, `^n-[0-9a-f]{8}$`, n1)
}

func TestParseSessionSummary(t *testing.T) {
	goal, disc, acc, next, files := parseSessionSummary("just a goal, no headings")
	assert.Equal(t, "just a goal, no headings", goal)
	assert.Empty(t, disc)

	goal, disc, acc, next, files = parseSessionSummary("## Goal\nShip it\n\n## Instructions\nBe careful\n\n## Discoveries\nFound X\n\n## Accomplished\nDid Y\n\n## Next Steps\nDo Z\n\n## Files\na.go\n")
	assert.Equal(t, "Ship it", goal)
	assert.Equal(t, "Be careful\nFound X", disc, "## Instructions folds into discoveries")
	assert.Equal(t, "Did Y", acc)
	assert.Equal(t, "Do Z", next)
	assert.Equal(t, "a.go", files)

	goal, disc, acc, next, files = parseSessionSummary("## Goal\nG\n\n## Custom\nkept text\n")
	assert.Equal(t, "G\n## Custom\nkept text", goal, "unknown headings keep heading and text in the current bucket")

	goal, disc, acc, next, files = parseSessionSummary("")
	assert.Empty(t, goal)
	assert.Empty(t, disc)
	assert.Empty(t, acc)
	assert.Empty(t, next)
	assert.Empty(t, files)

	assert.Equal(t, "handoff", mapMemoryKind("session_summary"))
	assert.Equal(t, "discovery", mapMemoryKind("field-note"))
	assert.Equal(t, "decision", mapMemoryKind("decision"))
}

// cli-hrq: Beads deps + notes land in issue_deps / issue_notes; a blocks edge
// flips the open blocked issue to blocked; reruns are idempotent.
func TestImportBeadsDepsNotesToDolt(t *testing.T) {
	if !doltstore.HasCGO {
		t.Skip("requires CGO_ENABLED=1 for the embedded Dolt engine")
	}
	isolateImportEnv(t)
	repo := t.TempDir()
	seed(t, repo, ".beads/issues.jsonl",
		`{"id":"cli-a","title":"blocker","description":"body a","status":"open","priority":2,"issue_type":"task","created_at":"2026-10-01T00:00:00Z","updated_at":"2026-10-01T00:00:00Z"}`+"\n"+
			`{"id":"cli-b","title":"blocked bug","description":"body b","status":"open","priority":1,"issue_type":"bug","created_at":"2026-10-02T00:00:00Z","updated_at":"2026-10-03T00:00:00Z",`+
			`"dependencies":[{"issue_id":"cli-b","depends_on_id":"cli-a","type":"blocks"},{"issue_id":"cli-b","depends_on_id":"cli-ghost","type":"blocks"}],`+
			`"comments":[{"author":"dev","text":"look here","created_at":"2026-10-03T00:00:00Z"}],"notes":"note line"} `+"\n"+
			`{"id":"cli-c","title":"closed feature","description":"body c","status":"closed","priority":0,"issue_type":"feature","created_at":"2026-10-04T00:00:00Z","updated_at":"2026-10-05T00:00:00Z","closed_at":"2026-10-05T00:00:00Z","close_reason":"Shipped",`+
			`"dependencies":[{"issue_id":"cli-c","depends_on_id":"cli-a","type":"discovered-from"}]}`+"\n")

	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out)))
	assert.Contains(t, out.String(), "imported 3 beads issues to .fabrik/dolt (0 already present)")
	assert.Contains(t, out.String(), "imported 2 beads deps and 3 beads notes to .fabrik/dolt (0 already present)")
	assert.Contains(t, out.String(), "outside the import set — skipped", "dangling cli-ghost edge is reported, not fatal")

	ctx := context.Background()
	st, err := doltstore.Open(ctx, repo)
	require.NoError(t, err)
	defer st.Close()

	blocked, err := st.Show(ctx, beadsImportID("cli-b"))
	require.NoError(t, err)
	assert.Equal(t, "blocked", blocked.Status, "new blocks edge flips open blocked issue")
	require.Len(t, blocked.Blockers, 1)
	assert.Equal(t, beadsImportID("cli-a"), blocked.Blockers[0].ID)
	require.Len(t, blocked.Notes, 2)
	assert.Equal(t, "dev: look here", blocked.Notes[0].Body)
	assert.Equal(t, "note line", blocked.Notes[1].Body)

	related, err := st.Show(ctx, beadsImportID("cli-c"))
	require.NoError(t, err)
	assert.Equal(t, "done", related.Status, "relates never changes status")
	require.Len(t, related.Notes, 1)
	assert.Equal(t, "Close reason: Shipped", related.Notes[0].Body)

	// Release the exclusive Dolt lock before the second Migrate opens the
	// same directory again (open_cgo.go gives up after 500ms).
	require.NoError(t, st.Close())

	var out2 bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out2)))
	assert.Contains(t, out2.String(), "imported 0 beads issues to .fabrik/dolt (3 already present)")
	assert.Contains(t, out2.String(), "imported 0 beads deps and 0 beads notes to .fabrik/dolt (5 already present)")
}

// cli-hrq: dry-run with deps/notes previews everything and creates nothing,
// not even the .fabrik directory.
func TestImportDryRunBeadsDepsNotes(t *testing.T) {
	memdir := isolateImportEnv(t)
	repo := t.TempDir()
	seed(t, repo, ".beads/issues.jsonl",
		`{"id":"cli-a","title":"A","status":"open","dependencies":[{"issue_id":"cli-a","depends_on_id":"cli-b","type":"blocks"}],"comments":[{"text":"c"}],"notes":"n"}`+"\n"+
			`{"id":"cli-b","title":"B","status":"open"}`+"\n")
	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo, DryRun: true}, testDeps(&out)))
	assert.Contains(t, out.String(), "would import 2 beads issues (.beads/issues.jsonl -> .fabrik/dolt)")
	assert.Contains(t, out.String(), "would import 1 beads deps and 2 beads notes (.beads/issues.jsonl -> .fabrik/dolt)")
	assert.NoDirExists(t, filepath.Join(repo, ".fabrik"))
	assert.NoFileExists(t, filepath.Join(memdir, "memory.db"))
}

// cli-hrq: older Engram DBs without a sessions table still import
// observations; the sessions leg is simply absent.
func TestImportEngramWithoutSessionsTable(t *testing.T) {
	memdir := isolateImportEnv(t)
	repo := t.TempDir()
	project := filepath.Base(repo)
	src := filepath.Join(t.TempDir(), "engram.db")
	db, err := sql.Open("sqlite", src)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE observations (
		id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL, type TEXT NOT NULL,
		title TEXT NOT NULL, content TEXT NOT NULL, project TEXT, topic_key TEXT,
		created_at TEXT NOT NULL, deleted_at TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO observations (session_id, type, title, content, project, created_at, deleted_at) VALUES
		('s-1', 'decision', 'T', 'B', ?, '2026-10-01T10:00:00Z', NULL)`, project)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	t.Setenv("FABRIK_ENGRAM_DB", src)

	plan, err := PlanImport(repo, testDeps(nil))
	require.NoError(t, err)
	require.Len(t, plan.Items, 1)
	assert.Equal(t, "1 engram observation", plan.Items[0].Summary)

	var out bytes.Buffer
	require.NoError(t, Migrate(Config{RepoPath: repo}, testDeps(&out)))
	assert.Contains(t, out.String(), "imported 1 engram observation")
	assert.NotContains(t, out.String(), "engram session")
	assert.Len(t, queryStrings(t, filepath.Join(memdir, "memory.db"), `SELECT id FROM sessions`), 0)
}
