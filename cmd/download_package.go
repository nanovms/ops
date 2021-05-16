package cmd

import (
	"os"
	"path"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
)

func downloadLocalPackage(pkg string) string {
	packagesDirPath := path.Join(api.GetOpsHome(), "local_packages")
	return downloadAndExtractPackage(packagesDirPath, pkg)
}

func packageDirectoryPath() string {
	return path.Join(api.GetOpsHome(), "packages")
}

func downloadPackage(pkg string) string {
	return downloadAndExtractPackage(packageDirectoryPath(), pkg)
}

func downloadAndExtractPackage(packagesDirPath, pkg string) string {
	err := os.MkdirAll(packagesDirPath, 0755)
	if err != nil {
		log.Fatal(err)
	}

	expackage := path.Join(packagesDirPath, pkg)
	opsPackage, err := api.DownloadPackage(pkg)
	if err != nil {
		log.Fatal(err)
	}

	api.ExtractPackage(opsPackage, packagesDirPath)

	err = os.Remove(opsPackage)
	if err != nil {
		log.Fatal(err)
	}

	return expackage
}
