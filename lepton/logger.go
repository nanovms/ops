package lepton

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

// NewLogger returns an instance of Logger
func NewLogger(output io.Writer) *Logger {
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
		l.Log(ConsoleColors.Blue()+message+ConsoleColors.White(), a...)
	}
}

// Warn checks warn level is activated to write the message
func (l *Logger) Warn(message string, a ...interface{}) {
	if l.warn == true {
		l.Log(ConsoleColors.Yellow()+message+ConsoleColors.White(), a...)
	}
}

// Error checks error level is activated to write the message
func (l *Logger) Error(message string, a ...interface{}) {
	if l.err == true {
		l.Log(ConsoleColors.Red()+message+ConsoleColors.White(), a...)
	}
}

// Debug checks debug level is activated to write the message
func (l *Logger) Debug(message string, a ...interface{}) {
	if l.debug == true {
		l.Log(ConsoleColors.Cyan()+message+ConsoleColors.White(), a...)
	}
}
