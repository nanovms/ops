package lepton

import (
	"path"
	"strings"
)

// NightlyReleaseURLm give URL for nightly build.
var NightlyReleaseURLm = NightlyReleaseURL()

func nightlyFileName() string {
	if AltGOARCH != "" {
		if AltGOARCH == "arm64" {
			return "nanos-nightly-linux-virt.tar.gz"
		}

		return "nanos-nightly-linux.tar.gz"

	} else {
		if RealGOARCH == "arm64" {
			return "nanos-nightly-linux-virt.tar.gz"
		}

		return "nanos-nightly-linux.tar.gz"
	}
}

func nightlyTimestamp() string {
	if AltGOARCH != "" {
		if AltGOARCH == "arm64" {
			return "nanos-nightly-linux-virt.timestamp"
		}

		return "nanos-nightly-linux.timestampz"

	} else {
		if RealGOARCH == "arm64" {
			return "nanos-nightly-linux-virt.timestamp"
		}

		return "nanos-nightly-linux.timestamp"
	}
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
