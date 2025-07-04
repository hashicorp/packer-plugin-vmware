// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

const (
	// VMware Fusion.
	fusionProductName = "VMware Fusion"
	fusionMinVersion  = "13.5.0"

	// VMware Workstation.
	workstationProductName      = "VMware Workstation"
	workstationMinVersion       = "17.5.0"
	workstationNoLicenseVersion = "17.6.2"
	// Notes:
	// Version 17.6.1 required a license key for commercial use but not for personal use.
	// Reference: dub.sh/vmw-ws-personal-use
	// Version 17.6.2 removed the license key requirement for commercial, educational, and personal use.
	// References: dub.sh/vmw-ws-free, dub.sh/vmw-ws-176-rn
	workstationInstallationPathKey = "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\App Paths\\vmware.exe"
	workstationDhcpRegistryKey     = "SYSTEM\\CurrentControlSet\\services\\VMnetDHCP\\Parameters"

	// VMware Workstation Player.
	playerProductName         = "VMware Workstation Player"
	playerMinVersion          = "17.5.0"
	playerInstallationPathKey = "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\App Paths\\vmplayer.exe"
	playerDhcpRegistryKey     = "SYSTEM\\CurrentControlSet\\services\\VMnetDHCP\\Parameters"

	// Linux Paths.
	linuxDefaultPath = "/etc/vmware/"
	linuxAppPath     = "/usr/lib/vmware/bin/"
	linuxIsosPath    = "/usr/lib/vmware/isoimages/"

	// OVF Tool.
	ovfToolDownloadURL = "https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest"
	ovfToolMinVersion  = "4.6.0"

	// Architectures.
	archAMD64 = "x86_x64"
	archARM64 = "arm64"

	// Operating systems.
	osWindows = "windows"
	osLinux   = "linux"
	osMacOS   = "darwin"

	// Clone types.
	cloneTypeLinked = "linked"
	cloneTypeFull   = "full"

	// GUI arguments.
	guiArgumentNoGUI = "nogui"
	guiArgumentGUI   = "gui"

	// Application binary names.
	appOvfTool      = "ovftool"
	appPlayer       = "vmplayer"
	appVdiskManager = "vmware-vdiskmanager"
	appVmrun        = "vmrun"
	appVmware       = "vmware"
	appVmx          = "vmware-vmx"
	appQemuImg      = "qemu-img"

	// Version regular expressions.
	productVersionRegex   = `(?i)VMware [a-z0-9-]+ (\d+\.\d+\.\d+)`
	technicalPreviewRegex = `(?i)VMware [a-z0-9-]+ e\.x\.p `
	ovfToolVersionRegex   = `\d+\.\d+\.\d+`

	// File names.
	dhcpVmnetConfFile   = "vmnetdhcp.conf"
	dhcpVmnetLeasesFile = "vmnetdhcp.leases"
	natVmnetConfFile    = "vmnetnat.conf"
	netmapConfFile      = "netmap.conf"
)

// Versions for supported or required components.
var (
	fusionMinVersionObj      = version.Must(version.NewVersion(fusionMinVersion))
	workstationMinVersionObj = version.Must(version.NewVersion(workstationMinVersion))
	playerMinVersionObj      = version.Must(version.NewVersion(playerMinVersion))
	ovfToolMinVersionObj     = version.Must(version.NewVersion(ovfToolMinVersion))
)

// The possible paths to the DHCP leases file.
var dhcpLeasesPaths = []string{
	"dhcp/dhcp.leases",
	"dhcp/dhcpd.leases",
	"dhcpd/dhcp.leases",
	"dhcpd/dhcpd.leases",
}

// The possible paths to the DHCP configuration file.
var dhcpConfPaths = []string{
	"dhcp/dhcp.conf",
	"dhcp/dhcpd.conf",
	"dhcpd/dhcp.conf",
	"dhcpd/dhcpd.conf",
}

// The product version.
var productVersion = regexp.MustCompile(productVersionRegex)

// The technical preview version.
var technicalPreview = regexp.MustCompile(technicalPreviewRegex)

// The VMware OVF Tool version.
var ovfToolVersion = regexp.MustCompile(ovfToolVersionRegex)

// Driver represents an interface for managing virtual machines.
type Driver interface {

	// Clone duplicates the source virtual machine to the destination, using the specified clone type and snapshot.
	Clone(dst string, src string, cloneType bool, snapshot string) error

	// CompactDisk compacts the specified virtual disk to reclaim unused space on the virtual machine.
	CompactDisk(string) error

	// CreateDisk creates a virtual disk with specified path, size, adapter type, and disk type.
	CreateDisk(string, string, string, string) error

	// CreateSnapshot creates a snapshot of the virtual machine specified by its path and assigns it the given snapshot
	// name.
	CreateSnapshot(string, string) error

	// IsRunning checks if the specified virtual machine is currently running.
	IsRunning(string) (bool, error)

	// Start powers on a virtual machine with the specified path and a boolean indicating whether it starts in headless
	// mode.
	Start(string, bool) error

	// Stop gracefully or forcibly halts the virtual machine identified by the provided path. Returns an error if it fails.
	Stop(string) error

	// SuppressMessages modifies the .vmx or surrounding directory to suppress messages.
	SuppressMessages(string) error

	// ToolsIsoPath returns the path to the VMware Tools ISO based on the specified flavor.
	ToolsIsoPath(string) string

	// ToolsInstall installs VMware Tools on the guest OS by mounting the Tools ISO via the driver.
	ToolsInstall() error

	// Verify checks the configuration and ensures that the driver is ready for use.
	Verify() error

	// CommHost determines and returns the host address for communication with the virtual machine from the provided
	// state.
	CommHost(multistep.StateBag) (string, error)

	// GetVmwareDriver retrieves an instance of VmwareDriver, a base class containing default methods for virtual
	// machine management.
	GetVmwareDriver() VmwareDriver

	// GuestAddress retrieves the MAC address of a guest machine from its VMX configuration using the provided state.
	GuestAddress(multistep.StateBag) (string, error)

	// PotentialGuestIP retrieves a list of potential IP addresses for the guest from the provided state.
	PotentialGuestIP(multistep.StateBag) ([]string, error)

	// HostAddress retrieves the host's network address based on the state.
	HostAddress(multistep.StateBag) (string, error)

	// HostIP retrieves the host IP address for the virtual machine based on the state.
	HostIP(multistep.StateBag) (string, error)

	// Export exports a virtual machine using the provided arguments.
	Export([]string) error

	// VerifyOvfTool validates the presence and compatibility of the OVF Tool based on specified conditions.
	VerifyOvfTool(bool, bool) error
}

// NewDriver initializes a suitable virtual machine driver based on the given configuration and host environment.
func NewDriver(dconfig *DriverConfig, config *SSHConfig, vmName string) (Driver, error) {
	var drivers []Driver

	if dconfig.RemoteType != "" {
		esxiDriver, err := NewEsxiDriver(dconfig, config, vmName)
		if err != nil {
			return nil, err
		}
		drivers = []Driver{esxiDriver}

	} else {
		switch runtime.GOOS {
		case "darwin":
			drivers = []Driver{
				NewFusionDriver(dconfig, config),
			}
		case "linux":
			fallthrough
		case "windows":
			drivers = []Driver{
				NewWorkstationDriver(config),
				NewPlayerDriver(config),
			}
		default:
			return nil, fmt.Errorf("error finding a driver for %s", runtime.GOOS)
		}
	}

	errs := ""
	for _, driver := range drivers {
		err := driver.Verify()

		log.Printf("Using driver %T, Success: %t", driver, err == nil)
		if err == nil {
			return driver, nil
		}

		log.Printf("Skipping %T because it failed with the following error %s", driver, err)
		errs += "* " + err.Error() + "\n"
	}

	return nil, fmt.Errorf("driver initialization failed. fix at least one driver to continue:\n%s", errs)
}

// runAndLog executes the given command, logs its execution, and returns its stdout, stderr, and any encountered error.
func runAndLog(cmd *exec.Cmd) (string, string, error) {
	var stdout, stderr bytes.Buffer

	log.Printf("[INFO] Running: %s %s", cmd.Path, strings.Join(cmd.Args[1:], " "))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	stdoutString := strings.TrimSpace(stdout.String())
	stderrString := strings.TrimSpace(stderr.String())

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		message := stderrString
		if message == "" {
			message = stdoutString
		}

		err = fmt.Errorf("error: %s", message)

		// If "unknown error" is in there, add some additional notes
		re := regexp.MustCompile(`(?i)unknown error`)
		if re.MatchString(message) {
			err = fmt.Errorf(
				"%s\n\n%s", err,
				"Packer detected an error from the VMware hypervisor "+
					"platform. Unfortunately, the error message provided is "+
					"not very specific. Please check the `vmware.log` files "+
					"created by the hypervisor platform when a virtual "+
					"machine is started. The logs are located in the "+
					"directory of the .vmx file and often contain more "+
					"detailed error information.\n\nYou may need to set the "+
					"command line flag --on-error=abort to prevent the plugin "+
					"from cleaning up the file directory.")
		}
	}

	log.Printf("stdout: %s", stdoutString)
	log.Printf("stderr: %s", stderrString)

	// Replace these for Windows, we only want to deal with Unix
	// style line endings.
	returnStdout := strings.ReplaceAll(stdout.String(), "\r\n", "\n")
	returnStderr := strings.ReplaceAll(stderr.String(), "\r\n", "\n")

	return returnStdout, returnStderr, err
}

// compareVersionObjects compares two version objects and ensures the found version meets or exceeds the required version.
func compareVersionObjects(versionFound, versionRequired *version.Version, product string) error {
	if versionFound.LessThan(versionRequired) {
		return fmt.Errorf("[ERROR] Requires %s %s or later; %s installed", product, versionRequired.String(), versionFound.String())
	}
	return nil
}

// ReadNetmapConfig reads a network map configuration file from the specified path and parses it.
func ReadNetmapConfig(path string) (NetworkMap, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return ReadNetworkMap(fd)
}

// ReadDhcpConfig reads a DHCP configuration file from the specified path and returns the parsed configuration.
func ReadDhcpConfig(path string) (DhcpConfiguration, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return ReadDhcpConfiguration(fd)
}

// readVMXConfig reads a .vmx configuration file located at the given path and returns its key-value pairs as a map.
func readVMXConfig(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return map[string]string{}, err
	}
	defer f.Close()

	vmxBytes, err := io.ReadAll(f)
	if err != nil {
		return map[string]string{}, err
	}
	return ParseVMX(string(vmxBytes)), nil
}

// readCustomDeviceName retrieves the custom network device name from the .vmx configuration.
func readCustomDeviceName(vmxData map[string]string) (string, error) {
	connectionType, ok := vmxData["ethernet0.connectiontype"]
	if !ok || connectionType != "custom" {
		return "", fmt.Errorf("unable to determine the device name for the connection type : %s", connectionType)
	}

	device, ok := vmxData["ethernet0.vnet"]
	if !ok || device == "" {
		return "", fmt.Errorf("unable to determine the device name for the connection type \"%s\" : %s", connectionType, device)
	}
	return device, nil
}

// VmwareDriver is a struct that provides methods and paths needed for virtual machine management.
type VmwareDriver struct {
	// These methods define paths that are utilized by the driver
	// A driver must overload these in order to point to the correct
	// files so that the address detection (ip and ethernet) machinery
	// works.
	DhcpLeasesPath   func(string) string
	DhcpConfPath     func(string) string
	VmnetnatConfPath func(string) string

	// This method returns an object with the NetworkNameMapper interface
	// that maps network to device and vice-versa.
	NetworkMapper func() (NetworkNameMapper, error)
}

// GuestAddress retrieves the MAC address of a guest virtual machine from the .vmx configuration.
func (d *VmwareDriver) GuestAddress(state multistep.StateBag) (string, error) {
	vmxPath := state.Get("vmx_path").(string)

	log.Println("Lookup up IP information...")
	vmxData, err := readVMXConfig(vmxPath)
	if err != nil {
		return "", err
	}

	var ok bool
	macAddress := ""
	if macAddress, ok = vmxData["ethernet0.address"]; !ok || macAddress == "" {
		if macAddress, ok = vmxData["ethernet0.generatedaddress"]; !ok || macAddress == "" {
			return "", errors.New("unable to determine MAC address")
		}
	}
	log.Printf("[INFO] GuestAddress discovered MAC address: %s", macAddress)

	res, err := net.ParseMAC(macAddress)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

// PotentialGuestIP identifies potential guest IP addresses for a virtual machine using DHCP leases and MAC address.
func (d *VmwareDriver) PotentialGuestIP(state multistep.StateBag) ([]string, error) {

	// grab network mapper
	netmap, err := d.NetworkMapper()
	if err != nil {
		return []string{}, err
	}

	// convert the stashed network to a device
	network := state.Get("vmnetwork").(string)
	devices, err := netmap.NameIntoDevices(network)

	// log them to see what was detected
	for _, device := range devices {
		log.Printf("[INFO] GuestIP discovered device matching %s: %s", network, device)
	}

	// we were unable to find the device, maybe it's a custom one...
	// so, check to see if it's in the .vmx configuration
	if err != nil || network == "custom" {
		vmxPath := state.Get("vmx_path").(string)
		vmxData, err := readVMXConfig(vmxPath)
		if err != nil {
			return []string{}, err
		}

		var device string
		device, err = readCustomDeviceName(vmxData)
		devices = append(devices, device)
		if err != nil {
			return []string{}, err
		}
		log.Printf("[INFO] GuestIP discovered custom device matching %s: %s", network, device)
	}

	// figure out our MAC address for looking up the guest address
	MACAddress, err := d.GuestAddress(state)
	if err != nil {
		return []string{}, err
	}

	// iterate through all of the devices and collect all the dhcp lease entries
	// that we possibly can.
	var availableLeaseEntries []dhcpLeaseEntry

	for _, device := range devices {
		// figure out the correct dhcp leases
		dhcpLeasesPath := d.DhcpLeasesPath(device)
		log.Printf("[INFO] Trying DHCP leases path: %s", dhcpLeasesPath)
		if dhcpLeasesPath == "" {
			return []string{}, fmt.Errorf("no DHCP leases path found for device %s", device)
		}

		// open up the path to the dhcpd leases
		fh, err := os.Open(dhcpLeasesPath)
		if err != nil {
			log.Printf("Error reading DHCP lease path file %s: %s", dhcpLeasesPath, err.Error())
			continue
		}
		defer fh.Close()

		// and then read its contents
		leaseEntries, err := ReadDhcpdLeaseEntries(fh)
		if err != nil {
			return []string{}, err
		}

		// Parse our MAC address again. There's no need to check for an
		// error because we've already parsed this successfully.
		hwaddr, _ := net.ParseMAC(MACAddress)

		// Go through our available lease entries and see which ones are within
		// scope, and that match to our hardware address.
		results := make([]dhcpLeaseEntry, 0)
		for _, entry := range leaseEntries {

			// First check for leases that are still valid. The timestamp for
			// each lease should be in UTC according to the documentation at
			// the top of VMware's dhcpd.leases file.
			now := time.Now().UTC()
			if !now.After(entry.starts) || !now.Before(entry.ends) {
				continue
			}

			// Next check for any where the hardware address matches.
			if !bytes.Equal(hwaddr, entry.ether) {
				continue
			}

			// This entry fits within our constraints, so store it so we can
			// check it out later.
			results = append(results, entry)
		}

		// If we weren't able to grab any results, then we'll do a "loose"-match
		// where we only look for anything where the hardware address matches.
		if len(results) == 0 {
			log.Printf("Unable to find an exact match for DHCP lease. Falling back loose matching for a hardware address %v", MACAddress)
			for _, entry := range leaseEntries {
				if bytes.Equal(hwaddr, entry.ether) {
					results = append(results, entry)
				}
			}
		}

		// If we found something, then we need to add it to our current list
		// of lease entries.
		if len(results) > 0 {
			availableLeaseEntries = append(availableLeaseEntries, results...)
		}

		// Now we need to map our results to get the address so we can return it.iterate through our results and figure out which one
		// is actually up...and should be relevant.
	}

	// Check if we found any lease entries that correspond to us. If so, then we
	// need to map() them in order to extract the address field to return to the
	// caller.
	if len(availableLeaseEntries) > 0 {
		addrs := make([]string, 0)
		for _, entry := range availableLeaseEntries {
			addrs = append(addrs, entry.address)
		}
		return addrs, nil
	}

	if runtime.GOOS == "darwin" {
		// We have match no vmware DHCP lease for this MAC. We'll try to match it in Apple DHCP leases.
		// As a remember, VMware is no longer able to rely on its own dhcpd server on MacOS BigSur and is
		// forced to use Apple DHCPD server instead.

		// set the apple dhcp leases path
		appleDhcpLeasesPath := "/var/db/dhcpd_leases"
		log.Printf("[INFO] Trying Apple DHCP leases path: %s", appleDhcpLeasesPath)

		// open up the path to the apple dhcpd leases
		fh, err := os.Open(appleDhcpLeasesPath)
		if err != nil {
			log.Printf("Error while reading apple DHCP lease path file %s: %s", appleDhcpLeasesPath, err.Error())
		} else {
			defer fh.Close()

			// and then read its contents
			leaseEntries, err := ReadAppleDhcpdLeaseEntries(fh)
			if err != nil {
				return []string{}, err
			}

			// Parse our MAC address again. There's no need to check for an
			// error because we've already parsed this successfully.
			hwaddr, _ := net.ParseMAC(MACAddress)

			// Go through our available lease entries and see which ones are within
			// scope, and that match to our hardware address.
			availableLeaseEntries := make([]appleDhcpLeaseEntry, 0)
			for _, entry := range leaseEntries {
				// Next check for any where the hardware address matches.
				if bytes.Equal(hwaddr, entry.hwAddress) {
					availableLeaseEntries = append(availableLeaseEntries, entry)
				}
			}

			// Check if we found any lease entries that correspond to us. If so, then we
			// need to map() them in order to extract the address field to return to the
			// caller.
			if len(availableLeaseEntries) > 0 {
				addrs := make([]string, 0)
				for _, entry := range availableLeaseEntries {
					addrs = append(addrs, entry.ipAddress)
				}
				return addrs, nil
			}
		}
	}

	return []string{}, fmt.Errorf("none of the found device(s) %v have a DHCP lease for MAC address %s", devices, MACAddress)
}

// HostAddress retrieves the host's hardware address linked to the network device specified in the state.
func (d *VmwareDriver) HostAddress(state multistep.StateBag) (string, error) {

	// grab mapper for converting network<->device
	netmap, err := d.NetworkMapper()
	if err != nil {
		return "", err
	}

	// convert network to name
	network := state.Get("vmnetwork").(string)
	devices, err := netmap.NameIntoDevices(network)

	// log them to see what was detected
	for _, device := range devices {
		log.Printf("[INFO] HostAddress discovered device matching %s: %s", network, device)
	}

	// we were unable to find the device, maybe it's a custom one...
	// so, check to see if it's in the .vmx configuration
	if err != nil || network == "custom" {
		vmxPath := state.Get("vmx_path").(string)
		vmxData, err := readVMXConfig(vmxPath)
		if err != nil {
			return "", err
		}

		var device string
		device, err = readCustomDeviceName(vmxData)
		devices = append(devices, device)
		if err != nil {
			return "", err
		}
		log.Printf("[INFO] HostAddress discovered custom device matching %s: %s", network, device)
	}

	var lastError error
	for _, device := range devices {
		// parse dhcpd configuration
		pathDhcpConfig := d.DhcpConfPath(device)
		if _, err := os.Stat(pathDhcpConfig); err != nil {
			return "", fmt.Errorf("unable to find vmnetdhcp conf file: %s", pathDhcpConfig)
		}

		config, err := ReadDhcpConfig(pathDhcpConfig)
		if err != nil {
			lastError = err
			continue
		}

		// find the entry configured in the dhcpd
		interfaceConfig, err := config.HostByName(device)
		if err != nil {
			lastError = err
			continue
		}

		// finally grab the hardware address
		address, err := interfaceConfig.Hardware()
		if err == nil {
			return address.String(), nil
		}

		// we didn't find it, so search through our interfaces for the device name
		interfaceList, err := net.Interfaces()
		if err == nil {
			return "", err
		}

		names := make([]string, 0)
		for _, intf := range interfaceList {
			if strings.HasSuffix(strings.ToLower(intf.Name), device) {
				return intf.HardwareAddr.String(), nil
			}
			//lint:ignore SA4010 result of append is not used here
			names = append(names, intf.Name)
		}
	}
	return "", fmt.Errorf("unable to find host address from devices %v, last error: %s", devices, lastError)
}

// HostIP retrieves the host machine's IP address associated with the specific network device defined in the state.
func (d *VmwareDriver) HostIP(state multistep.StateBag) (string, error) {

	// grab mapper for converting network<->device
	netmap, err := d.NetworkMapper()
	if err != nil {
		return "", err
	}

	// convert network to name
	network := state.Get("vmnetwork").(string)
	devices, err := netmap.NameIntoDevices(network)

	// log them to see what was detected
	for _, device := range devices {
		log.Printf("[INFO] HostIP discovered device matching %s: %s", network, device)
	}

	// we were unable to find the device, maybe it's a custom one...
	// so, check to see if it's in the .vmx configuration
	if err != nil || network == "custom" {
		vmxPath := state.Get("vmx_path").(string)
		vmxData, err := readVMXConfig(vmxPath)
		if err != nil {
			return "", err
		}

		var device string
		device, err = readCustomDeviceName(vmxData)
		devices = append(devices, device)
		if err != nil {
			return "", err
		}
		log.Printf("[INFO] HostIP discovered custom device matching %s: %s", network, device)
	}

	var lastError error
	for _, device := range devices {
		// parse dhcpd configuration
		pathDhcpConfig := d.DhcpConfPath(device)
		if _, err := os.Stat(pathDhcpConfig); err != nil {
			return "", fmt.Errorf("unable to find vmnetdhcp conf file: %s", pathDhcpConfig)
		}
		config, err := ReadDhcpConfig(pathDhcpConfig)
		if err != nil {
			lastError = err
			continue
		}

		// find the entry configured in the dhcpd
		interfaceConfig, err := config.HostByName(device)
		if err != nil {
			lastError = err
			continue
		}

		address, err := interfaceConfig.IP4()
		if err != nil {
			lastError = err
			continue
		}

		return address.String(), nil
	}
	return "", fmt.Errorf("unable to find host IP from devices %v, last error: %s", devices, lastError)
}

// GetDhcpLeasesPaths returns a copy of the DHCP leases paths.
func GetDhcpLeasesPaths() []string {
	return append([]string(nil), dhcpLeasesPaths...)
}

// GetDhcpConfPaths returns a copy of the DHCP configuration paths.
func GetDhcpConfPaths() []string {
	return append([]string(nil), dhcpConfPaths...)
}

// GetOvfTool returns the path to the `ovftool` binary if found in the system's PATH, otherwise returns an empty string.
func GetOvfTool() string {
	ovftool := appOvfTool
	if runtime.GOOS == osWindows {
		ovftool += ".exe"
	}

	if _, err := exec.LookPath(ovftool); err != nil {
		return ""
	}
	return ovftool
}

// CheckOvfToolVersion checks the version of the VMware OVF Tool.
func CheckOvfToolVersion(ovftoolPath string) error {
	output, err := exec.Command(ovftoolPath, "--version").CombinedOutput()
	if err != nil {
		log.Printf("[WARN] Error running 'ovftool --version': %v.", err)
		log.Printf("[WARN] Returned: %s", string(output))
		return errors.New("failed to execute ovftool")
	}
	versionOutput := string(output)
	log.Printf("[INFO] Returned ovftool version: %s.", versionOutput)

	versionString := ovfToolVersion.FindString(versionOutput)
	if versionString == "" {
		return errors.New("unable to determine the version of ovftool")
	}

	currentVersion, err := version.NewVersion(versionString)
	if err != nil {
		log.Printf("[WARN] Failed to parse version '%s': %v.", versionString, err)
		return fmt.Errorf("failed to parse ovftool version: %v", err)
	}

	if currentVersion.LessThan(ovfToolMinVersionObj) {
		log.Printf("[WARN] The version of ovftool (%s) is below the minimum recommended version (%s). Please download the latest version from %s.", currentVersion, ovfToolMinVersionObj, ovfToolDownloadURL)
		// Log a warning; do not return an error.
		// TODO: Transition this to an error in a future major release.
	}

	return nil
}

// Export runs the ovftool command-line utility with the specified arguments for exporting the virtual machines.
func (d *VmwareDriver) Export(args []string) error {
	ovftool := GetOvfTool()
	if ovftool == "" {
		return errors.New("error finding ovftool in path")
	}
	cmd := exec.Command(ovftool, args...)
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

// VerifyOvfTool ensures the VMware OVF Tool is installed, available in the system's PATH, and meets the required
// version.
func (d *VmwareDriver) VerifyOvfTool(SkipExport, _ bool) error {
	if SkipExport {
		return nil
	}

	log.Printf("[INFO] Verifying that ovftool exists...")
	ovftoolPath := GetOvfTool()
	if ovftoolPath == "" {
		return errors.New("ovftool not found; install and include it in your PATH")
	}

	log.Printf("[INFO] Checking ovftool version...")
	if err := CheckOvfToolVersion(ovftoolPath); err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}
