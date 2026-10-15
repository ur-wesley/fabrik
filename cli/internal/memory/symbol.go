package memory

import (
	"context"
)

type SymbolObservation struct {
	ID    string
	Title string
	Kind  string
}

func ObservationsForSymbol(ctx context.Context, project, symbolID string) ([]SymbolObservation, error) {
	db, err := OpenReadOnly(ctx)
	if err != nil || db == nil {
		return nil, err
	}
	defer releaseDB(db)
	rows, err := db.QueryContext(ctx, `
		SELECT id, title, kind FROM observations
		WHERE deleted_at IS NULL AND project = ? AND symbol_id = ?`, project, symbolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SymbolObservation
	for rows.Next() {
		var o SymbolObservation
		if err := rows.Scan(&o.ID, &o.Title, &o.Kind); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
