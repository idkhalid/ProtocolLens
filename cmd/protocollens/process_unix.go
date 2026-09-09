//go:build !windows

package main

import (
	"syscall"
)

func configureCaptureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func requestCaptureStop(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Send SIGTERM to the process group
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

func forceKillCaptureTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// Sleep gracefully handled in caller
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
