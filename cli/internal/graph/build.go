package graph

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/listen"
)

func (d *DB) Build(ctx context.Context) (BuildResult, error) {
	files, err := walkSourceFiles(d.Repo)
	if err != nil {
		return BuildResult{}, fmt.Errorf("fabrik: graph: %w", err)
	}
	res := BuildResult{Files: len(files)}
	impMap := buildImportMap(d.Repo, files)

	existing := map[string]string{}
	rows, err := d.sql.QueryContext(ctx, `SELECT path, mtime FROM graph_files`)
	if err != nil {
		return BuildResult{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var p, mt string
		if err := rows.Scan(&p, &mt); err != nil {
			return BuildResult{}, err
		}
		existing[p] = mt
	}
	if err := rows.Err(); err != nil {
		return BuildResult{}, err
	}

	onDisk := map[string]bool{}
	for _, rel := range files {
		onDisk[rel] = true
		abs := filepath.Join(d.Repo, rel)
		mt, err := fileMtimeRFC3339(abs)
		if err != nil {
			res.Skipped++
			continue
		}
		st, err := os.Stat(abs)
		if err != nil {
			res.Skipped++
			continue
		}
		if st.Size() > maxFileSize {
			if err := d.upsertFileRow(ctx, rel, mt, "", 0); err != nil {
				return BuildResult{}, err
			}
			res.Skipped++
			continue
		}
		if existing[rel] == mt {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			res.Skipped++
			continue
		}
		lang := langForPath(rel)
		if err := d.indexFile(ctx, rel, lang, data, impMap); err != nil {
			return BuildResult{}, err
		}
		res.Parsed++
		ok := 1
		if !parseFile(rel, lang, data).OK {
			ok = 0
		}
		if err := d.upsertFileRow(ctx, rel, mt, lang, ok); err != nil {
			return BuildResult{}, err
		}
	}

	for path := range existing {
		if onDisk[path] {
			continue
		}
		if err := d.deleteFile(ctx, path); err != nil {
			return BuildResult{}, err
		}
	}

	if err := d.rebuildAllCalls(ctx, impMap); err != nil {
		return BuildResult{}, err
	}

	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_nodes`).Scan(&res.Nodes); err != nil {
		return BuildResult{}, err
	}
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_edges`).Scan(&res.Edges); err != nil {
		return BuildResult{}, err
	}

	listen.SignalGraph(d.Repo)
	return res, nil
}

func (d *DB) RefreshStale(ctx context.Context) error {
	files, err := walkSourceFiles(d.Repo)
	if err != nil {
		return err
	}
	impMap := buildImportMap(d.Repo, files)
	existing := map[string]string{}
	rows, err := d.sql.QueryContext(ctx, `SELECT path, mtime FROM graph_files`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p, mt string
		if err := rows.Scan(&p, &mt); err != nil {
			return err
		}
		existing[p] = mt
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var changed []string
	for _, rel := range files {
		abs := filepath.Join(d.Repo, rel)
		st, err := os.Stat(abs)
		if err != nil || st.Size() > maxFileSize {
			continue
		}
		mt, err := fileMtimeRFC3339(abs)
		if err != nil || existing[rel] == mt {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		lang := langForPath(rel)
		if err := d.indexFile(ctx, rel, lang, data, impMap); err != nil {
			return err
		}
		ok := 1
		if !parseFile(rel, lang, data).OK {
			ok = 0
		}
		if err := d.upsertFileRow(ctx, rel, mt, lang, ok); err != nil {
			return err
		}
		if ok == 1 {
			changed = append(changed, rel)
		}
	}
	for _, rel := range changed {
		if err := d.rebuildCallsForFile(ctx, rel, impMap); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) rebuildCallsForFile(ctx context.Context, rel string, impMap importMap) error {
	prefix := "sym:" + rel + ":"
	_, err := d.sql.ExecContext(ctx, `
		DELETE FROM graph_edges WHERE kind = 'calls' AND (src LIKE ? OR dst LIKE ?)`,
		prefix+"%", prefix+"%")
	if err != nil {
		return err
	}
	var lang string
	err = d.sql.QueryRowContext(ctx, `SELECT lang FROM graph_files WHERE path = ?`, rel).Scan(&lang)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(d.Repo, rel))
	if err != nil {
		return nil
	}
	return d.insertCallsForFile(ctx, rel, parseFile(rel, lang, data), impMap)
}

func (d *DB) upsertFileRow(ctx context.Context, rel, mtime, lang string, ok int) error {
	_, err := d.sql.ExecContext(ctx, `
		INSERT INTO graph_files (path, mtime, lang, ok) VALUES (?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET mtime=excluded.mtime, lang=excluded.lang, ok=excluded.ok`,
		rel, mtime, lang, ok)
	return err
}

func (d *DB) deleteFile(ctx context.Context, rel string) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := d.purgePath(ctx, tx, rel); err != nil {
		_ = tx.Rollback()
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM graph_files WHERE path = ?`, rel)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *DB) purgePath(ctx context.Context, tx *sql.Tx, rel string) error {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM graph_nodes WHERE path = ?`, rel)
	if err != nil {
		return err
	}
	var ids []string
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM graph_edges WHERE src = ? OR dst = ?`, id, id); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM graph_nodes WHERE path = ?`, rel)
	return err
}

func (d *DB) indexFile(ctx context.Context, rel, lang string, data []byte, impMap importMap) error {
	fe := parseFile(rel, lang, data)
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := d.purgePath(ctx, tx, rel); err != nil {
		_ = tx.Rollback()
		return err
	}
	fid := fileID(rel)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, kind, name, path, line, lang) VALUES (?, 'file', ?, ?, 0, ?)`,
		fid, filepath.Base(rel), rel, lang); err != nil {
		_ = tx.Rollback()
		return err
	}
	seenSym := map[string]int{}
	for _, s := range fe.Symbols {
		if seenSym[s.Name] == 0 {
			seenSym[s.Name] = s.Line
		}
	}
	for name, line := range seenSym {
		sid := symID(rel, name)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO graph_nodes (id, kind, name, path, line, lang) VALUES (?, 'symbol', ?, ?, ?, ?)`,
			sid, name, rel, line, lang); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO graph_edges (src, dst, kind, label, confidence) VALUES (?, ?, 'contains', '', 'extracted')`,
			fid, sid); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	for _, imp := range fe.Imports {
		dst := ""
		label := imp
		conf := "inferred"
		switch lang {
		case "go":
			if target := resolveGoImport(imp, impMap); target != "" {
				dst = fileID(target)
				label = ""
				conf = "extracted"
			}
		case "java", "kotlin":
			if target := resolveJvmImport(imp, impMap); target != "" {
				dst = fileID(target)
				label = ""
				conf = "extracted"
			}
		case "c_sharp":
			if target := resolveCSharpImport(imp, impMap); target != "" {
				dst = fileID(target)
				label = ""
				conf = "extracted"
			}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO graph_edges (src, dst, kind, label, confidence) VALUES (?, ?, 'imports', ?, ?)`,
			fid, dst, label, conf); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) rebuildAllCalls(ctx context.Context, impMap importMap) error {
	if _, err := d.sql.ExecContext(ctx, `DELETE FROM graph_edges WHERE kind = 'calls'`); err != nil {
		return err
	}
	rows, err := d.sql.QueryContext(ctx, `SELECT path, lang FROM graph_files WHERE ok = 1`)
	if err != nil {
		return err
	}
	var paths []struct {
		path, lang string
	}
	defer rows.Close()
	for rows.Next() {
		var p, lang string
		if err := rows.Scan(&p, &lang); err != nil {
			return err
		}
		paths = append(paths, struct{path, lang string}{p, lang})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range paths {
		data, err := os.ReadFile(filepath.Join(d.Repo, item.path))
		if err != nil {
			continue
		}
		fe := parseFile(item.path, item.lang, data)
		if err := d.insertCallsForFile(ctx, item.path, fe, impMap); err != nil {
			return err
		}
	}
	return nil
}

func enclosingSymbol(syms []extractedSymbol, line int) string {
	best := ""
	bestLine := -1
	for _, s := range syms {
		if s.Line <= line && s.Line > bestLine {
			bestLine = s.Line
			best = s.Name
		}
	}
	return best
}

func (d *DB) insertCallsForFile(ctx context.Context, rel string, fe fileExtract, impMap importMap) error {
	byFile, byName := d.loadSymIndexes(ctx)
	imported := importedFilesFor(ctx, d, rel)
	dstFiles := imported
	if len(dstFiles) == 0 {
		dstFiles = nil
	}
	for _, c := range fe.Calls {
		callerName := enclosingSymbol(fe.Symbols, c.Line)
		src := fileID(rel)
		if callerName != "" {
			src = symID(rel, callerName)
		}
		dst, conf := resolveCallee(c.Name, dstFiles, byFile, byName)
		label := ""
		if dst == "" {
			label = c.Name
		}
		_, err := d.sql.ExecContext(ctx, `
			INSERT INTO graph_edges (src, dst, kind, label, confidence) VALUES (?, ?, 'calls', ?, ?)`,
			src, dst, label, conf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) loadSymIndexes(ctx context.Context) (byFile map[string][]string, byName map[string][]string) {
	byFile = map[string][]string{}
	byName = map[string][]string{}
	rows, err := d.sql.QueryContext(ctx, `SELECT id, path FROM graph_nodes WHERE kind = 'symbol'`)
	if err != nil {
		return byFile, byName
	}
	defer rows.Close()
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return byFile, byName
		}
		byFile[path] = append(byFile[path], id)
		_, name, ok := parseSymID(id)
		if ok {
			byName[name] = append(byName[name], id)
		}
	}
	_ = rows.Err()
	return byFile, byName
}

func importedFilesFor(ctx context.Context, d *DB, rel string) []string {
	fid := fileID(rel)
	rows, err := d.sql.QueryContext(ctx, `
		SELECT dst FROM graph_edges WHERE src = ? AND kind = 'imports' AND dst != ''`, fid)
	if err != nil {
		return nil
	}
	var out []string
	defer rows.Close()
	for rows.Next() {
		var dst string
		if err := rows.Scan(&dst); err != nil {
			return out
		}
		if strings.HasPrefix(dst, "file:") {
			out = append(out, dst[5:])
		}
	}
	_ = rows.Err()
	return out
}
