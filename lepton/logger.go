package lepton

import (
	"fmt"
	"io"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
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
		l.Log(string(colorBlue)+message, a...)
	}
}

// Warn checks warn level is activated to write the message
func (l *Logger) Warn(message string, a ...interface{}) {
	if l.warn == true {
		l.Log(string(colorYellow)+message, a...)
	}
}

// Error checks error level is activated to write the message
func (l *Logger) Error(message string, a ...interface{}) {
	if l.err == true {
		l.Log(string(colorRed)+message, a...)
	}
}

// Debug checks debug level is activated to write the message
func (l *Logger) Debug(message string, a ...interface{}) {
	if l.debug == true {
		l.Log(string(colorPurple)+message, a...)
	}
}
