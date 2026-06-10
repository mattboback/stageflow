//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// Windows has no POSIX process groups; signalling falls back to the direct
// os.Process handle in sendSignal.
func setProcessGroup(_ *exec.Cmd) {}

func signalProcessGroup(_ *os.Process, _ syscall.Signal) bool {
	return false
}
