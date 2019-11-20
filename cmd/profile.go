package cmd

import (
	"fmt"
	"io/ioutil"
	"path"
	"runtime"

	api "github.com/nanovms/ops/lepton"
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
}

func (p *Profile) save() {

	str := fmt.Sprintf("ops version:%s\nnanos version:%s\nqemu version:%s\narch:%s",
		p.OpsVersion, p.NanosVersion, p.QemuVersion, p.Arch)

	local := path.Join(api.GetOpsHome(), "profile")

	err := ioutil.WriteFile(local, []byte(str), 0644)
	if err != nil {
		fmt.Println(err)
	}
}

func (p *Profile) setProfile() {

	p.OpsVersion = api.Version
	p.NanosVersion = api.LocalReleaseVersion

	qv, err := api.QemuVersion()
	if err != nil {
		fmt.Println(err)
	}

	p.QemuVersion = qv

	p.Arch = runtime.GOOS

	p.save()

}

func (p *Profile) display() {
	fmt.Printf("Ops version: %s\n", p.OpsVersion)
	fmt.Printf("Nanos version: %s\n", p.NanosVersion)
	fmt.Printf("Qemu version: %s\n", p.QemuVersion)
	fmt.Printf("Arch: %s\n", p.Arch)
}

func printProfile(cmd *cobra.Command, args []string) {
	p := Profile{}

	p.setProfile()

	p.display()
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
