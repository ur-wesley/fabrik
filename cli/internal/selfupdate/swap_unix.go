//go:build !windows

package selfupdate

import (
	"fmt"
	"os"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
)

// swap renames exe to exe.old and moves bin into place.
// On Unix the running binary can be renamed while it executes,
// so the old image keeps running and the new one takes effect
// on next launch.
func swap(exe, bin string) error {
	old := exe + ".old"
	os.Remove(old) // stale rollback from a previous run
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := fileutil.CopyFile(bin, exe, fileutil.ExePerm); err != nil {
		// best-effort rollback
		_ = os.Rename(old, exe)
		return fmt.Errorf("install new binary: %w", err)
	}
	return nil
}
