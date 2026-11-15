package cmd

import (
	"runtime"
)

func getPkgArch() string {
	rt := runtime.GOARCH
	if rt == "amd64" {
		return "x86_64"
	}

	return rt
}

// only for distinguishing between legacy 'x86_64' package types and
// arm64
func getPackageArch(parch string) string {
	if parch == "amd64" {
		return "x86_64"
	}

	return parch
}

func getPreferredArch(archflag string) string {
	if archflag != "" {
		if runtime.GOARCH == "arm64" && archflag != "arm64" {
			return "amd64"
		}

		return archflag
	}

	return runtime.GOARCH
}
