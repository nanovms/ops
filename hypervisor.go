package main
import ( 
        "fmt"
        "os"
        "os/exec"
    )

// Hypervisor interface
type Hypervisor interface {
    start(string) error
}

type drive struct {
    path   string
    format string
}

type netdev struct {
    nettype string
    id string
    ifname string
    mac string
}

type qemu struct {
    drives []drive
    ifaces []netdev
}

func (q *qemu) addDrive(path string, format string) {

}

func (q *qemu) addNetworkDevice(n netdev) {

}

func (q *qemu) start(image string) error {	 
    cmd := exec.Command("qemu-system-x86_64", q.Args()...)
    cmd.Stdout = os.Stdout
    cmd.Stdin = os.Stdin
    cmd.Stderr = os.Stderr

     if err := cmd.Run(); err != nil {
        fmt.Println(err)
        return err
     }
     return nil
}

func (q *qemu) Args() []string {
    // TODO : this should come from q.drives and ifaces
    args := []string{}

    boot := []string{"-boot", "c", "-drive", "file=image,format=raw,if=ide"}
    storage := []string {"-drive", "file=image2,format=raw,if=virtio"}
    net := []string{"-device", "virtio-net,mac=7e:b8:7e:87:4a:ea,netdev=n0", "-netdev", "tap,id=n0,ifname=tap0"}

    args = append(args, boot...)
    args = append(args, []string {"-nographic", "-m", "2G", "-device", "isa-debug-exit"}...)
    args = append(args, storage...)
    args = append(args, net...)
    args = append(args, "-enable-kvm")
    return args
}

func newQemu() Hypervisor {
    return 	&qemu{}
}

// available hypervisors
var hypervisors = map[string]func() Hypervisor {
        "qemu-system-x86_64" : newQemu,
    } 
