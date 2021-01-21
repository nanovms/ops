package cmd_test

import (
	"testing"
	"time"

	"github.com/nanovms/ops/cmd"
)

var (
	ErrInvalidTimeNotation = func(notation string) string {
		return "expected to throw an error because time notation \"" + notation + "\" is invalid"
	}
)

func TestSubtractDateNotation(t *testing.T) {
	t.Run("should return error if time notation is invalid", func(t *testing.T) {
		layout := "01/02/2006"
		str := "01/10/2020"
		date, _ := time.Parse(layout, str)

		_, err := cmd.SubtractTimeNotation(date, "kjfgd")
		if err == nil {
			t.Errorf(ErrInvalidTimeNotation("kjfgd"))
		}

		_, err = cmd.SubtractTimeNotation(date, "123")
		if err == nil {
			t.Errorf(ErrInvalidTimeNotation("123"))
		}
	})

	t.Run("should return date of days ago", func(t *testing.T) {
		layout := "2006-01-02"
		str := "2020-10-10"
		date, _ := time.Parse(layout, str)

		got, _ := cmd.SubtractTimeNotation(date, "5d")
		want := "2020-10-05"

		if want != got.Format(layout) {
			t.Errorf("got %s, want %s", got.Format(layout), want)
		}
	})

	t.Run("should return date of weeks ago", func(t *testing.T) {
		layout := "2006-01-02"
		str := "2020-10-10"
		date, _ := time.Parse(layout, str)

		got, _ := cmd.SubtractTimeNotation(date, "5w")
		want := "2020-09-05"

		if want != got.Format(layout) {
			t.Errorf("got %s, want %s", got.Format(layout), want)
		}
	})

	t.Run("should return date of months ago", func(t *testing.T) {
		layout := "2006-01-02"
		str := "2020-10-10"
		date, _ := time.Parse(layout, str)

		got, _ := cmd.SubtractTimeNotation(date, "2m")
		want := "2020-08-10"

		if want != got.Format(layout) {
			t.Errorf("got %s, want %s", got.Format(layout), want)
		}
	})

	t.Run("should return date of years ago", func(t *testing.T) {
		layout := "2006-01-02"
		str := "2020-10-10"
		date, _ := time.Parse(layout, str)

		got, _ := cmd.SubtractTimeNotation(date, "3y")
		want := "2017-10-10"

		if want != got.Format(layout) {
			t.Errorf("got %s, want %s", got.Format(layout), want)
		}
	})
}
