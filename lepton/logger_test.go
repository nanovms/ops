package lepton_test

import (
	"bytes"
	"testing"

	"github.com/nanovms/ops/lepton"
)

const (
	newline = "\n"
)

func TestLogger(t *testing.T) {
	t.Run("Log should print to output", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

		logger.Log("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := "test 1,2,3" + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Info should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

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
		logger := lepton.NewLogger(&b)

		logger.SetInfo(true)
		logger.Info("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := lepton.ConsoleColors.Blue() + "test 1,2,3" + lepton.ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Warn should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

		logger.Warn("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ""

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Warn should print if set", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

		logger.SetWarn(true)
		logger.Warn("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := lepton.ConsoleColors.Yellow() + "test 1,2,3" + lepton.ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Error should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

		logger.Error("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ""

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Error should print if set", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

		logger.SetError(true)
		logger.Error("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := lepton.ConsoleColors.Red() + "test 1,2,3" + lepton.ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Debug should not print to output by default", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

		logger.Debug("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := ""

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("Debug should print if set", func(t *testing.T) {
		var b bytes.Buffer
		logger := lepton.NewLogger(&b)

		logger.SetDebug(true)
		logger.Debug("test %d,%d,%d", 1, 2, 3)

		got := b.String()
		want := lepton.ConsoleColors.Cyan() + "test 1,2,3" + lepton.ConsoleColors.White() + newline

		if got != want {
			t.Errorf("got %v want %v", got, want)
		}
	})
}
