//go:build windows

package selfupdate

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procMoveFileEx = kernel32.NewProc("MoveFileExW")
)

// MOVEFILE_DELAY_UNTIL_REBOOT schedules the move for the next boot,
// when the running image is no longer locked.
const movefileDelayUntilReboot = 0x4

// swap installs bin over exe on Windows.
//
// A running .exe is locked: it can neither be renamed nor overwritten
// in place (os.Rename / copy-onto fail with access denied). So:
//  1. Stage the new binary as exe.new and back up the running exe to
//     exe.old by copy (reading a running image is allowed).
//  2. Fast path: rename exe.new onto exe. Works when exe is not locked
//     (tests, copies that are not executing).
//  3. Locked path: schedule exe.new -> exe via MoveFileEx with
//     MOVEFILE_DELAY_UNTIL_REBOOT and report that a reboot is required.
func swap(exe, bin string) error {
	old := exe + ".old"
	os.Remove(old) // stale rollback from a previous run

	staged := exe + ".new"
	os.Remove(staged) // stale stage from a previous run
	if err := fileutil.CopyFile(bin, staged, fileutil.ExePerm); err != nil {
		return fmt.Errorf("stage new binary: %w", err)
	}
	// Best-effort backup by copy; reads of the running image are allowed.
	if err := fileutil.CopyFile(exe, old, fileutil.ExePerm); err != nil {
		os.Remove(staged)
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(staged, exe); err == nil {
		return nil
	}
	if err := moveFileExDelayUntilReboot(staged, exe); err != nil {
		return fmt.Errorf("replace locked binary: %w", err)
	}
	return fmt.Errorf("update staged at %s — reboot to complete the replace of %s (backup at %s)", staged, exe, old)
}

func moveFileExDelayUntilReboot(src, dst string) error {
	src16, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	dst16, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	r1, _, e1 := procMoveFileEx.Call(
		uintptr(unsafe.Pointer(src16)),
		uintptr(unsafe.Pointer(dst16)),
		uintptr(movefileDelayUntilReboot),
	)
	if r1 == 0 {
		if e1 != nil && e1 != syscall.Errno(0) {
			return e1
		}
		return fmt.Errorf("MoveFileEx failed")
	}
	return nil
}
