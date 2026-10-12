package memory

import "context"

func Stats(ctx context.Context) (ContextStats, error) {
	r, err := Context(ctx, ContextOptions{})
	if err != nil {
		return ContextStats{}, err
	}
	return r.Stats, nil
}
