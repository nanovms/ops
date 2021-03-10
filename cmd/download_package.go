package cmd

import (
	"fmt"
	"os"
	"path"

	api "github.com/nanovms/ops/lepton"
)

func downloadLocalPackage(pkg string) string {
	packagesDirPath := path.Join(api.GetOpsHome(), "local_packages")
	return downloadAndExtractPackage(packagesDirPath, pkg)
}

func downloadPackage(pkg string) string {
	packagesDirPath := path.Join(api.GetOpsHome(), "packages")
	return downloadAndExtractPackage(packagesDirPath, pkg)
}

func downloadAndExtractPackage(packagesDirPath, pkg string) string {
	err := os.MkdirAll(packagesDirPath, 0755)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	expackage := path.Join(packagesDirPath, pkg)
	opsPackage, err := api.DownloadPackage(pkg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	api.ExtractPackage(opsPackage, packagesDirPath)

	err = os.Remove(opsPackage)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return expackage
}
