package lepton_test

import (
	"testing"

	"github.com/nanovms/ops/lepton"
)

func TestGetEnaSupport(t *testing.T) {

	t.Run("should return false if flavor is not specified", func(t *testing.T) {

		got := lepton.GetEnaSupportForFlavor("")
		want := false

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("should return false if flavor is not built in nitro system", func(t *testing.T) {

		got := lepton.GetEnaSupportForFlavor("t2.micro")
		want := false

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("should return true if flavor is built in nitro system", func(t *testing.T) {

		got := lepton.GetEnaSupportForFlavor("t3.nano")
		want := true

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

}
