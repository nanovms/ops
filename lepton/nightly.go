package lepton

import (
	"path"
	"strings"
)

// NightlyReleaseURLm give URL for nightly build.
var NightlyReleaseURLm = NightlyReleaseURL()

func nightlybase() string {
	if AltGOARCH != "" {
		if AltGOARCH == "arm64" {
			return "nanos-nightly-linux-virt"
		}

		return "nanos-nightly-linux"

	} else {
		if RealGOARCH == "arm64" {
			return "nanos-nightly-linux-virt"
		}

		return "nanos-nightly-linux"
	}
}

func nightlyFileName() string {
	return nightlybase() + ".tar.gz"
}

func nightlyTimestamp() string {
	return nightlybase() + ".timestamp"
}

// NightlyReleaseURL points to the latest nightly release url that is
// arch dependent upon flag set.
func NightlyReleaseURL() string {
	var sb strings.Builder
	sb.WriteString(nightlyReleaseBaseURL)
	sb.WriteString(nightlyFileName())
	return sb.String()
}

// NightlyLocalFolder points to the latest nightly release url that is
// arch dependent upon flag set.
func NightlyLocalFolder() string {
	if AltGOARCH != "" {
		if AltGOARCH == "arm64" {
			return path.Join(GetOpsHome(), "nightly-arm")
		}

		return path.Join(GetOpsHome(), "nightly")
	} else {
		if RealGOARCH == "arm64" {
			return path.Join(GetOpsHome(), "nightly-arm")
		}

		return path.Join(GetOpsHome(), "nightly")
	}
}

// NightlyLocalFolderm is directory path where nightly builds are stored
var NightlyLocalFolderm = NightlyLocalFolder()
