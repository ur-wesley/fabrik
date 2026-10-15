package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/ur-wesley/fabrik/cli/internal/clierr"
)

type SearchOptions struct {
	Query   string
	Limit   int
	Project string
	// AllProjects mirrors the mem_search all_projects argument
	// (default true when Project is omitted). When Project is set,
	// results narrow to that project regardless of this flag.
	AllProjects bool
}

func Search(ctx context.Context, opt SearchOptions) ([]Observation, error) {
	ftsQ, err := buildFTSQuery(opt.Query)
	if err != nil {
		return nil, err
	}
	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return nil, clierr.Usage("fabrik: --limit must be between 1 and 100")
	}

	db, _, err := openDB(ctx, false)
	if err != nil {
		return nil, err
	}
	if db == nil {
		if opt.Project != "" {
			return nil, fmt.Errorf("%w\nknown: ", ErrProjectNotFound)
		}
		return []Observation{}, nil
	}
	defer releaseDB(db)

	if opt.Project != "" {
		projects, err := listProjects(ctx, db)
		if err != nil {
			return nil, err
		}
		found := false
		for _, p := range projects {
			if p == opt.Project {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w\nknown: %s", ErrProjectNotFound, strings.Join(projects, ", "))
		}
	}

	q := `
		SELECT o.id, o.session_id, o.project, o.kind, o.title, o.body, o.symbol_id, o.created_at
		FROM observations o
		INNER JOIN observations_fts fts ON o.rowid = fts.rowid
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL`
	args := []any{ftsQ}
	if opt.Project != "" {
		q += ` AND o.project = ?`
		args = append(args, opt.Project)
	}
	q += ` ORDER BY bm25(observations_fts), o.created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Observation
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func buildFTSQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", clierr.Usage("fabrik: search query is required")
	}
	if strings.Contains(q, `"`) {
		return "", errors.New(`fabrik: search query must not contain "`)
	}
	if utf8.RuneCountInString(q) > 200 {
		return "", errors.New("fabrik: search query must be at most 200 runes")
	}
	words := strings.Fields(q)
	if len(words) == 0 {
		return "", clierr.Usage("fabrik: search query is required")
	}
	parts := make([]string, 0, len(words))
	for _, w := range words {
		parts = append(parts, ftsQuoteToken(w)+"*")
	}
	return strings.Join(parts, " "), nil
}

func ftsQuoteToken(s string) string {
	s = strings.ReplaceAll(s, `"`, `""`)
	return `"` + s + `"`
}

func PrintSearchJSON(w io.Writer, obs []Observation) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(obs)
}

func PrintSearch(w io.Writer, obs []Observation) {
	if len(obs) == 0 {
		_, _ = fmt.Fprintln(w, "no matches")
		return
	}
	for _, o := range obs {
		_, _ = fmt.Fprintf(w, "%s  %s\n", o.ID, o.Title)
	}
}
