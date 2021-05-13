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

		logger.Log("test %d,%d,%d", 1, 2, 3)

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
		logger.Info("test %d,%d,%d", 1, 2, 3)

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
		logger.Info("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ConsoleColors.Blue() + "test 1,2,3" + ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Warn should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.Warn("test %d,%d,%d", 1, 2, 3)

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
		logger.Warn("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ConsoleColors.Yellow() + "test 1,2,3" + ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Error should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.Errorf("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ""

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Error should print if set", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.SetError(true)
		logger.Errorf("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ConsoleColors.Red() + "test 1,2,3" + ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Debug should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := New(&b)

		logger.Debug("test %d,%d,%d", 1, 2, 3)

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
		logger.Debug("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ConsoleColors.Cyan() + "test 1,2,3" + ConsoleColors.White() + newline

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
		want := ConsoleColors.Red() + "something wrong is happening" + ConsoleColors.White() + newline

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
		want := ConsoleColors.Red() + "something fishy is happening" + ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})
}
