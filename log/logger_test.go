package log

import (
	"bytes"
	"errors"
	"testing"
)

const (
	newline = "\n"
)

func TestLogger(t *testing.T) {
	t.Run("Log should print to output", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.Logf("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := "test 1,2,3" + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Info should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.SetInfo(false)
		logger.Infof("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ""

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Info should print to output by default if set", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.SetInfo(true)
		logger.Infof("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ConsoleColors.Blue() + "test 1,2,3" + ConsoleColors.Reset() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Warn should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.Warnf("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ""

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Warn should print if set", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.SetWarn(true)
		logger.Warnf("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ConsoleColors.Yellow() + "test 1,2,3" + ConsoleColors.Reset() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Debug should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.Debugf("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ""

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Debug should print if set", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.SetDebug(true)
		logger.Debugf("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ConsoleColors.Cyan() + "test 1,2,3" + ConsoleColors.Reset() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})
}

func TestLoggerError(t *testing.T) {
	t.Run("Log Error should print error string to output", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)
		logger.SetError(true)

		logger.Error(errors.New("something wrong is happening"))

		got := b.String()
		want := ConsoleColors.Red() + "something wrong is happening" + ConsoleColors.Reset() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Log Errorf should print formatted string to output", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)
		logger.SetError(true)

		logger.Errorf("something %s is happening", "fishy")

		got := b.String()
		want := ConsoleColors.Red() + "something fishy is happening" + ConsoleColors.Reset() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})
}
