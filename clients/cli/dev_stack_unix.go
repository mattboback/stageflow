//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcessGroup runs the dev server in its own process group so stop
// signals reach child processes (e.g. a package manager spawning the server).
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalProcessGroup(p *os.Process, sig syscall.Signal) bool {
	return syscall.Kill(-p.Pid, sig) == nil
}
