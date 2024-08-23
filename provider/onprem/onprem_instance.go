package onprem

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/network"
	"github.com/nanovms/ops/qemu"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/sys/unix"
)

// CreateInstancePID creates an instance and returns the pid.
func (p *OnPrem) CreateInstancePID(ctx *lepton.Context) (string, error) {
	return p.createInstance(ctx)
}

// genMgmtPort should generate mgmt before launching instance and
// persist in instance metadata
func genMgmtPort() string {
	dd := rand.Int31n(10000) + 40000
	return strconv.Itoa(int(dd))
}

func (p *OnPrem) createInstance(ctx *lepton.Context) (string, error) {
	c := ctx.Config()

	imageName := c.CloudConfig.ImageName

	if _, err := os.Stat(path.Join(lepton.GetOpsHome(), "images", c.CloudConfig.ImageName)); os.IsNotExist(err) {
		return "", fmt.Errorf("image \"%s\" not found", imageName)
	}

	// hack - should figure out how to conjoin these 2 together
	if c.Mounts != nil {
		err := AddVirtfsShares(c)
		if err != nil {
			return "", err
		}
	}

	if c.Mounts != nil {
		c.VolumesDir = lepton.LocalVolumeDir
		err := AddMountsFromConfig(c)
		if err != nil {
			return "", err
		}
	}

	// linux local only; mac uses diff bridge
	if runtime.GOOS == "linux" && c.RunConfig.Bridged {
		tapDeviceName := c.RunConfig.TapName
		bridged := c.RunConfig.Bridged
		ipaddress := c.RunConfig.IPAddress
		netmask := c.RunConfig.NetMask
		bridgeipaddress := c.RunConfig.BridgeIPAddress

		bridgeName := c.RunConfig.BridgeName
		if bridged && bridgeName == "" {
			bridgeName = "br0"
		}

		networkService := network.NewIprouteNetworkService()

		if tapDeviceName != "" {
			err := network.SetupNetworkInterfaces(networkService, tapDeviceName, bridgeName, ipaddress, netmask, bridgeipaddress)
			if err != nil {
				return "", err
			}
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

	c.RunConfig.Mgmt = genMgmtPort()

	err := hypervisor.Start(&c.RunConfig)
	if err != nil {
		return "", err
	}

	pid, err := hypervisor.PID()
	if err != nil {
		return "", err
	}

	// if atexit is set it's actually executing through a parent shell
	// so get the child
	if c.RunConfig.AtExit != "" {
		pid = findPIDFromHook(pid)
	}

	instances := path.Join(opshome, "instances")

	arch := "arm64"
	if qemu.ArchCheck() {
		arch = "amd64"
	}

	i := instance{
		Instance: c.RunConfig.InstanceName,
		Image:    c.RunConfig.ImageName,
		Ports:    c.RunConfig.Ports,
		Pid:      pid,
		Mgmt:     c.RunConfig.Mgmt,
		Arch:     arch,
	}

	if c.RunConfig.Bridged {
		i.Bridged = true
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

func findPIDFromHook(ppid string) string {
	pid, err := execCmd("pgrep -P " + ppid)
	if err != nil {
		fmt.Println(err)
	}

	return strings.TrimSpace(pid)
}

// CreateInstance on premise
// assumes local
func (p *OnPrem) CreateInstance(ctx *lepton.Context) error {
	_, err := p.createInstance(ctx)
	return err
}

// GetMetaInstanceByName returns onprem metadata about a given named instance.
func (p *OnPrem) GetMetaInstanceByName(ctx *lepton.Context, name string) (*instance, error) {
	instances, err := p.GetMetaInstances(ctx)
	if err != nil {
		return nil, err
	}

	for _, i := range instances {
		if i.Instance == name {
			return &i, nil
		}
	}

	return nil, lepton.ErrInstanceNotFound(name)
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
	out, err := execCmd("ps ww -p " + pid) //fixme: cmd injection
	if err != nil {
		if 1 == 2 {
			fmt.Println(err)
		}
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
		if 1 == 2 {
			fmt.Println(err)
		}
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

	devspl := ""
	if runtime.GOOS == "darwin" {
		devspl = "netdev=vmnet,mac="
	} else {
		devspl = ",mac="
	}

	oo := strings.Split(out, devspl)
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

// GetMetaInstances returns instance data for onprem metadata found in
// ~/.ops/instances .
func (p *OnPrem) GetMetaInstances(ctx *lepton.Context) (instances []instance, err error) {
	opshome := lepton.GetOpsHome()
	instancesPath := path.Join(opshome, "instances")

	files, err := os.ReadDir(instancesPath)
	if err != nil {
		return
	}

	for _, f := range files {
		fullpath := path.Join(instancesPath, f.Name())

		// this is a cleanup helper
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

		instances = append(instances, i)
	}

	return instances, err
}

// GetInstances return all instances on prem
func (p *OnPrem) GetInstances(ctx *lepton.Context) (instances []lepton.CloudInstance, err error) {
	opshome := ""
	if ctx.Config().Home != "" {
		opshome = filepath.Join(ctx.Config().Home, ".ops")
	} else {
		opshome = lepton.GetOpsHome()
	}

	instancesPath := path.Join(opshome, "instances")

	files, err := os.ReadDir(instancesPath)
	if err != nil {
		return
	}

	// this logic is duped in GetMetaInstances - need to de-dupe.
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

		// relying on file ctime; prob should actually store the date in
		// the metadata file.
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
			Ports:      strings.Split(i.portList(), ","),
		})
	}

	return
}

func (p *OnPrem) getInstancesStats(ctx *lepton.Context, rinstances []lepton.CloudInstance) ([]lepton.CloudInstance, error) {
	for i := 0; i < len(rinstances); i++ {
		instance, err := p.GetMetaInstanceByName(ctx, rinstances[i].Name)
		if err != nil {
			fmt.Println(err)
		}

		last := instance.Mgmt

		devid := "2"
		if instance.Arch == "amd64" {
			devid = "3"
		}

		commands := []string{
			`{ "execute": "qmp_capabilities" }`,
			`{ "execute": "qom-set", "arguments": { "path": "/machine/peripheral-anon/device[` + devid + `]", "property": "guest-stats-polling-interval", "value": 2}}`,
			`{ "execute": "qom-get", "arguments": { "path": "/machine/peripheral-anon/device[` + devid + `]", "property": "guest-stats" } }`,
		}

		s := executeQMPLastRead(commands, last)

		var lr qmpResponse

		err = json.Unmarshal([]byte(s), &lr)
		if err != nil {
			// silently fail here as sometimes we get invalid values
			// (eg: at instance start)
			//
			//			fmt.Println(err)
		}

		rinstances[i].FreeMemory = (lr.qmpReturn.Stats.FreeMemory / int64(1000000))
		rinstances[i].TotalMemory = (lr.qmpReturn.Stats.TotalMemory / int64(1000000))
	}

	return rinstances, nil
}

// InstanceStats shows metrics for instance onprem .
func (p *OnPrem) InstanceStats(ctx *lepton.Context, iname string, watch bool) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	rinstances := []lepton.CloudInstance{}

	for i := 0; i < len(instances); i++ {
		if iname != "" {
			if iname != instances[i].Name {
				continue
			} else {
				rinstances = append(rinstances, instances[i])
			}
		} else {
			rinstances = append(rinstances, instances[i])
		}
	}

	// FIXME: if we're watching no need to establish a connection
	// everytime
	if watch {
		for {
			rinstances, err = p.getInstancesStats(ctx, rinstances)
			if err != nil {
				fmt.Println(err)
			}

			json.NewEncoder(os.Stdout).Encode(rinstances)
			time.Sleep(500 * time.Millisecond)
		}
	} else {
		rinstances, err = p.getInstancesStats(ctx, rinstances)
		if err != nil {
			fmt.Println(err)
		}

		if ctx.Config().RunConfig.JSON {
			if len(rinstances) == 0 {
				fmt.Println("[]")
				return nil
			}
			return json.NewEncoder(os.Stdout).Encode(rinstances)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"PID", "Name", "Memory"})
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgCyanColor})

		table.SetRowLine(true)

		for _, i := range rinstances {
			var rows []string

			rows = append(rows, i.ID)
			rows = append(rows, i.Name)
			rows = append(rows, i.HumanMem())

			table.Append(rows)
		}

		table.Render()

		return nil
	}
}

// ListInstances on premise
func (p *OnPrem) ListInstances(ctx *lepton.Context) error {
	instances, err := p.GetInstances(ctx)
	if err != nil {
		return err
	}

	// perhaps this could be a new type
	if ctx.Config().RunConfig.JSON {
		if len(instances) == 0 {
			fmt.Println("[]")
			return nil
		}
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
		rows = append(rows, strings.Join(i.Ports, ","))

		table.Append(rows)
	}

	table.Render()

	return nil

}

// StartInstance from on premise
// right now this assumes it was paused; not a boot; there's another
// call we can use here to get the status first
func (p *OnPrem) StartInstance(ctx *lepton.Context, instancename string) error {
	instance, err := p.GetMetaInstanceByName(ctx, instancename)
	if err != nil {
		return err
	}

	last := instance.Mgmt

	commands := []string{
		`{ "execute": "qmp_capabilities" }`,
		`{ "execute": "cont" }`,
	}

	executeQMP(commands, last)
	return nil
}

func executeQMP(commands []string, last string) {
	c, err := net.Dial("tcp", "localhost:"+last)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	for i := 0; i < len(commands); i++ {
		_, err := c.Write([]byte(commands[i] + "\n"))
		if err != nil {
			fmt.Println(err)
		}
		received := make([]byte, 1024)
		_, err = c.Read(received)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

type qmpResponse struct {
	qmpReturn `json:"return"`
}

type qmpReturn struct {
	Stats      qmpStats `json:"stats"`
	LastUpdate int64    `json:"last-update"`
}

type qmpStats struct {
	HtlbPGalloc     int64 `"json:stat-htlb-pgalloc"`
	SwapOut         int64 `json:"stat-swap-out"`
	AvailableMemory int64 `json:"stat-available-memory"`
	HtlbPgfail      int64 `json:"stat-htlb-pgfail"`
	FreeMemory      int64 `json:"stat-free-memory"`
	MinorFaults     int64 `json:"stat-minor-faults"`
	MajorFaults     int64 `json:"stat-major-faults"`
	TotalMemory     int64 `json:"stat-total-memory"`
	SwapIn          int64 `json:"stat-swap-in"`
	DiskCaches      int64 `json:"stat-disk-caches"`
}

// bit of a hack
func executeQMPLastRead(commands []string, last string) string {
	lo := ""

	c, err := net.Dial("tcp", "localhost:"+last)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	str, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		fmt.Println(str)
	}

	for i := 0; i < len(commands); i++ {
		_, err := c.Write([]byte(commands[i] + "\n"))
		if err != nil {
			fmt.Println(err)
		}
		str, err := bufio.NewReader(c).ReadString('\n')
		lo = str
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	return lo
}

// RebootInstance from on premise
func (p *OnPrem) RebootInstance(ctx *lepton.Context, instancename string) error {
	instance, err := p.GetMetaInstanceByName(ctx, instancename)
	if err != nil {
		return err
	}

	last := instance.Mgmt

	commands := []string{
		`{ "execute": "qmp_capabilities" }`,
		`{ "execute": "system_reset" }`,
	}

	executeQMP(commands, last)

	return nil
}

// StopInstance from on premise
func (p *OnPrem) StopInstance(ctx *lepton.Context, instancename string) error {
	instance, err := p.GetMetaInstanceByName(ctx, instancename)
	if err != nil {
		return err
	}

	last := instance.Mgmt

	commands := []string{
		`{ "execute": "qmp_capabilities" }`,
		`{ "execute": "stop" }`,
	}

	executeQMP(commands, last)
	return nil
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
	if watch {
		file, err := os.Open("/tmp/" + instancename + ".log")
		if err != nil {
			return err
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(500 * time.Millisecond)
					continue
				}

				break
			}
			fmt.Printf("%s", string(line))
		}
		return nil
	}

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
