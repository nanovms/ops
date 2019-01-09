package lepton

import (
	"os"
	"path"
	"path/filepath"
)

// file system manifest
const manifest string = `(
    #64 bit elf to boot from host
    children:(kernel:(contents:(host:%v))
              #user program
              %v:(contents:(host:%v)))
    # filesystem path to elf for kernel to run
    program:/%v
    fault:t
    arguments:[%v sec third]
    environment:(USER:bobby PWD:/)
)`

// boot loader
const BootImg string = ".staging/boot.img"

// kernel
const KernelImg string = ".staging/stage3.img"

// kernel + ELF image
const mergedImg string = ".staging/tempimage"

// final bootable image
const FinalImg string = "image"

const Mkfs string = ".staging/mkfs"

const ReleaseBaseUrl string = "https://storage.googleapis.com/nanos/release/%v"
const DevBaseUrl string = "https://storage.googleapis.com/nanos/%v"

const PackageBaseURL string = "https://storage.googleapis.com/packagehub/%v"
const PackageManifestURL string = "https://storage.googleapis.com/packagehub/manifest.json"
const PackageManifest string = ".staging/manifest.json"

var PackagesCache string

func GetPackageCache() string {
	if PackagesCache == "" {
		e, err := os.Executable()
		if err != nil {
			panic(err)
		}
		PackagesCache = path.Join(filepath.Dir(e), ".packages")
		if _, err := os.Stat(PackagesCache); os.IsNotExist(err) {
			os.MkdirAll(PackagesCache, 0755)
		}
	}
	return PackagesCache
}
