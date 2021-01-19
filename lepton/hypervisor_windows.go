package lepton

import "os/exec"

func checkExists(key string) bool {
	_, err := exec.LookPath(key)
	if err != nil {
		return false
	}
	return true
}

// HypervisorInstance provides available hypervisor
func HypervisorInstance() Hypervisor {
	for k := range hypervisors {
		if checkExists(k) {
			hypervisor := hypervisors[k]()
			return hypervisor
		}
	}
	return nil
}

// Hypervisor interface
type Hypervisor interface {
	Start(rconfig *RunConfig) error
	Command(rconfig *RunConfig) *exec.Cmd
	Stop()
}

// available hypervisors
var hypervisors = map[string]func() Hypervisor{}
