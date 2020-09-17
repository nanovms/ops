package lepton

import (
	"errors"
)

// sysKill wraps syscall.Kill
func sysKill(pid int) error {
	return errors.New("not supported")
}
