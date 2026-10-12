package store

import "errors"

var ErrNoStore = errors.New("fabrik: no store (run fabrik init)")

func RequireRepo(start string) (string, error) {
	root, ok := RepoRoot(start)
	if !ok {
		return "", ErrNoStore
	}
	return root, nil
}
