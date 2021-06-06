package log

import (
	"fmt"
	"io"
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

// Log writes a message to the specified output
func (l *Logger) Log(message string, a ...interface{}) {
	fmt.Fprintf(l.output, message+"\n", a...)
}

// Info checks info level is activated to write the message
func (l *Logger) Info(message string, a ...interface{}) {
	if l.info == true {
		l.Log(ConsoleColors.Blue()+message+ConsoleColors.Reset(), a...)
	}
}

// Warn checks warn level is activated to write the message
func (l *Logger) Warn(message string, a ...interface{}) {
	if l.warn == true {
		l.Log(ConsoleColors.Yellow()+message+ConsoleColors.Reset(), a...)
	}
}

// Errorf checks error level is activated to write the formatted message
func (l *Logger) Errorf(message string, a ...interface{}) {
	l.Log(ConsoleColors.Red()+message+ConsoleColors.Reset(), a...)
}

// Error checks error level is activated to write error object
func (l *Logger) Error(err error) {
	l.Log(ConsoleColors.Red() + err.Error() + ConsoleColors.Reset())
}

// Debug checks debug level is activated to write the message
func (l *Logger) Debug(message string, a ...interface{}) {
	if l.debug == true {
		l.Log(ConsoleColors.Cyan()+message+ConsoleColors.Reset(), a...)
	}
}
