//go:build windows

package listen

import "os"

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if pid == os.Getpid() {
		return true
	}
	// os.FindProcess on Windows calls OpenProcess: success means the
	// PID exists and we could open it; failure means it is gone (or
	// access denied, treated as not alive — the HTTP probe in
	// existingListenBusy covers the truly-busy case).
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	defer p.Release()
	return true
}
