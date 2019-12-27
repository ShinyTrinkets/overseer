// +build !windows

package signals

import (
	"os"
	"syscall"
)

func SendSignal(pid int, sig syscall.Signal) error {
	proc, err := os.FindProcess(-pid)
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
	localSig := sig.(syscall.Signal)
	pid := process.Pid
	if sigChildren {
		pid = -pid
	}
	return syscall.Kill(pid, localSig)
}
