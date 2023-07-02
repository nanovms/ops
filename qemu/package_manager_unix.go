package qemu

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
)

type PackageManager int32

const (
	// Linux
	PackageManagerNotFound PackageManager = iota
	PackageManagerYum
	PackageManagerAptGet
	PackageManagerApk
	PackageManagerEmerge
	PackageManagerZypp
	PackageManagerPacman
	PackageManagerDNF

	// Mac
	PackageManagerBrew
	PackageManagerMacPorts

	// Windows
	PackageManagerWindowsBinaries
	PackageManagerWindowsMSYS2
)

var (
	distribsPackageManagers = map[string]PackageManager{
		"/etc/redhat-release": PackageManagerYum,
		"/etc/debian_version": PackageManagerAptGet,
		"/etc/alpine-release": PackageManagerApk,
		"/etc/gentoo-release": PackageManagerEmerge,
		"/etc/SuSE-release":   PackageManagerZypp,
		"/etc/arch-release":   PackageManagerPacman,
		"/etc/fedora-release": PackageManagerDNF,
	}
)

func (pm PackageManager) InstallQEMUCommand() string {
	switch pm {
	// Not Found
	case PackageManagerNotFound:
		return "https://www.qemu.org/download"

	// Linux
	case PackageManagerYum:
		return "yum install qemu-kvm"
	case PackageManagerAptGet:
		return "apt-get install qemu"
	case PackageManagerApk:
		return "sudo apt install qemu"
	case PackageManagerEmerge:
		return "emerge --ask app-emulation/qemu"
	case PackageManagerZypp:
		return "zypper install qemu"
	case PackageManagerPacman:
		return "pacman -S qemu"
	case PackageManagerDNF:
		return "dnf install @virtualization"

	// Mac OS
	case PackageManagerBrew:
		return "brew install qemu"
	case PackageManagerMacPorts:
		return "sudo port install qemu"

	// Windows
	case PackageManagerWindowsBinaries:
		return "https://www.qemu.org/download/#windows"
	case PackageManagerWindowsMSYS2:
		return "https://www.qemu.org/download/#windows"
	}
	return "https://www.qemu.org/download"
}

func (pm *PackageManager) detectManager() {
	*pm = PackageManagerNotFound

	if runtime.GOOS == "windows" {
		*pm = PackageManagerWindowsBinaries
	}

	if runtime.GOOS == "darwin" {
		if pm.isBrewInstalled() {
			*pm = PackageManagerBrew
		}
		if pm.isMacPortsInstalled() {
			*pm = PackageManagerMacPorts
		}
	}

	if runtime.GOOS == "linux" {
		for path, manager := range distribsPackageManagers {
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				continue
			}
			*pm = manager
		}
	}
}

func (pm *PackageManager) isBrewInstalled() bool {
	c := exec.Command("which", "brew")
	if err := c.Run(); err != nil {
		return false
	}
	return true
}

func (pm *PackageManager) isMacPortsInstalled() bool {
	c := exec.Command("which", "port")
	if err := c.Run(); err != nil {
		return false
	}
	return true
}
