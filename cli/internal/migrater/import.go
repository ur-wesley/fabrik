// Legacy import leg of `fabrik migrate`: Beads + Graphify + Engram into the
// Fabrik-native stack, WITHOUT deleting anything.
//
// Mapping:
//
//	beads (.beads/issues.jsonl, else `bd list --all --json`) -> .fabrik/dolt
//	  issues (id -> deterministic f- id, title/body/status/priority/kind,
//	  design folded into body) plus dependency edges (blocks vs relates) and
//	  notes (comments, notes string, close reason -> issue_notes).
//	  Idempotent via stable ids; reruns skip existing rows.
//	graphify-out/graph.json                             -> .fabrik/graph.db
//	Engram observations (~/.engram/engram.db, FABRIK_ENGRAM_DB override)
//	  project-scoped to the repo                        -> ~/.fabrik/memory.db
//	                                                     (FABRIK_MEMORY_DIR override)
//	  observations (deleted rows skipped, unknown kinds -> discovery,
//	  session_summary -> handoff, FTS mirrored) plus sessions rows with
//	  markdown summaries parsed into goal/discoveries/accomplished/next/files.
//
// Deletes are explicitly out of scope: no code path here removes .beads/,
// graphify-out/, or Engram data, even when --prune is set (--prune only
// drives the pre-existing generated-dump pruning in migrater.go).
//
// Beads issue ids (e.g. cli-hrq) do not fit the Fabrik id shape
// (^f-[0-9a-f]{8,32}$), so imports get deterministic f- ids derived from the
// Beads id: re-running an import reuses the same ids and skips rows that
// already exist. Writing .fabrik/dolt needs the embedded Dolt engine
// (CGO_ENABLED=1); without it the beads leg reports a skip and the
// graph/memory legs still run.
package migrater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/sqliteutil"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

// errNoDoltEngine reports that .fabrik/dolt cannot be written because the
// binary was built without the embedded Dolt engine (CGO_ENABLED=0).
var errNoDoltEngine = errors.New("dolt engine unavailable in this build (needs CGO_ENABLED=1)")

// ImportItem is one detected legacy source with its native target.
type ImportItem struct {
	Source  string // beads | graphify | engram
	From    string // human-readable source location
	To      string // human-readable target location
	Summary string // e.g. "12 beads issues", "34 nodes, 41 edges"
	Count   int    // total rows the source holds (imported + already present)
}

// ImportPlan is the result of detection. Payloads (unexported) are reused by
// runImport so detection happens once per Migrate call.
type ImportPlan struct {
	Items   []ImportItem
	Notices []string

	beads      []beadsIssue
	beadsFrom  string
	beadsDeps  int
	beadsNotes int
	graph      graphPayload
	graphFrom  string
	engramSrc  string
	engramTo   string
	engramObs  int
	engramSess int
}

// PlanImport detects legacy Beads/Graphify/Engram state under repo without
// touching anything on disk (the Engram source opens read-only). Corrupt but
// present sources are errors; missing sources are simply absent.
func PlanImport(repo string, d Deps) (ImportPlan, error) {
	d.defaults()
	abs, err := filepath.Abs(repo)
	if err != nil {
		return ImportPlan{}, err
	}
	ctx := context.Background()
	var plan ImportPlan

	beads, from, notice, err := detectBeads(ctx, abs, &d)
	if err != nil {
		return ImportPlan{}, err
	}
	if notice != "" {
		plan.Notices = append(plan.Notices, notice)
	}
	if len(beads) > 0 {
		plan.beads = beads
		plan.beadsFrom = from
		for _, b := range beads {
			plan.beadsDeps += len(b.Deps)
			plan.beadsNotes += len(b.Notes)
		}
		plan.Items = append(plan.Items, ImportItem{
			Source:  "beads",
			From:    from,
			To:      ".fabrik/dolt",
			Summary: fmt.Sprintf("%d beads issues", len(beads)),
			Count:   len(beads),
		})
	}

	graph, gfrom, gnotice, err := detectGraphify(abs)
	if err != nil {
		return ImportPlan{}, err
	}
	if gnotice != "" {
		plan.Notices = append(plan.Notices, gnotice)
	}
	if len(graph.nodes) > 0 || len(graph.edges) > 0 {
		plan.graph = graph
		plan.graphFrom = gfrom
		plan.Items = append(plan.Items, ImportItem{
			Source:  "graphify",
			From:    gfrom,
			To:      ".fabrik/graph.db",
			Summary: fmt.Sprintf("%d nodes, %d edges", len(graph.nodes), len(graph.edges)),
			Count:   len(graph.nodes) + len(graph.edges),
		})
	}

	src, to, obs, sess, enotice, err := detectEngram(abs)
	if err != nil {
		return ImportPlan{}, err
	}
	if enotice != "" {
		plan.Notices = append(plan.Notices, enotice)
	}
	if obs > 0 || sess > 0 {
		plan.engramSrc = src
		plan.engramTo = to
		plan.engramObs = obs
		plan.engramSess = sess
		summary := fmt.Sprintf("%d engram %s", obs, pluralize(obs, "observation", "observations"))
		if sess > 0 {
			summary += fmt.Sprintf(", %d %s", sess, pluralize(sess, "session", "sessions"))
		}
		plan.Items = append(plan.Items, ImportItem{
			Source:  "engram",
			From:    src,
			To:      to,
			Summary: summary,
			Count:   obs + sess,
		})
	}
	return plan, nil
}

func pluralize(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// runImport executes a plan produced by PlanImport and reports per-source
// results. It never deletes source data.
func runImport(ctx context.Context, abs string, d Deps, plan ImportPlan, out io.Writer) ([]ImportItem, []string, error) {
	notices := append([]string{}, plan.Notices...)
	var done []ImportItem
	if len(plan.beads) > 0 {
		res, err := importBeads(ctx, abs, plan.beads)
		if err != nil {
			if errors.Is(err, errNoDoltEngine) {
				notices = append(notices, fmt.Sprintf(
					"beads: found %d issues, %d deps, %d notes (%s); %s — skipped",
					len(plan.beads), plan.beadsDeps, plan.beadsNotes, plan.beadsFrom, errNoDoltEngine))
			} else {
				return done, notices, err
			}
		} else {
			done = append(done, ImportItem{Source: "beads", From: plan.beadsFrom,
				To: ".fabrik/dolt", Summary: fmt.Sprintf("%d beads issues", res.issues), Count: res.issues})
			fmt.Fprintf(out, "imported %d beads issues to .fabrik/dolt (%d already present)\n", res.issues, res.issuesPresent)
			if res.deps+res.notes > 0 || res.depsPresent+res.notesPresent > 0 {
				fmt.Fprintf(out, "imported %d beads deps and %d beads notes to .fabrik/dolt (%d already present)\n",
					res.deps, res.notes, res.depsPresent+res.notesPresent)
			}
			if res.skippedDeps > 0 {
				notices = append(notices, fmt.Sprintf(
					"beads: %d dependencies reference issues outside the import set — skipped", res.skippedDeps))
			}
		}
	}
	if len(plan.graph.nodes) > 0 || len(plan.graph.edges) > 0 {
		nodes, edges, err := importGraphify(ctx, abs, plan.graph)
		if err != nil {
			return done, notices, err
		}
		done = append(done, ImportItem{Source: "graphify", From: plan.graphFrom,
			To: ".fabrik/graph.db", Summary: fmt.Sprintf("%d nodes, %d edges", nodes, edges),
			Count: nodes + edges})
		fmt.Fprintf(out, "imported %d graphify nodes, %d edges to .fabrik/graph.db\n", nodes, edges)
	}
	if plan.engramSrc != "" {
		obsIns, obsPresent, sessIns, sessPresent, err := importEngram(ctx, plan.engramSrc, plan.engramTo, abs)
		if err != nil {
			return done, notices, err
		}
		summary := fmt.Sprintf("%d engram %s", obsIns, pluralize(obsIns, "observation", "observations"))
		if plan.engramSess > 0 {
			summary += fmt.Sprintf(", %d %s", sessIns, pluralize(sessIns, "session", "sessions"))
		}
		done = append(done, ImportItem{Source: "engram", From: plan.engramSrc,
			To: plan.engramTo, Summary: summary, Count: obsIns + sessIns})
		fmt.Fprintf(out, "imported %d engram %s to %s (%d already present)\n", obsIns, pluralize(obsIns, "observation", "observations"), plan.engramTo, obsPresent)
		if plan.engramSess > 0 {
			fmt.Fprintf(out, "imported %d engram %s to %s (%d already present)\n", sessIns, pluralize(sessIns, "session", "sessions"), plan.engramTo, sessPresent)
		}
	}
	for _, n := range notices {
		fmt.Fprintln(out, n)
	}
	return done, notices, nil
}

// ---------------------------------------------------------------------------
// Beads
// ---------------------------------------------------------------------------

// beadsIssue is a Beads issue normalized to Fabrik issue columns, plus its
// dependency edges and notes for the Fabrik issue_deps / issue_notes tables.
type beadsIssue struct {
	ID         string
	Title      string
	Body       string
	Acceptance string
	Status     string // open | active | done
	Priority   int
	Kind       string
	Assignee   string
	CreatedAt  string
	UpdatedAt  string
	StartedAt  string
	ClosedAt   string
	Deps       []beadsDep
	Notes      []beadsNote
}

// beadsDep is a normalized Beads dependency edge. Beads stores the edge on
// the depending issue (issue_id depends on depends_on_id); Fabrik stores
// (blocker_id, blocked_id, kind) with kind blocks|relates.
type beadsDep struct {
	BlockerBeadsID string
	BlockedBeadsID string
	Kind           string // blocks | relates
}

// beadsNote is one note for Fabrik issue_notes: a Beads comment, the issue's
// notes string, or its close reason.
type beadsNote struct {
	Body      string
	CreatedAt string
}

// beadsDepJSON decodes one Beads dependency object (`bd export` shape, also
// present in `bd list --json` when non-empty).
type beadsDepJSON struct {
	IssueID     string `json:"issue_id"`
	DependsOnID string `json:"depends_on_id"`
	Type        string `json:"type"`
}

// beadsCommentJSON decodes one Beads comment (`bd export` shape only;
// `bd list --json` omits comments, so imports from the `bd list` fallback
// carry deps and notes but no comments).
type beadsCommentJSON struct {
	ID        any    `json:"id"`
	Text      string `json:"text"`
	Body      string `json:"body"`
	Content   string `json:"content"`
	CreatedAt any    `json:"created_at"`
	Author    any    `json:"author"`
}

// beadsJSON decodes one Beads issue object tolerantly: field names cover
// `bd list --json` (description, issue_type, acceptance_criteria) and JSONL
// exports (body, kind, acceptance), plus `bd export` extras (design,
// close_reason, dependencies, comments, notes).
type beadsJSON struct {
	ID                 string             `json:"id"`
	Title              string             `json:"title"`
	Description        string             `json:"description"`
	Body               string             `json:"body"`
	Design             string             `json:"design"`
	Acceptance         string             `json:"acceptance"`
	AcceptanceCriteria string             `json:"acceptance_criteria"`
	CloseReason        string             `json:"close_reason"`
	Notes              any                `json:"notes"`
	Status             string             `json:"status"`
	Priority           any                `json:"priority"`
	IssueType          string             `json:"issue_type"`
	Kind               string             `json:"kind"`
	Type               string             `json:"type"`
	Assignee           any                `json:"assignee"`
	CreatedAt          any                `json:"created_at"`
	UpdatedAt          any                `json:"updated_at"`
	StartedAt          any                `json:"started_at"`
	ClosedAt           any                `json:"closed_at"`
	Dependencies       []beadsDepJSON     `json:"dependencies"`
	Comments           []beadsCommentJSON `json:"comments"`
}

func strOf(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func priorityOf(v any) int {
	switch t := v.(type) {
	case nil:
		return 2
	case float64:
		return clampPriority(int(t))
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return clampPriority(n)
		}
		return 2
	default:
		if n, err := strconv.Atoi(strOf(v)); err == nil {
			return clampPriority(n)
		}
		return 2
	}
}

func clampPriority(n int) int {
	if n < 0 {
		return 0
	}
	if n > 4 {
		return 4
	}
	return n
}

// mapBeadsStatus maps Beads operational states onto Fabrik issue statuses
// (open/active/done; blocked is derived by Fabrik itself).
func mapBeadsStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "open":
		return "open"
	case "in_progress", "active":
		return "active"
	case "closed", "done":
		return "done"
	default:
		return "open"
	}
}

func mapBeadsKind(raw beadsJSON) string {
	for _, k := range []string{raw.Kind, raw.IssueType, raw.Type} {
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "task", "bug", "feature", "epic", "chore":
			return strings.ToLower(strings.TrimSpace(k))
		}
	}
	return "task"
}

func mapBeadsDepKind(t string) string {
	// Only an explicit blocks edge carries Fabrik ordering semantics
	// (blocked waits until every blocker is done). Every other Beads edge
	// type (relates-to, discovered-from, ...) is provenance, not ordering,
	// so it maps to relates, which never changes status or ready.
	if strings.ToLower(strings.TrimSpace(t)) == "blocks" {
		return "blocks"
	}
	return "relates"
}

// normalizeBeadsDep normalizes one Beads dependency row. Direction comes from
// the row itself (issue_id depends on depends_on_id); rows with an empty
// side or a self-edge are skipped.
func normalizeBeadsDep(dep beadsDepJSON) (beadsDep, bool) {
	owner := strings.TrimSpace(dep.IssueID)
	target := strings.TrimSpace(dep.DependsOnID)
	if owner == "" || target == "" || owner == target {
		return beadsDep{}, false
	}
	return beadsDep{
		BlockedBeadsID: owner,
		BlockerBeadsID: target,
		Kind:           mapBeadsDepKind(dep.Type),
	}, true
}

// beadsNoteID derives a stable Fabrik note id from a Beads issue id plus the
// note's position and content, so repeated imports reuse rows instead of
// duplicating them.
func beadsNoteID(beadsID string, idx int, body, created string) string {
	sum := sha256.Sum256([]byte("fabrik-migrate-note-v1:" + beadsID + "\x00" + strconv.Itoa(idx) + "\x00" + body + "\x00" + created))
	return "n-" + hex.EncodeToString(sum[:4])
}

func normalizeBeadsIssue(raw beadsJSON, origin string, line int) (beadsIssue, error) {
	where := origin
	if line > 0 {
		where = fmt.Sprintf("%s: line %d", origin, line)
	}
	id := strings.TrimSpace(raw.ID)
	if id == "" {
		return beadsIssue{}, fmt.Errorf("%s: issue without id", where)
	}
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return beadsIssue{}, fmt.Errorf("%s: issue %s without title", origin, id)
	}
	body := raw.Description
	if body == "" {
		body = raw.Body
	}
	if design := strings.TrimSpace(raw.Design); design != "" {
		if strings.TrimSpace(body) != "" {
			body += "\n\n"
		}
		body += "## Design\n" + design
	}
	acceptance := raw.Acceptance
	if acceptance == "" {
		acceptance = raw.AcceptanceCriteria
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := strOf(raw.CreatedAt)
	if created == "" {
		created = now
	}
	updated := strOf(raw.UpdatedAt)
	if updated == "" {
		updated = created
	}
	started := strOf(raw.StartedAt)
	closed := strOf(raw.ClosedAt)
	out := beadsIssue{
		ID:         id,
		Title:      title,
		Body:       body,
		Acceptance: acceptance,
		Status:     mapBeadsStatus(raw.Status),
		Priority:   priorityOf(raw.Priority),
		Kind:       mapBeadsKind(raw),
		Assignee:   strings.TrimSpace(strOf(raw.Assignee)),
		CreatedAt:  created,
		UpdatedAt:  updated,
		StartedAt:  started,
		ClosedAt:   closed,
	}
	for _, dep := range raw.Dependencies {
		if d, ok := normalizeBeadsDep(dep); ok {
			out.Deps = append(out.Deps, d)
		}
	}
	for _, c := range raw.Comments {
		text := strings.TrimSpace(firstNonEmpty(c.Text, c.Body, c.Content))
		if text == "" {
			continue
		}
		if author := strings.TrimSpace(strOf(c.Author)); author != "" {
			text = author + ": " + text
		}
		when := strings.TrimSpace(strOf(c.CreatedAt))
		if when == "" {
			when = updated
		}
		out.Notes = append(out.Notes, beadsNote{Body: text, CreatedAt: when})
	}
	// `bd note` appends to a single newline-joined notes string; the join is
	// lossy, so the whole string imports as one note to preserve text exactly.
	if n := strings.TrimSpace(strOf(raw.Notes)); n != "" {
		out.Notes = append(out.Notes, beadsNote{Body: n, CreatedAt: updated})
	}
	if cr := strings.TrimSpace(raw.CloseReason); cr != "" {
		when := closed
		if when == "" {
			when = updated
		}
		out.Notes = append(out.Notes, beadsNote{Body: "Close reason: " + cr, CreatedAt: when})
	}
	return out, nil
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// parseBeadsBytes accepts either a JSON array (`bd list --json`) or
// line-delimited JSON objects (.beads/issues.jsonl).
func parseBeadsBytes(data []byte, origin string) ([]beadsIssue, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var arr []beadsJSON
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		dec.UseNumber()
		if err := dec.Decode(&arr); err != nil {
			return nil, fmt.Errorf("%s: %w", origin, err)
		}
		out := make([]beadsIssue, 0, len(arr))
		for i, raw := range arr {
			n, err := normalizeBeadsIssue(raw, origin, i+1)
			if err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		return out, nil
	}
	var out []beadsIssue
	for i, line := range bytes.Split(trimmed, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var raw beadsJSON
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.UseNumber()
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("%s: line %d: %w", origin, i+1, err)
		}
		n, err := normalizeBeadsIssue(raw, origin, i+1)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// defaultBDList shells out to `bd list --all --json` scoped to repo.
// --all matters: bare `bd list` hides closed issues, but migration must see
// everything. Best-effort: callers turn failures into skip notices, never
// hard errors.
func defaultBDList(ctx context.Context, repo string) ([]byte, error) {
	path, err := exec.LookPath("bd")
	if err != nil {
		return nil, fmt.Errorf("bd not on PATH")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "list", "--all", "--json")
	cmd.Dir = repo
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("bd list --all --json: %s", msg)
	}
	return stdout.Bytes(), nil
}

// detectBeads finds readable Beads issues: .beads/issues.jsonl first, else
// `bd list --json` when .beads/ exists. An embedded store with no readable
// export yields a skip notice, never an error.
func detectBeads(ctx context.Context, abs string, d *Deps) (issues []beadsIssue, from, notice string, err error) {
	if raw, rerr := os.ReadFile(filepath.Join(abs, ".beads", "issues.jsonl")); rerr == nil {
		issues, err := parseBeadsBytes(raw, ".beads/issues.jsonl")
		if err != nil {
			return nil, "", "", err
		}
		return issues, ".beads/issues.jsonl", "", nil
	}
	if st, serr := os.Stat(filepath.Join(abs, ".beads")); serr != nil || !st.IsDir() {
		return nil, "", "", nil
	}
	bdList := d.BDList
	if bdList == nil {
		bdList = defaultBDList
	}
	out, berr := bdList(ctx, abs)
	if berr == nil {
		issues, perr := parseBeadsBytes(out, "bd list --all --json")
		if perr != nil {
			return nil, "", "", perr
		}
		return issues, "bd list --all --json", "", nil
	}
	if st, serr := os.Stat(filepath.Join(abs, ".beads", "embeddeddolt")); serr == nil && st.IsDir() {
		return nil, "", fmt.Sprintf(
			"beads: .beads/embeddeddolt present but no readable issues export (%s) — skipped", berr), nil
	}
	return nil, "", fmt.Sprintf("beads: .beads/ present but no readable issues export (%s) — skipped", berr), nil
}

// beadsImportID derives a stable Fabrik issue id from a Beads id so repeated
// imports reuse rows instead of duplicating them.
func beadsImportID(old string) string {
	sum := sha256.Sum256([]byte("fabrik-migrate-beads-v1:" + old))
	return "f-" + hex.EncodeToString(sum[:4])
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// beadsImportResult counts one importBeads run. Issues carry their own
// already-present count; deps and notes likewise; skippedDeps counts edges
// whose endpoints are outside the import set (reported as a notice).
type beadsImportResult struct {
	issues, issuesPresent int
	deps, depsPresent     int
	notes, notesPresent   int
	skippedDeps           int
}

// importBeads inserts detected Beads issues into .fabrik/dolt, skipping ids
// that already exist, then edges (issue_deps) and notes (issue_notes) with
// the same idempotency. A new blocks edge flips an open blocked issue to
// blocked, mirroring `fabrik issue dep`.
func importBeads(ctx context.Context, abs string, issues []beadsIssue) (beadsImportResult, error) {
	var res beadsImportResult
	if len(issues) == 0 {
		return res, nil
	}
	if !doltstore.HasCGO {
		return res, errNoDoltEngine
	}
	st, err := doltstore.Open(ctx, abs)
	if err != nil {
		return res, err
	}
	defer st.Close()
	fabID := map[string]string{}
	for _, iss := range issues {
		fabID[iss.ID] = beadsImportID(iss.ID)
	}
	for _, iss := range issues {
		id := fabID[iss.ID]
		var have string
		if err := st.DB.QueryRowContext(ctx, `SELECT id FROM issues WHERE id = ?`, id).Scan(&have); err == nil {
			res.issuesPresent++
			continue
		}
		if _, err := st.DB.ExecContext(ctx, `
			INSERT INTO issues (id, title, body, acceptance, status, priority, kind, assignee,
				created_at, updated_at, started_at, closed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, iss.Title, iss.Body, iss.Acceptance, iss.Status, iss.Priority, iss.Kind,
			nullIfEmpty(iss.Assignee), iss.CreatedAt, iss.UpdatedAt,
			nullIfEmpty(iss.StartedAt), nullIfEmpty(iss.ClosedAt),
		); err != nil {
			return res, fmt.Errorf("import beads issue %s: %w", iss.ID, err)
		}
		res.issues++
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, iss := range issues {
		for _, d := range iss.Deps {
			blocker, okB := fabID[d.BlockerBeadsID]
			blocked, okD := fabID[d.BlockedBeadsID]
			if !okB || !okD {
				res.skippedDeps++
				continue
			}
			var have int
			if err := st.DB.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM issue_deps WHERE blocker_id = ? AND blocked_id = ? AND kind = ?`,
				blocker, blocked, d.Kind).Scan(&have); err != nil {
				return res, err
			}
			if have > 0 {
				res.depsPresent++
				continue
			}
			if _, err := st.DB.ExecContext(ctx, `
				INSERT INTO issue_deps (blocker_id, blocked_id, kind) VALUES (?, ?, ?)`,
				blocker, blocked, d.Kind); err != nil {
				return res, fmt.Errorf("import beads dep %s -> %s: %w", d.BlockerBeadsID, d.BlockedBeadsID, err)
			}
			res.deps++
			if d.Kind == "blocks" {
				if err := applyImportedBlock(ctx, st.DB, blocked, blocker, now); err != nil {
					return res, err
				}
			}
		}
	}
	for _, iss := range issues {
		id := fabID[iss.ID]
		for i, n := range iss.Notes {
			if strings.TrimSpace(n.Body) == "" {
				continue
			}
			nid := beadsNoteID(iss.ID, i, n.Body, n.CreatedAt)
			var have string
			if err := st.DB.QueryRowContext(ctx, `SELECT id FROM issue_notes WHERE id = ?`, nid).Scan(&have); err == nil {
				res.notesPresent++
				continue
			}
			when := n.CreatedAt
			if when == "" {
				when = now
			}
			if _, err := st.DB.ExecContext(ctx, `
				INSERT INTO issue_notes (id, issue_id, body, created_at) VALUES (?, ?, ?, ?)`,
				nid, id, n.Body, when); err != nil {
				return res, fmt.Errorf("import beads note for %s: %w", iss.ID, err)
			}
			res.notes++
		}
	}
	if total := res.issues + res.deps + res.notes; total > 0 {
		msg := fmt.Sprintf("migrate: import %d beads issues", res.issues)
		if res.issues == 0 {
			msg = "migrate: import beads deps/notes"
		}
		if err := st.Commit(ctx, msg); err != nil {
			return res, err
		}
	}
	return res, nil
}

// applyImportedBlock mirrors doltstore Dep for kind=blocks: when the insert
// is new and the blocker is not done and the blocked issue is open, the
// blocked issue becomes blocked.
func applyImportedBlock(ctx context.Context, db *sql.DB, blocked, blocker, now string) error {
	var blockerStatus, blockedStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM issues WHERE id = ?`, blocker).Scan(&blockerStatus); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM issues WHERE id = ?`, blocked).Scan(&blockedStatus); err != nil {
		return err
	}
	if blockerStatus != "done" && blockedStatus == "open" {
		_, err := db.ExecContext(ctx, `UPDATE issues SET status = 'blocked', updated_at = ? WHERE id = ?`, now, blocked)
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Graphify
// ---------------------------------------------------------------------------

type graphNode struct {
	ID, Kind, Name, Path, Lang string
	Line                       int
}

type graphEdge struct {
	Src, Dst, Kind, Label string
}

type graphPayload struct {
	nodes []graphNode
	edges []graphEdge
}

func gstr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s := strings.TrimSpace(strOf(v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func gint(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case json.Number:
				if n, err := t.Int64(); err == nil {
					return int(n)
				}
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
					return n
				}
			}
		}
	}
	return 0
}

// parseGraphifyBytes accepts Graphify's graph.json shape
// ({"nodes":[...],"edges":[...]}), tolerating a single {"graph":{...}}
// envelope and alias field names.
func parseGraphifyBytes(data []byte, origin string) (graphPayload, error) {
	var top map[string]any
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(data)))
	dec.UseNumber()
	if err := dec.Decode(&top); err != nil {
		return graphPayload{}, fmt.Errorf("%s: %w", origin, err)
	}
	if len(top) == 0 {
		return graphPayload{}, nil
	}
	if _, ok := top["nodes"]; !ok {
		if inner, ok := top["graph"].(map[string]any); ok {
			top = inner
		}
	}
	rawNodes, _ := top["nodes"].([]any)
	rawEdges, _ := top["edges"].([]any)
	var out graphPayload
	for _, rn := range rawNodes {
		m, ok := rn.(map[string]any)
		if !ok {
			continue
		}
		id := gstr(m, "id")
		if id == "" {
			continue
		}
		out.nodes = append(out.nodes, graphNode{
			ID:   id,
			Kind: gstr(m, "kind", "type"),
			Name: gstr(m, "name", "title", "label"),
			Path: gstr(m, "path", "file"),
			Line: gint(m, "line", "lineno"),
			Lang: gstr(m, "lang", "language"),
		})
	}
	for _, re := range rawEdges {
		m, ok := re.(map[string]any)
		if !ok {
			continue
		}
		src := gstr(m, "src", "from", "source", "source_id")
		dst := gstr(m, "dst", "to", "target", "target_id")
		if src == "" || dst == "" {
			continue
		}
		out.edges = append(out.edges, graphEdge{
			Src:   src,
			Dst:   dst,
			Kind:  gstr(m, "kind", "type", "relation"),
			Label: gstr(m, "label", "name"),
		})
	}
	return out, nil
}

func detectGraphify(abs string) (graphPayload, string, string, error) {
	rel := filepath.Join("graphify-out", "graph.json")
	raw, err := os.ReadFile(filepath.Join(abs, rel))
	if err != nil {
		return graphPayload{}, "", "", nil
	}
	payload, err := parseGraphifyBytes(raw, filepath.ToSlash(rel))
	if err != nil {
		return graphPayload{}, "", "", err
	}
	if len(payload.nodes) == 0 && len(payload.edges) == 0 {
		return graphPayload{}, "", fmt.Sprintf(
			"%s holds no nodes or edges — skipped", filepath.ToSlash(rel)), nil
	}
	return payload, filepath.ToSlash(rel), "", nil
}

// importGraphify inserts Graphify nodes/edges into .fabrik/graph.db
// (INSERT OR IGNORE / NOT EXISTS: re-runs add nothing). Distinct node paths
// are recorded in graph_files so `graph status` sees the imported files.
func importGraphify(ctx context.Context, abs string, payload graphPayload) (nodes, edges int, err error) {
	dbPath := store.GraphDBPath(abs)
	if _, err := sqlmigrate.ApplySQLite(ctx, "graph", dbPath); err != nil {
		return 0, 0, err
	}
	db, err := sql.Open("sqlite", sqliteutil.DSN(dbPath))
	if err != nil {
		return 0, 0, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	rollback := func(e error) (int, int, error) {
		_ = tx.Rollback()
		return 0, 0, e
	}
	now := time.Now().UTC().Format(time.RFC3339)
	paths := map[string]string{}
	for _, n := range payload.nodes {
		res, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO graph_nodes (id, kind, name, path, line, lang)
			VALUES (?, ?, ?, ?, ?, ?)`,
			n.ID, n.Kind, n.Name, n.Path, n.Line, n.Lang)
		if err != nil {
			return rollback(err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			nodes += int(n)
		}
		if n.Path != "" {
			if _, ok := paths[n.Path]; !ok {
				paths[n.Path] = n.Lang
			}
		}
	}
	for _, e := range payload.edges {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO graph_edges (src, dst, kind, label, confidence)
			SELECT ?, ?, ?, ?, 'imported'
			WHERE NOT EXISTS (
				SELECT 1 FROM graph_edges WHERE src = ? AND dst = ? AND kind = ? AND label = ?)`,
			e.Src, e.Dst, e.Kind, e.Label, e.Src, e.Dst, e.Kind, e.Label)
		if err != nil {
			return rollback(err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			edges += int(n)
		}
	}
	for p, lang := range paths {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO graph_files (path, mtime, lang, ok) VALUES (?, ?, ?, 1)`,
			p, now, lang); err != nil {
			return rollback(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return nodes, edges, nil
}

// ---------------------------------------------------------------------------
// Engram
// ---------------------------------------------------------------------------

func mapMemoryKind(k string) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "session_summary":
		// Engram's per-session markdown summaries become Fabrik handoffs,
		// the kind `fabrik mem session-summary` writes.
		return "handoff"
	case "decision", "bugfix", "pattern", "discovery", "handoff", "prompt":
		return strings.ToLower(strings.TrimSpace(k))
	default:
		return "discovery"
	}
}

// engramSourcePath is ~/.engram/engram.db, overridable for tests via
// FABRIK_ENGRAM_DB (mirrors FABRIK_MEMORY_DIR for the target).
func engramSourcePath() string {
	if p := strings.TrimSpace(os.Getenv("FABRIK_ENGRAM_DB")); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".engram", "engram.db")
}

func engramRODSN(path string) string {
	return "file:" + path + "?mode=ro&_pragma=busy_timeout(5000)"
}

func tableColumns(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	// NOTE: the table name is a fixed literal, never user input
	// (PRAGMA does not take bound parameters).
	rows, err := db.QueryContext(ctx, `SELECT name FROM pragma_table_info('observations')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// engramProjectCandidates returns the Engram project names that belong to
// repoAbs: the absolute path itself (Fabrik stores absolute repo roots as
// project), plus the directory base name in both cases (Engram short names
// like "fabrik" for C:\Arbeit\fabrik). Only rows matching a candidate
// import; other projects' rows stay untouched in the global Engram DB.
func engramProjectCandidates(repoAbs string) []string {
	var out []string
	add := func(s string) {
		if s == "" {
			return
		}
		for _, have := range out {
			if have == s {
				return
			}
		}
		out = append(out, s)
	}
	add(repoAbs)
	base := filepath.Base(repoAbs)
	add(base)
	add(strings.ToLower(base))
	return out
}

func projectFilter(where *string, args *[]any, cols map[string]bool, candidates []string) {
	if !cols["project"] || len(candidates) == 0 {
		return
	}
	place := make([]string, len(candidates))
	for i := range candidates {
		place[i] = "?"
		*args = append(*args, candidates[i])
	}
	if strings.TrimSpace(*where) == "" {
		*where = " WHERE project IN (" + strings.Join(place, ", ") + ")"
	} else {
		*where += " AND project IN (" + strings.Join(place, ", ") + ")"
	}
}

// detectEngram counts live (non-deleted) Engram observations and sessions for
// this repo's project, read-only. A missing source is absent, not an error.
func detectEngram(repoAbs string) (src, target string, obs, sess int, notice string, err error) {
	src = engramSourcePath()
	if src == "" {
		return "", "", 0, 0, "", nil
	}
	if _, err := os.Stat(src); err != nil {
		return "", "", 0, 0, "", nil
	}
	ctx := context.Background()
	db, err := sql.Open("sqlite", engramRODSN(src))
	if err != nil {
		return "", "", 0, 0, "", err
	}
	defer db.Close()
	cols, err := tableColumns(ctx, db)
	if err != nil {
		return "", "", 0, 0, "", err
	}
	if !cols["id"] || !cols["title"] || !cols["content"] {
		return "", "", 0, 0, "", fmt.Errorf("%s: not an Engram observations table", src)
	}
	candidates := engramProjectCandidates(repoAbs)
	where := ""
	var args []any
	if cols["deleted_at"] {
		where = " WHERE deleted_at IS NULL"
	}
	projectFilter(&where, &args, cols, candidates)
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM observations`+where, args...).Scan(&obs); err != nil {
		return "", "", 0, 0, "", err
	}
	if hasTable(ctx, db, "sessions") {
		scols, serr := sessionColumns(ctx, db)
		if serr != nil {
			return "", "", 0, 0, "", serr
		}
		swhere := ""
		var sargs []any
		projectFilter(&swhere, &sargs, scols, candidates)
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`+swhere, sargs...).Scan(&sess); err != nil {
			return "", "", 0, 0, "", err
		}
	}
	target, err = store.MemoryDBPath()
	if err != nil {
		return "", "", 0, 0, "", err
	}
	return src, target, obs, sess, "", nil
}

func hasTable(ctx context.Context, db *sql.DB, name string) bool {
	var have string
	if err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&have); err != nil {
		return false
	}
	return true
}

// sessionColumns lists columns of the Engram sessions table.
func sessionColumns(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	// NOTE: the table name is a fixed literal, never user input
	// (PRAGMA does not take bound parameters).
	rows, err := db.QueryContext(ctx, `SELECT name FROM pragma_table_info('sessions')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

type engramObservation struct {
	id        string
	sessionID sql.NullString
	kind      string
	title     string
	body      string
	project   string
	topicKey  sql.NullString
	createdAt string
}

// importEngram copies live Engram observations for this repo's project into
// the Fabrik memory.db (created on demand, FABRIK_MEMORY_DIR-aware),
// preserving ids for stable re-runs, plus Engram sessions rows whose markdown
// summaries are parsed into the Fabrik session fields. The source stays
// read-only and is never modified. Returns (obsInserted, obsPresent,
// sessInserted, sessPresent, err).
func importEngram(ctx context.Context, src, target, repoAbs string) (int, int, int, int, error) {
	if _, err := sqlmigrate.ApplySQLite(ctx, "memory", target); err != nil {
		return 0, 0, 0, 0, err
	}
	candidates := engramProjectCandidates(repoAbs)
	sdb, err := sql.Open("sqlite", engramRODSN(src))
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer sdb.Close()
	cols, err := tableColumns(ctx, sdb)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	optCols := []string{}
	for _, c := range []string{"session_id", "type", "project", "topic_key", "created_at"} {
		if cols[c] {
			optCols = append(optCols, c)
		}
	}
	query := "SELECT id"
	for _, c := range optCols {
		query += ", " + c
	}
	query += ", title, content"
	where := ""
	var args []any
	if cols["deleted_at"] {
		where = " WHERE deleted_at IS NULL"
	}
	projectFilter(&where, &args, cols, candidates)
	query += " FROM observations" + where + " ORDER BY id"
	rows, err := sdb.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer rows.Close()
	idx := map[string]int{}
	for i, c := range optCols {
		idx[c] = i
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var obs []engramObservation
	for rows.Next() {
		vals := make([]any, 1+len(optCols)+2)
		var id sql.NullString
		vals[0] = &id
		optVals := make([]sql.NullString, len(optCols))
		for i := range optVals {
			vals[1+i] = &optVals[i]
		}
		var title, body sql.NullString
		vals[1+len(optCols)] = &title
		vals[2+len(optCols)] = &body
		if err := rows.Scan(vals...); err != nil {
			return 0, 0, 0, 0, err
		}
		if !id.Valid || !title.Valid || !body.Valid {
			continue
		}
		get := func(c string) string {
			if i, ok := idx[c]; ok && optVals[i].Valid {
				return optVals[i].String
			}
			return ""
		}
		session := sql.NullString{}
		if s := get("session_id"); s != "" {
			session = sql.NullString{String: s, Valid: true}
		}
		topic := sql.NullString{}
		if t := get("topic_key"); t != "" {
			topic = sql.NullString{String: t, Valid: true}
		}
		created := get("created_at")
		if created == "" {
			created = now
		}
		obs = append(obs, engramObservation{
			id:        strings.TrimSpace(id.String),
			sessionID: session,
			kind:      mapMemoryKind(get("type")),
			title:     title.String,
			body:      body.String,
			project:   get("project"),
			topicKey:  topic,
			createdAt: created,
		})
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, 0, err
	}

	sessions, err := readEngramSessions(ctx, sdb, candidates)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	tdb, err := sql.Open("sqlite", sqliteutil.DSN(target))
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer tdb.Close()
	tx, err := tdb.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	inserted, present := 0, 0
	for _, o := range obs {
		var have string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM observations WHERE id = ?`, o.id).Scan(&have); err == nil {
			present++
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO observations (id, session_id, project, kind, title, body, topic_key, symbol_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?)`,
			o.id, o.sessionID, o.project, o.kind, o.title, o.body, o.topicKey, o.createdAt,
		); err != nil {
			_ = tx.Rollback()
			return inserted, present, 0, 0, fmt.Errorf("import engram observation %s: %w", o.id, err)
		}
		var rowid int64
		if err := tx.QueryRowContext(ctx, `SELECT rowid FROM observations WHERE id = ?`, o.id).Scan(&rowid); err != nil {
			_ = tx.Rollback()
			return inserted, present, 0, 0, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO observations_fts (rowid, title, body) VALUES (?, ?, ?)`,
			rowid, o.title, o.body); err != nil {
			_ = tx.Rollback()
			return inserted, present, 0, 0, err
		}
		inserted++
	}
	sessIns, sessPresent := 0, 0
	for _, s := range sessions {
		var have string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM sessions WHERE id = ?`, s.id).Scan(&have); err == nil {
			sessPresent++
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessions (id, actor, started_at, ended_at, goal, discoveries, accomplished, next_steps, files)
			VALUES (?, '', ?, ?, ?, ?, ?, ?, ?)`,
			s.id, s.startedAt, s.endedAt, s.goal, s.discoveries, s.accomplished, s.nextSteps, s.files,
		); err != nil {
			_ = tx.Rollback()
			return inserted, present, sessIns, sessPresent, fmt.Errorf("import engram session %s: %w", s.id, err)
		}
		sessIns++
	}
	if err := tx.Commit(); err != nil {
		return inserted, present, sessIns, sessPresent, err
	}
	return inserted, present, sessIns, sessPresent, nil
}

type engramSession struct {
	id           string
	startedAt    string
	endedAt      sql.NullString
	goal         string
	discoveries  string
	accomplished string
	nextSteps    string
	files        string
}

// readEngramSessions reads this repo's Engram sessions rows (project-scoped).
// A missing sessions table (older Engram DBs) yields no rows, never an error.
func readEngramSessions(ctx context.Context, sdb *sql.DB, candidates []string) ([]engramSession, error) {
	if !hasTable(ctx, sdb, "sessions") {
		return nil, nil
	}
	scols, err := sessionColumns(ctx, sdb)
	if err != nil {
		return nil, err
	}
	cols := []string{"id"}
	for _, c := range []string{"project", "started_at", "ended_at", "summary"} {
		if scols[c] {
			cols = append(cols, c)
		}
	}
	where := ""
	var args []any
	projectFilter(&where, &args, scols, candidates)
	rows, err := sdb.QueryContext(ctx, `SELECT `+strings.Join(cols, ", ")+` FROM sessions`+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pos := map[string]int{}
	for i, c := range cols {
		pos[c] = i
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var out []engramSession
	for rows.Next() {
		vals := make([]any, len(cols))
		cells := make([]sql.NullString, len(cols))
		for i := range cells {
			vals[i] = &cells[i]
		}
		if err := rows.Scan(vals...); err != nil {
			return nil, err
		}
		get := func(c string) string {
			if i, ok := pos[c]; ok && cells[i].Valid {
				return cells[i].String
			}
			return ""
		}
		id := strings.TrimSpace(get("id"))
		if id == "" {
			continue
		}
		started := get("started_at")
		if started == "" {
			started = now
		}
		var ended sql.NullString
		if e := get("ended_at"); e != "" {
			ended = sql.NullString{String: e, Valid: true}
		}
		goal, discoveries, accomplished, nextSteps, files := parseSessionSummary(get("summary"))
		out = append(out, engramSession{
			id: id, startedAt: started, endedAt: ended,
			goal: goal, discoveries: discoveries, accomplished: accomplished,
			nextSteps: nextSteps, files: files,
		})
	}
	return out, rows.Err()
}

// parseSessionSummary splits an Engram session markdown summary into the
// Fabrik session fields. Text before the first ## heading becomes the goal;
// known headings (goal, discoveries, accomplished, next steps, files) fill
// their field; ## Instructions has no Fabrik column and folds into
// discoveries. Heading-less text is the goal as-is.
func parseSessionSummary(text string) (goal, discoveries, accomplished, nextSteps, files string) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	if text == "" {
		return "", "", "", "", ""
	}
	buckets := map[string]*string{
		"goal": &goal, "discoveries": &discoveries, "accomplished": &accomplished,
		"next steps": &nextSteps, "next_steps": &nextSteps, "next": &nextSteps,
		"files": &files, "instructions": &discoveries,
	}
	current := &goal
	flush := func(line string) {
		// Blank lines are skipped: buckets join with single newlines so
		// re-imports stay byte-stable regardless of source spacing.
		if strings.TrimSpace(line) == "" {
			return
		}
		if *current == "" {
			*current = line
		} else {
			*current += "\n" + line
		}
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if h, ok := strings.CutPrefix(trimmed, "##"); ok {
			name := strings.ToLower(strings.TrimSpace(strings.Trim(h, "# ")))
			if b, known := buckets[name]; known {
				current = b
				continue
			}
			// Unknown heading: keep its text in the current bucket so
			// nothing is silently dropped.
			flush(trimmed)
			continue
		}
		flush(strings.TrimRight(line, " \t"))
	}
	goal = strings.TrimSpace(goal)
	discoveries = strings.TrimSpace(discoveries)
	accomplished = strings.TrimSpace(accomplished)
	nextSteps = strings.TrimSpace(nextSteps)
	files = strings.TrimSpace(files)
	return goal, discoveries, accomplished, nextSteps, files
}
