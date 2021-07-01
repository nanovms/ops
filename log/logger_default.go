package log

import (
	"fmt"
	"io"
	"os"

	"github.com/nanovms/ops/types"
)

var defaultLogger *Logger

// Make sure default logger instantiated by default.
func init() {
	defaultLogger = New(os.Stdout)
}

// InitDefault creates default logger for package-level logging access.
func InitDefault(output io.Writer, config *types.Config) {
	defaultLogger = New(output)

	if config == nil {
		return
	}

	if config.RunConfig.ShowDebug {
		defaultLogger.SetDebug(true)
		defaultLogger.SetWarn(true)
		defaultLogger.SetError(true)
		defaultLogger.SetInfo(true)
	}

	if config.RunConfig.ShowWarnings {
		defaultLogger.SetWarn(true)
	}

	if config.RunConfig.ShowErrors {
		defaultLogger.SetError(true)
	}

	if config.RunConfig.Verbose {
		defaultLogger.SetInfo(true)
	}
}

// Info logs info-level message using default logger.
func Info(a ...interface{}) {
	defaultLogger.Info(a...)
}

// Infof logs formatted info-level message using default logger.
func Infof(format string, a ...interface{}) {
	defaultLogger.Infof(format, a...)
}

// Warn logs warning-level message using default logger.
func Warn(a ...interface{}) {
	defaultLogger.Warn(a...)
}

// Warnf logs formatted warning-level message using default logger.
func Warnf(format string, a ...interface{}) {
	defaultLogger.Warnf(format, a...)
}

// Errorf logs formatted error-level formatted string message using default logger.
func Errorf(format string, a ...interface{}) {
	defaultLogger.Errorf(format, a...)
}

// Error logs error-level message using default logger.
func Error(err error) {
	defaultLogger.Error(err)
}

// Fatalf logs formatted error-level formatted string message using default logger then calls os.Exit(1).
func Fatalf(format string, a ...interface{}) {
	defaultLogger.Errorf(format, a...)
	os.Exit(1)
}

// Fatal logs error-level message using default logger then calls os.Exit(1).
func Fatal(err error) {
	defaultLogger.Error(err)
	os.Exit(1)
}

// Panicf logs formatted error-level formatted string message using default logger then calls panic().
func Panicf(format string, a ...interface{}) {
	defaultLogger.Errorf(format, a...)
	panic(fmt.Sprintf(format+"\n", a...))
}

// Panic logs error-level message using default logger then calls panic().
func Panic(err error) {
	defaultLogger.Error(err)
	panic(err)
}

// Debug logs debug-level message using default logger.
func Debug(a ...interface{}) {
	defaultLogger.Debug(a...)
}

// Debugf logs formatted debug-level message using default logger.
func Debugf(format string, a ...interface{}) {
	defaultLogger.Debugf(format, a...)
}
