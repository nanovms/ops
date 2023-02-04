package onprem

import (
	"syscall"
)

// sysKill wraps syscall.Kill
func sysKill(pid int) error {
	return syscall.Kill(pid, 9)
}
