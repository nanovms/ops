package qemu

import (
	"errors"
)

type errCustom struct {
	Msg   string
	Cause error
}

func (e *errCustom) Error() string {
	if e.Cause == nil {
		return e.Msg
	}
	return e.Msg + ": " + e.Cause.Error()
}

func (e *errCustom) Unwrap() error {
	return e.Cause
}

type errQemuHWAccelDisabledInConfig struct{ errCustom }
type errQemuNotInstalled struct{ errCustom }
type errQemuCannotExecute struct{ errCustom }
type errQemuCannotGetQemuVersion struct{ errCustom }
type errQemuHWAccelNotSupported struct{ errCustom }
type errQemuHWAccelNoUserRights struct{ errCustom }

func qemuAccelWarningMessage(err error) (message string, terminate bool) {
	var (
		targetErrQemuHWAccelDisabledInConfig *errQemuHWAccelDisabledInConfig
		targetErrQemuNotInstalled            *errQemuNotInstalled
		targetErrQemuCannotExecute           *errQemuCannotExecute
		targetQemuHWAccelNoUserRights        *errQemuHWAccelNoUserRights
		targetQemuHWAccelNotSupported        *errQemuHWAccelNotSupported
	)
	if errors.As(err, &targetErrQemuHWAccelDisabledInConfig) {
		return "You have disabled hardware acceleration", false
	}
	if errors.As(err, &targetErrQemuNotInstalled) {
		return "Cannot find QEMU (looks like it is not installed)\n" +
			"Please install QEMU using your package manager and re-run current command\n", true
	}
	if errors.As(err, &targetErrQemuCannotExecute) {
		return "QEMU installed, but cannot be executed\n" +
			"Please check current user rights\n", true
	}

	if errors.As(err, &targetQemuHWAccelNoUserRights) {
		return "You don't have rights for using hardware acceleration\n" +
			"Try adding yourself to the kvm group: `sudo adduser $user kvm`\n" +
			"You'll need to re-login for this to take affect\n", false
	}

	if errors.As(err, &targetQemuHWAccelNotSupported) {
		return "You specified hardware acceleration, but it is not supported\n" +
			"Are you running inside a vm? If so disable accel with --accel=false\n", false
	}

	return "Hardware acceleration cannot be used on the current host", false
}
