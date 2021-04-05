package hyperv

import (
	"strconv"
	"strings"
)

// GetVirtualMachines returns the list of virtual machines
func GetVirtualMachines() (string, error) {
	var script = `
	Get-VM | Select VMName,State,CreationTime
	`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script)

	return cmdOut, err
}

// GetVirtualMachineByName returns a virtual machine details
func GetVirtualMachineByName(vmName string) (string, error) {
	var script = `
	param([string]$vmName)
	Get-VM -Name $vmName | Select VMName,State,CreationTime
	`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, vmName)

	return cmdOut, err
}

// GetVirtualMachineNetworkAdapterAddress returns a virtual machine ip address
func GetVirtualMachineNetworkAdapterAddress(vmName string) (string, error) {

	var script = `
param([string]$vmName, [int]$addressIndex)
try {
  $adapter = Get-VMNetworkAdapter -VMName $vmName -ErrorAction SilentlyContinue
  $ip = $adapter.IPAddresses[$addressIndex]
  if($ip -eq $null) {
    return $false
  }
} catch {
  return $false
}
$ip
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, vmName, "0")

	return cmdOut, err
}

// CreateVirtualMachine creates a virtual machine
func CreateVirtualMachine(vmName string, path string, ram int64, diskSize int64, switchName string, generation uint) error {

	if generation == 2 {
		var script = `
param([string]$vmName, [string]$path, [long]$memoryStartupBytes, [long]$newVHDSizeBytes, [string]$switchName, [int]$generation)
$vhdx = $vmName + '.vhdx'
$vhdPath = Join-Path -Path $path -ChildPath $vhdx
New-VM -Name $vmName -Path $path -MemoryStartupBytes $memoryStartupBytes -NewVHDPath $vhdPath -NewVHDSizeBytes $newVHDSizeBytes -SwitchName $switchName -Generation $generation
`
		var ps PowerShellCmd
		err := ps.Run(script, vmName, path, strconv.FormatInt(ram, 10), strconv.FormatInt(diskSize, 10), switchName, strconv.FormatInt(int64(generation), 10))
		if err != nil {
			return err
		}

		return SetVMSecureBoot(vmName, false)
	}

	var script = `
param([string]$vmName, [string]$path, [long]$memoryStartupBytes, [long]$newVHDSizeBytes, [string]$switchName)
$vhdx = $vmName + '.vhdx'
$vhdPath = Join-Path -Path $path -ChildPath $vhdx
New-VM -Name $vmName -Path $path -MemoryStartupBytes $memoryStartupBytes -NewVHDPath $vhdPath -NewVHDSizeBytes $newVHDSizeBytes -SwitchName $switchName
`
	var ps PowerShellCmd
	err := ps.Run(script, vmName, path, strconv.FormatInt(ram, 10), strconv.FormatInt(diskSize, 10), switchName)

	if err != nil {
		return err
	}

	return deleteAllDvdDrives(vmName)

}

// CreateVirtualMachineWithNoHD creates a virtual machine without disks attached. The image disk is attached after using 'AddVirtualMachineHardDiskDrive'
func CreateVirtualMachineWithNoHD(vmName string, ram int64, generation uint) error {

	if generation == 2 {
		var script = `
param([string]$vmName, [long]$memoryStartupBytes, [int]$generation)
New-VM -Name $vmName -NoVHD -MemoryStartupBytes $memoryStartupBytes -Generation $generation
`
		var ps PowerShellCmd
		err := ps.Run(script, vmName, strconv.FormatInt(ram, 10), strconv.FormatInt(int64(generation), 10))
		if err != nil {
			return err
		}

		return SetVMSecureBoot(vmName, false)
	}

	var script = `
param([string]$vmName, [long]$memoryStartupBytes)
New-VM -Name $vmName -MemoryStartupBytes $memoryStartupBytes -NoVHD
`
	var ps PowerShellCmd
	err := ps.Run(script, vmName, strconv.FormatInt(ram, 10))

	if err != nil {
		return err
	}

	return deleteAllDvdDrives(vmName)

}

func deleteAllDvdDrives(vmName string) error {
	var script = `
param([string]$vmName)
Hyper-V\Get-VMDvdDrive -VMName $vmName | Hyper-V\Remove-VMDvdDrive
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}

// AddVirtualMachineHardDiskDrive adds virtual hard disk (vhdx) in path to virtual machine
// Due to issues on using vhdx files located inside WSL it copies the image to a user directory
// located at `~/vhdx-images` before attaching to the virtual machine.
// See more details on https://github.com/MicrosoftDocs/windows-powershell-docs/issues/2286
func AddVirtualMachineHardDiskDrive(vmName string, path string, hdController string, setBootDev bool) error {

	var script = `
param([string]$vmName, [string]$path, [string]$hdController)
New-Item -ItemType Directory -Force -Path ~/vhdx-images
$vhdx = $vmName + '.vhdx'
$newPath = Join-Path -Path ~/vhdx-images -ChildPath $vhdx
cp $path $newPath
$newPath = Convert-Path $newPath
Add-VMHardDiskDrive -VMName $vmName -Path $newPath -ControllerType $hdController -ControllerNumber 0
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName, path, hdController)
	if err != nil {
		return err
	}

	if setBootDev {
		script = `
param([string]$vmName, [string]$hdController)
$hdDrive = Get-VMHardDiskDrive -VMName $vmName -ControllerType $hdController -ControllerNumber 0
Set-VMFirmware -VMName $vmName -FirstBootDevice $hdDrive
`
		err = ps.Run(script, vmName, hdController)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetVMSecureBoot enables or disables secure boot on a VM
func SetVMSecureBoot(vmName string, enable bool) error {
	var onOff string
	if enable {
		onOff = "On"
	} else {
		onOff = "Off"
	}
	var script = `
param([string]$vmName, [string]$onOff)
Set-VMFirmware -VMName $vmName -EnableSecureBoot $onOff
`
	var ps PowerShellCmd
	err := ps.Run(script, vmName, onOff)
	return err
}

// SetVMComPort sets com port with a named pipe with the same name of the vm
func SetVMComPort(vmName string) error {

	var script = `
	param([string]$vmName)
	Set-VMComPort $vmName 1 \\.\pipe\$vmName
	`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}

// DeleteVirtualMachine deletes a virtual machine
func DeleteVirtualMachine(vmName string) error {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName
if (($vm.State -ne 'Off') -and ($vm.State -ne 'OffCritical')) {
    Stop-VM -VM $vm -TurnOff -Force -Confirm:$false
}
Remove-VM -Name $vmName -Force -Confirm:$false
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}

// CreateVirtualSwitch creates a virtual switch
func CreateVirtualSwitch(switchName string, switchType string) (bool, error) {

	var script = `
param([string]$switchName,[string]$switchType)
$switches = Get-VMSwitch -Name $switchName -ErrorAction SilentlyContinue
if ($switches.Count -eq 0) {
  New-VMSwitch -Name $switchName -SwitchType $switchType
  return $true
}
return $false
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, switchName, switchType)
	var created = strings.TrimSpace(cmdOut) == "True"
	return created, err
}

// DeleteVirtualSwitch deletes a virtual switch
func DeleteVirtualSwitch(switchName string) error {

	var script = `
param([string]$switchName)
$switch = Get-VMSwitch -Name $switchName -ErrorAction SilentlyContinue
if ($switch -ne $null) {
    $switch | Remove-VMSwitch -Force -Confirm:$false
}
`

	var ps PowerShellCmd
	err := ps.Run(script, switchName)
	return err
}

// StartVirtualMachine starts a virtual machine
func StartVirtualMachine(vmName string) error {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName -ErrorAction SilentlyContinue
if ($vm.State -eq 'Off') {
  Start-VM -Name $vmName -Confirm:$false
}
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}

// RestartVirtualMachine restarts a virtual machine
func RestartVirtualMachine(vmName string) error {

	var script = `
param([string]$vmName)
Restart-VM $vmName -Force -Confirm:$false
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}

// StopVirtualMachine stops a virtual machine
func StopVirtualMachine(vmName string) error {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName
if ($vm.State -eq 'Running') {
    Stop-VM -VM $vm -Force -Confirm:$false
}
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}

// GetExternalOnlineVirtualSwitch returns an online external virtual switch name
func GetExternalOnlineVirtualSwitch() (string, error) {

	var script = `
$adapters = Get-NetAdapter -Physical -ErrorAction SilentlyContinue | Where-Object { $_.Status -eq 'Up' } | Sort-Object -Descending -Property Speed
foreach ($adapter in $adapters) {
  $switch = Get-VMSwitch -SwitchType External | Where-Object { $_.NetAdapterInterfaceDescription -eq $adapter.InterfaceDescription }
  if ($switch -ne $null) {
    $switch.Name
    break
  }
}
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script)
	if err != nil {
		return "", err
	}

	var switchName = strings.TrimSpace(cmdOut)
	return switchName, nil
}

// CreateExternalVirtualSwitch creates an external virtual switch connected to a physical network adapter
func CreateExternalVirtualSwitch(switchName string) error {

	var script = `
param([string]$switchName)
$switch = $null
$adapters = Get-NetAdapter -Physical -ErrorAction SilentlyContinue | where status -eq 'Up'
foreach ($adapter in $adapters) {
  $switch = Get-VMSwitch -SwitchType External | where { $_.NetAdapterInterfaceDescription -eq $adapter.InterfaceDescription }
  if ($switch -eq $null) {
    $switch = New-VMSwitch -Name $switchName -NetAdapterName $adapter.Name -AllowManagementOS $true -Notes 'Parent OS, VMs, WiFi'
  }
  if ($switch -ne $null) {
    break
  }
}
if($switch -eq $null) {
  Write-Error 'No internet adapters found'
}
`
	var ps PowerShellCmd
	err := ps.Run(script, switchName)
	return err
}

// GetVirtualMachineSwitchName returns the name of the switch connected to a virtual machine
func GetVirtualMachineSwitchName(vmName string) (string, error) {

	var script = `
param([string]$vmName)
(Get-VMNetworkAdapter -VMName $vmName).SwitchName
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, vmName)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(cmdOut), nil
}

// AddVirtualMachineNetworkAdapter creates a virtual machine network adapter that connects to a switch
func AddVirtualMachineNetworkAdapter(vmName string, vmSwitch string) error {
	var script = `
param([string]$vmName, [string]$vmSwitch)
Add-VMNetworkAdapter -VMName $vmName -SwitchName $vmSwitch
	`

	var ps PowerShellCmd
	err := ps.Run(script, vmName, vmSwitch)
	if err != nil {
		return err
	}

	return nil
}

// ConnectVirtualMachineNetworkAdapterToSwitch connects current virtual machine network adapter to switch
func ConnectVirtualMachineNetworkAdapterToSwitch(vmName string, switchName string) error {

	var script = `
param([string]$vmName,[string]$switchName)
Get-VMNetworkAdapter -VMName $vmName | Connect-VMNetworkAdapter -SwitchName $switchName
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName, switchName)
	return err
}

// IsRunning checks whether a virtual machine is running
func IsRunning(vmName string) (bool, error) {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName -ErrorAction SilentlyContinue
$vm.State -eq [Microsoft.HyperV.VMState]::Running
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, vmName)

	if err != nil {
		return false, err
	}

	var isRunning = strings.TrimSpace(cmdOut) == "True"
	return isRunning, err
}

// IsOff checks whether a virtual machine is off
func IsOff(vmName string) (bool, error) {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName -ErrorAction SilentlyContinue
$vm.State -eq [Microsoft.HyperV.VMState]::Off
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, vmName)

	if err != nil {
		return false, err
	}

	var isRunning = strings.TrimSpace(cmdOut) == "True"
	return isRunning, err
}

// Uptime gets virtual machine uptime
func Uptime(vmName string) (uint64, error) {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName -ErrorAction SilentlyContinue
$vm.Uptime.TotalSeconds
`
	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, vmName)

	if err != nil {
		return 0, err
	}

	uptime, err := strconv.ParseUint(strings.TrimSpace(string(cmdOut)), 10, 64)

	return uptime, err
}

// Mac returns mac address of virtual machine network adapter
func Mac(vmName string) (string, error) {
	var script = `
param([string]$vmName, [int]$adapterIndex)
try {
  $adapter = Get-VMNetworkAdapter -VMName $vmName -ErrorAction SilentlyContinue
  $mac = $adapter[$adapterIndex].MacAddress
  if($mac -eq $null) {
    return ""
  }
} catch {
  return ""
}
$mac
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, vmName, "0")

	return cmdOut, err
}

// IPAddress gets virtual machine ip addresses
func IPAddress(mac string) (string, error) {
	var script = `
param([string]$mac, [int]$addressIndex)
try {
  $ip = Get-Vm | %{$_.NetworkAdapters} | ?{$_.MacAddress -eq $mac} | %{$_.IpAddresses[$addressIndex]}

  if($ip -eq $null) {
    return ""
  }
} catch {
  return ""
}
$ip
`

	var ps PowerShellCmd
	cmdOut, err := ps.Output(script, mac, "0")

	return cmdOut, err
}

// TurnOff turns virtual machine off
func TurnOff(vmName string) error {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName -ErrorAction SilentlyContinue
if ($vm.State -eq [Microsoft.HyperV.VMState]::Running) {
  Stop-VM -Name $vmName -TurnOff -Force -Confirm:$false
}
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}

// ShutDown forces virtual machine to stop
func ShutDown(vmName string) error {

	var script = `
param([string]$vmName)
$vm = Get-VM -Name $vmName -ErrorAction SilentlyContinue
if ($vm.State -eq [Microsoft.HyperV.VMState]::Running) {
  Stop-VM -Name $vmName -Force -Confirm:$false
}
`

	var ps PowerShellCmd
	err := ps.Run(script, vmName)
	return err
}
