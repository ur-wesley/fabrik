package graph

import "errors"

var (
	ErrEmpty = errors.New("fabrik: graph is empty (run fabrik graph build)")
)
