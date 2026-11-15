package cmd

import "runtime"

func getPkgArch() string {
	rt := runtime.GOARCH
	if rt == "amd64" {
		return "x86_64"
	}

	return rt
}

func getPackageArch(parch string) string {
	if parch == "amd64" {
		return "x86_64"
	} else {
		return parch
	}
}

func getPreferredArch(archflag string) string {
	if archflag != "" {
		return archflag
	}

	return runtime.GOARCH
}
