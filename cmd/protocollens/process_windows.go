//go:build windows

package main

import (
	"os/exec"
	"strconv"
)

func configureCaptureProcess(cmd *exec.Cmd) {
	// No special sys proc attr needed for the fallback strategy.
}

func requestCaptureStop(cmd *exec.Cmd) error {
	// Graceful stop on Windows requires excessive complexity (GenerateConsoleCtrlEvent + CREATE_NEW_PROCESS_GROUP).
	// For T7A, Windows uses forced taskkill as the explicit fallback.
	// We don't do anything here, we just let the grace period expire and then forceKillCaptureTree.
	return nil
}

func forceKillCaptureTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// Do NOT call cmd.Process.Kill() first. Use taskkill to kill the entire tree.
	pid := strconv.Itoa(cmd.Process.Pid)
	tk := exec.Command("taskkill.exe", "/T", "/F", "/PID", pid)
	tk.Run()
}
