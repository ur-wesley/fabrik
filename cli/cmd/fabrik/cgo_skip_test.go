package main

import (
	"testing"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
)

// skipWithoutCGO skips tests that need the embedded Dolt store when the
// test binary was built without cgo. It must use doltstore.HasCGO
// (compile-time, via build tags) rather than the CGO_ENABLED environment
// variable, which is a build-time setting and usually unset at test runtime.
func skipWithoutCGO(t *testing.T) {
	t.Helper()
	if !doltstore.HasCGO {
		t.Skip("requires CGO_ENABLED=1 (embedded Dolt)")
	}
}
