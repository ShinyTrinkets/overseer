// +build windows

package overseer

import "syscall"

func setSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
