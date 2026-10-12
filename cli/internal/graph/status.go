package graph

import (
	"context"
	"path/filepath"
)

func (d *DB) Status(ctx context.Context) (StatusResult, error) {
	idx, err := d.indexed(ctx)
	if err != nil {
		return StatusResult{}, err
	}
	if !idx {
		return StatusResult{Indexed: false}, nil
	}
	res := StatusResult{Indexed: true}
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_files WHERE ok = 1`).Scan(&res.Files); err != nil {
		return StatusResult{}, err
	}
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_nodes WHERE kind = 'symbol'`).Scan(&res.Symbols); err != nil {
		return StatusResult{}, err
	}
	stale, err := d.countStale(ctx)
	if err != nil {
		return StatusResult{}, err
	}
	res.StaleFiles = stale
	var last string
	_ = d.sql.QueryRowContext(ctx, `SELECT MAX(mtime) FROM graph_files`).Scan(&last)
	res.LastBuildAt = last
	return res, nil
}

func (d *DB) countStale(ctx context.Context) (int, error) {
	files, err := walkSourceFiles(d.Repo)
	if err != nil {
		return 0, err
	}
	existing := map[string]string{}
	rows, err := d.sql.QueryContext(ctx, `SELECT path, mtime FROM graph_files`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var p, mt string
		if err := rows.Scan(&p, &mt); err != nil {
			return 0, err
		}
		existing[p] = mt
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	stale := 0
	for _, rel := range files {
		mt, err := fileMtimeRFC3339(filepath.Join(d.Repo, rel))
		if err != nil {
			continue
		}
		if existing[rel] != mt {
			stale++
		}
	}
	return stale, nil
}
