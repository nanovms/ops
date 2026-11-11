package lepton

import (
	"path"
	"strings"
)

func nightlybase(parch string) string {
	if parch == "arm64" {
		return "nanos-nightly-linux-virt"
	}

	return "nanos-nightly-linux"
}

func nightlyFileName(parch string) string {
	return nightlybase(parch) + ".tar.gz"
}

func nightlyTimestamp(parch string) string {
	return nightlybase(parch) + ".timestamp"
}

// NightlyReleaseURL points to the latest nightly release url that is
// arch dependent upon flag set.
func NightlyReleaseURL(parch string) string {
	var sb strings.Builder
	sb.WriteString(nightlyReleaseBaseURL)
	sb.WriteString(nightlyFileName(parch))
	return sb.String()
}

// NightlyLocalFolder points to the latest nightly release url that is
// arch dependent upon flag set.
func NightlyLocalFolder(parch string) string {
	if parch == "arm64" {
		return path.Join(GetOpsHome(), "nightly-arm")
	}

	return path.Join(GetOpsHome(), "nightly")

}
