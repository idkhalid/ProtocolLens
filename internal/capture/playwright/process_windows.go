//go:build windows

package playwright

import (
	"os/exec"
	"strconv"
)

func configureProcess(cmd *exec.Cmd) {}

func requestStop(cmd *exec.Cmd) error { return nil }

func forceKillTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = exec.Command("taskkill.exe", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
