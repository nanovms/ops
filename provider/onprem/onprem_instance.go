package onprem

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/qemu"
	"github.com/olekukonko/tablewriter"

	"golang.org/x/sys/unix"
)

// CreateInstancePID creates an instance and returns the pid.
func (p *OnPrem) CreateInstancePID(ctx *lepton.Context) (string, error) {
	return p.createInstance(ctx)
}

func (p *OnPrem) createInstance(ctx *lepton.Context) (string, error) {
	c := ctx.Config()

	imageName := c.CloudConfig.ImageName

	if _, err := os.Stat(path.Join(lepton.GetOpsHome(), "images", c.CloudConfig.ImageName)); os.IsNotExist(err) {
		return "", fmt.Errorf("image \"%s\" not found", imageName)
	}

	if c.Mounts != nil {
		c.VolumesDir = lepton.LocalVolumeDir
		err := AddMountsFromConfig(c)
		if err != nil {
			return "", err
		}
	}

	hypervisor := qemu.HypervisorInstance()
	if hypervisor == nil {
		fmt.Println("No hypervisor found on $PATH")
		fmt.Println("Please install OPS using curl https://ops.city/get.sh -sSfL | sh")
		os.Exit(1)
	}

	if c.RunConfig.InstanceName == "" {
		c.RunConfig.InstanceName = strings.Split(c.CloudConfig.ImageName, ".")[0]
	}

	fmt.Printf("booting %s ...\n", c.RunConfig.InstanceName)

	opshome := lepton.GetOpsHome()
	imgpath := path.Join(opshome, "images", c.CloudConfig.ImageName)

	c.RunConfig.ImageName = imgpath
	c.RunConfig.Background = true

	err := hypervisor.Start(&c.RunConfig)
	if err != nil {
		return "", err
	}

	pid, err := hypervisor.PID()
	if err != nil {
		return "", err
	}

	instances := path.Join(opshome, "instances")

	i := instance{
		Instance: c.RunConfig.InstanceName,
		Image:    c.RunConfig.ImageName,
		Ports:    c.RunConfig.Ports,
	}

	if qemu.OPSD != "" {
		i.Bridged = true
	}

	d1, err := json.Marshal(i)
	if err != nil {
		log.Error(err)
	}

	err = os.WriteFile(instances+"/"+pid, d1, 0644)
	if err != nil {
		log.Error(err)
	}

	return pid, err
}

// CreateInstance on premise
// assumes local
func (p *OnPrem) CreateInstance(ctx *lepton.Context) error {
	_, err := p.createInstance(ctx)
	return err
}

// GetInstanceByName returns instance with given name
func (p *OnPrem) GetInstanceByName(ctx *lepton.Context, name string) (*lepton.CloudInstance, error) {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return nil, err
	}

	for _, i := range instances {
		if i.Name == name {
			return &i, nil
		}
	}

	return nil, lepton.ErrInstanceNotFound(name)
}

// FindBridgedIPByPID finds a qemu process with the pid and returns the
// ip.
func FindBridgedIPByPID(pid string) string {
	out, err := execCmd("ps -p " + pid) //fixme: cmd injection
	if err != nil {
		fmt.Println(err)
	}

	oo := strings.Split(out, "netdev=vmnet,mac=")
	mac := ""
	if len(oo) > 1 {
		ooz := strings.Split(oo[1], " ")
		mac = ooz[0]
	} else {
		fmt.Println("couldn't find mac")
	}

	return arpMac(pid, mac)
}

func arpMac(pid string, mac string) string {
	/// only use for resolution not for storage
	dmac, err := formatOctet(mac)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	// needs recent activity to register
	out, err := execCmd("arp -a | grep " + dmac)
	if err != nil {
		fmt.Println(err)
	}

	if strings.Contains(out, "(") {
		oo := strings.Split(out, "(")
		ooz := strings.Split(oo[1], ")")
		ip := ooz[0]

		logMac(pid, mac, ip)

		return ip
	}

	return ""
}

// FindBridgedIP returns the ip for an instance by forcing arp
// resolution. You should probably use the other function that takes a
// pid though.
//
// super hacky mac extraction, arp resolution; revisit in future
// could also potentially extract from logs || could have nanos ping
// upon boot
func FindBridgedIP(instanceID string) string {
	out, err := execCmd("ps aux | grep " + instanceID + " | grep -v grep") //fixme: cmd injection
	if err != nil {
		fmt.Println(err)
	}

	oo := strings.Split(out, "netdev=vmnet,mac=")
	mac := ""
	if len(oo) > 1 {
		ooz := strings.Split(oo[1], " ")
		mac = ooz[0]
	} else {
		fmt.Println("couldn't find mac")
	}

	// hack hack hack
	out, err = execCmd("ps aux  |grep -a " + instanceID + " | grep qemu | grep -v grep | awk {'print $2'}")
	if err != nil {
		fmt.Println(err)
	}
	pid := strings.TrimSpace(out)

	return arpMac(pid, mac)
}

// returns a mac with leading zeros dropped which is what mac does
// d6:3f:9b:0f:0c:c8
// d6:3f:9b:f:c:c8
func formatOctet(mac string) (string, error) {
	if mac == "" {
		return "", errors.New("mac is an empty string")
	}

	octets := strings.Split(mac, ":")
	newmac := ""
	for i := 0; i < len(octets); i++ {
		oi := strings.TrimLeft(octets[i], "0")
		newmac += oi + ":"
	}

	newmac = strings.TrimRight(newmac, ":")
	return newmac, nil
}

// log our mac,ip,pid so we don't have to lookup again
// FIXME: for bridged we can store in a more proper fashion
// we still want to support daemon-less as much as we can
//
// also should throw a lock on this at some point unless we migrate to
// something else that is a bit more industrial
func logMac(pid string, mac string, ip string) {

	opshome := lepton.GetOpsHome()
	instancesPath := path.Join(opshome, "instances")

	fullpath := path.Join(instancesPath, pid)

	body, err := os.ReadFile(fullpath)
	if err != nil {
		fmt.Println(err)
	}

	var i instance
	if err := json.Unmarshal(body, &i); err != nil {
		fmt.Println(err)
	}

	i.PrivateIP = ip
	i.Pid = pid
	i.Mac = mac

	d1, err := json.Marshal(i)
	if err != nil {
		fmt.Println(err)
	}

	err = os.WriteFile(instancesPath+"/"+pid, d1, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

func execCmd(cmdStr string) (output string, err error) {
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	output = string(out)
	return
}

// GetInstances return all instances on prem
func (p *OnPrem) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	opshome := lepton.GetOpsHome()
	instancesPath := path.Join(opshome, "instances")

	files, err := os.ReadDir(instancesPath)
	if err != nil {
		return
	}

	for _, f := range files {
		fullpath := path.Join(instancesPath, f.Name())

		pid, err := strconv.ParseInt(f.Name(), 10, 32)
		if err != nil {
			return nil, err
		}
		process, err := os.FindProcess(int(pid))
		if err != nil {
			return nil, err
		}
		if err = process.Signal(syscall.Signal(0)); err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "already finished") ||
				strings.Contains(errMsg, "already released") ||
				strings.Contains(errMsg, "not initialized") {
				os.Remove(fullpath)
				continue
			}
		}

		body, err := os.ReadFile(fullpath)
		if err != nil {
			return nil, err
		}

		var i instance
		if err := json.Unmarshal(body, &i); err != nil {
			return nil, err
		}

		pips := []string{}
		if i.Bridged {
			if i.PrivateIP == "" || i.Mac == "" {
				i.PrivateIP = FindBridgedIP(i.Instance)
			}

			pips = append(pips, i.PrivateIP)
		} else {
			pips = append(pips, "127.0.0.1")
		}

		file, err := os.Open(fullpath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		stat := &unix.Stat_t{}
		err = unix.Fstat(int(file.Fd()), stat)
		if err != nil {
			return nil, err
		}

		// rather un-helpful if we're just overwriting it though
		ctime := time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))

		// perhaps return proto'd version here instead then wrap
		// w/cloudinstance for cli
		instances = append(instances, lepton.CloudInstance{
			ID:         f.Name(), // pid
			Name:       i.Instance,
			Image:      i.Image,
			Status:     "Running",
			Created:    lepton.Time2Human(ctime),
			PrivateIps: pips,
			PublicIps:  strings.Split(i.portList(), ","),
		})
	}

	return
}

// ListInstances on premise
func (p *OnPrem) ListInstances(ctx *lepton.Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	if ctx.Config().RunConfig.JSON {
		return json.NewEncoder(os.Stdout).Encode(instances)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"PID", "Name", "Image", "Status", "Created", "Private Ips", "Port"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})

	table.SetRowLine(true)

	for _, i := range instances {
		var rows []string

		rows = append(rows, i.ID)
		rows = append(rows, i.Name)
		rows = append(rows, i.Image)
		rows = append(rows, i.Status)
		rows = append(rows, i.Created)
		rows = append(rows, strings.Join(i.PrivateIps, ","))
		rows = append(rows, strings.Join(i.PublicIps, ","))

		table.Append(rows)
	}

	table.Render()

	return nil

}

// StartInstance from on premise
func (p *OnPrem) StartInstance(ctx *lepton.Context, instancename string) error {
	return fmt.Errorf("operation not supported")
}

// StopInstance from on premise
func (p *OnPrem) StopInstance(ctx *lepton.Context, instancename string) error {
	return fmt.Errorf("operation not supported")
}

// DeleteInstance from on premise
func (p *OnPrem) DeleteInstance(ctx *lepton.Context, instancename string) error {

	var pid int
	instance, err := p.GetInstanceByName(ctx, instancename)
	if err != nil {
		return err
	}

	pid, _ = strconv.Atoi(instance.ID)

	if pid == 0 {
		fmt.Printf("did not find pid of instance \"%s\"\n", instancename)
		return nil
	}

	err = sysKill(pid)
	if err != nil {
		log.Error(err)
	}

	opshome := lepton.GetOpsHome()
	ipath := path.Join(opshome, "instances", strconv.Itoa(pid))
	err = os.Remove(ipath)
	if err != nil {
		return err
	}

	return nil
}

// PrintInstanceLogs writes instance logs to console
func (p *OnPrem) PrintInstanceLogs(ctx *lepton.Context, instancename string, watch bool) error {
	l, err := p.GetInstanceLogs(ctx, instancename)
	if err != nil {
		return err
	}
	fmt.Printf(l)
	return nil
}

// GetInstanceLogs for onprem instance logs
func (p *OnPrem) GetInstanceLogs(ctx *lepton.Context, instancename string) (string, error) {

	body, err := os.ReadFile("/tmp/" + instancename + ".log")
	if err != nil {
		log.Fatal(err)
	}

	return string(body), nil
}
