package lepton

import "testing"

func TestNightlyFilename(t *testing.T) {
	RealGOARCH = "amd64"

	fname := nightlyFileName()

	if fname != "nanos-nightly-linux.tar.gz" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

	RealGOARCH = "arm64"

	fname = nightlyFileName()

	if fname != "nanos-nightly-linux-virt.tar.gz" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

	RealGOARCH = "arm64"
	AltGOARCH = "amd64"

	fname = nightlyFileName()

	if fname != "nanos-nightly-linux.tar.gz" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

	RealGOARCH = "amd64"
	AltGOARCH = "arm64"

	fname = nightlyFileName()

	if fname != "nanos-nightly-linux-virt.tar.gz" {
		t.Fatalf("invalid nightly filename: %s", fname)
	}

}
