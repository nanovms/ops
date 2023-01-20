//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package cmd

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	api "github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/qemu"
	"github.com/spf13/cobra"
)

// Profile holds a particular host's configuration that is used to
// troubleshoot different environments. This is separate and different
// from a runtime cache.
type Profile struct {
	OpsVersion   string
	NanosVersion string
	QemuVersion  string
	Arch         string
	Hypervisor   bool
}

func (p *Profile) save() {

	str := fmt.Sprintf("ops version:%s\nnanos version:%s\nqemu version:%s\narch:%s",
		p.OpsVersion, p.NanosVersion, p.QemuVersion, p.Arch)

	local := path.Join(api.GetOpsHome(), "profile")

	err := os.WriteFile(local, []byte(str), 0644)
	if err != nil {
		log.Error(err)
	}
}

func (p *Profile) setProfile() {

	p.OpsVersion = api.Version
	p.NanosVersion = api.LocalReleaseVersion

	qv, err := qemu.Version()
	if err != nil {
		log.Error(err)
	}

	p.QemuVersion = qv

	p.Arch = runtime.GOOS

	p.Hypervisor = p.virtualized()

	p.save()

}

func (p *Profile) virtualized() bool {
	s := "/proc/cpuinfo"

	_, err := os.Stat(s)
	if os.IsNotExist(err) {
		return false
	}

	content, err := os.ReadFile(s)
	if err != nil {
		log.Error(err)
	}

	text := string(content)

	return strings.Contains(text, "hypervisor")
}

func (p *Profile) display() {
	fmt.Printf("Ops version: %s\n", p.OpsVersion)
	fmt.Printf("Nanos version: %s\n", p.NanosVersion)
	fmt.Printf("Qemu version: %s\n", p.QemuVersion)
	fmt.Printf("Arch: %s\n", p.Arch)
	fmt.Printf("Virtualized: %t\n", p.Hypervisor)
}

// ProfileCommand provides a profile command
func ProfileCommand() *cobra.Command {
	var cmdProfile = &cobra.Command{
		Use:   "profile",
		Short: "Profile",
		Run:   printProfile,
	}
	return cmdProfile
}

func printProfile(cmd *cobra.Command, args []string) {
	p := Profile{}

	p.setProfile()

	p.display()
}
