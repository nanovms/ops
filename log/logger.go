package log

import (
	"fmt"
	"io"
	"strings"
)

// Logger filters and prints messages to a destination
type Logger struct {
	output io.Writer
	info   bool
	warn   bool
	err    bool
	debug  bool
}

// New returns an instance of Logger
func New(output io.Writer) *Logger {
	return &Logger{output, false, false, false, false}
}

// SetInfo activates/deactivates info level
func (l *Logger) SetInfo(value bool) {
	l.info = value
}

// SetWarn activates/deactivates warn level
func (l *Logger) SetWarn(value bool) {
	l.warn = value
}

// SetError activates/deactivates error level
func (l *Logger) SetError(value bool) {
	l.err = value
}

// SetDebug activates/deactivates debug level
func (l *Logger) SetDebug(value bool) {
	l.debug = value
}

// Logf writes a formatted message to the specified output
func (l *Logger) Logf(format string, a ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	fmt.Fprintf(l.output, format, a...)
}

// Log writes message to the specified output
func (l *Logger) Log(a ...interface{}) {
	fmt.Fprintln(l.output, a...)
}

// Calls Log with foreground color set
func (l *Logger) logWithColor(color string, a ...interface{}) {
	msg := fmt.Sprintln(a...)
	l.Log(color + strings.TrimSuffix(msg, "\n") + ConsoleColors.Reset())
}

// Info checks info level is activated to write the message
func (l *Logger) Info(a ...interface{}) {
	if l.info == true {
		l.logWithColor(ConsoleColors.Blue(), a...)
	}
}

// Infof checks info level is activated to write the formatted message
func (l *Logger) Infof(format string, a ...interface{}) {
	if l.info == true {
		l.Logf(ConsoleColors.Blue()+format+ConsoleColors.Reset(), a...)
	}
}

// Warn checks warn level is activated to write the message
func (l *Logger) Warn(a ...interface{}) {
	if l.warn == true {
		l.logWithColor(ConsoleColors.Yellow(), a...)
	}
}

// Warnf checks warn level is activated to write the formatted message
func (l *Logger) Warnf(format string, a ...interface{}) {
	if l.warn == true {
		l.Logf(ConsoleColors.Yellow()+format+ConsoleColors.Reset(), a...)
	}
}

// Error checks error level is activated to write error object
func (l *Logger) Error(err error) {
	l.logWithColor(ConsoleColors.Red(), err.Error())
}

// Errorf checks error level is activated to write the formatted message
func (l *Logger) Errorf(format string, a ...interface{}) {
	l.Logf(ConsoleColors.Red()+format+ConsoleColors.Reset(), a...)
}

// Debug checks debug level is activated to write the message
func (l *Logger) Debug(a ...interface{}) {
	if l.debug == true {
		l.logWithColor(ConsoleColors.Cyan(), a...)
	}
}

// Debugf checks debug level is activated to write the message
func (l *Logger) Debugf(format string, a ...interface{}) {
	if l.debug == true {
		l.Logf(ConsoleColors.Cyan()+format+ConsoleColors.Reset(), a...)
	}
}
