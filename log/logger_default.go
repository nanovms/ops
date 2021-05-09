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
	defaultLogger.SetWarn(true)
	defaultLogger.SetError(true)
}

// InitDefaultLogger creates default logger for package-level logging access.
func InitDefault(output io.Writer, config *types.Config) {
	defaultLogger = New(output)
	defaultLogger.SetError(true)

	if config == nil {
		defaultLogger.SetWarn(true)
		return
	}

	if config.RunConfig.ShowDebug {
		defaultLogger.SetDebug(true)
		defaultLogger.SetWarn(true)
		defaultLogger.SetInfo(true)
	}

	if config.RunConfig.ShowWarnings {
		defaultLogger.SetWarn(true)
	}

	if config.RunConfig.Verbose {
		defaultLogger.SetInfo(true)
	}
}

// Info logs info-level message using default logger.
func Info(message string, a ...interface{}) {
	defaultLogger.Info(message, a...)
}

// Warn logs warning-level message using default logger.
func Warn(message string, a ...interface{}) {
	defaultLogger.Warn(message, a...)
}

// Error logs error-level message using default logger.
func Error(message string, a ...interface{}) {
	defaultLogger.Error(message, a...)
}

// Fatal logs error-level message using default logger then calls os.Exit(1).
func Fatal(message string, a ...interface{}) {
	defaultLogger.Error(message, a...)
	os.Exit(1)
}

// Panic logs error-level message using default logger then calls panic().
func Panic(message string, a ...interface{}) {
	defaultLogger.Error(message, a...)
	panic(fmt.Sprintf(message+"\n", a...))
}

// Debug logs debug-level message using default logger.
func Debug(message string, a ...interface{}) {
	defaultLogger.Debug(message, a...)
}
