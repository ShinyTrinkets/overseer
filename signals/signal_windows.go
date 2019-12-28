// +build windows

package signals

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func SendSignal(pid int, sig syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	err = proc.Signal(sig)

	if err != nil {
		return err
	}

	return nil
}

func Kill(process *os.Process, sig os.Signal, sigChildren bool) error {
	// Signal command can't kill children processes, call  taskkill command to kill them
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", process.Pid))
	err := cmd.Start()
	if err == nil {
		return cmd.Wait()
	}
	// if fail to find taskkill, fallback to normal signal
	return process.Signal(sig)
}
