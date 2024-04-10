package lepton

import "testing"

func TestNightlyFilename(t *testing.T) {
	RealGOARCH = "amd64"

	fname := nightlybase()

	if fname != "nanos-nightly-linux" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

	RealGOARCH = "arm64"

	fname = nightlybase()

	if fname != "nanos-nightly-linux-virt" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

	RealGOARCH = "arm64"
	AltGOARCH = "amd64"

	fname = nightlybase()

	if fname != "nanos-nightly-linux" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

	RealGOARCH = "amd64"
	AltGOARCH = "arm64"

	fname = nightlybase()

	if fname != "nanos-nightly-linux-virt" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

}
