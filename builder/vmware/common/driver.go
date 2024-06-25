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
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

const (
	ovfToolDownloadURL = "https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest"
	ovfToolMinVersion  = "4.6.0"
)

// The minimum recommended version of the VMware OVF Tool.
var ovfToolMinRecommended = version.Must(version.NewVersion(ovfToolMinVersion))

// A regex to match the version of the VMware OVF Tool.
var ovfToolVersionRegex = regexp.MustCompile(`\d+\.\d+\.\d+`)

// A driver is able to talk to VMware, control virtual machines, etc.
type Driver interface {
	// Clone clones the VMX and the disk to the destination path. The
	// destination is a path to the VMX file. The disk will be copied
	// to that same directory.
	Clone(dst string, src string, cloneType bool, snapshot string) error

	// CompactDisk compacts a virtual disk.
	CompactDisk(string) error

	// CreateDisk creates a virtual disk with the given size.
	CreateDisk(string, string, string, string) error

	// CreateSnapshot creates a snapshot of the supplied .vmx file with
	// the given name
	CreateSnapshot(string, string) error

	// Checks if the VMX file at the given path is running.
	IsRunning(string) (bool, error)

	// Start starts a VM specified by the path to the VMX given.
	Start(string, bool) error

	// Stop stops a VM specified by the path to the VMX given.
	Stop(string) error

	// SuppressMessages modifies the VMX or surrounding directory so that
	// VMware doesn't show any annoying messages.
	SuppressMessages(string) error

	// Get the path to the VMware ISO for the given flavor.
	ToolsIsoPath(string) string

	// Attach the VMware tools ISO
	ToolsInstall() error

	// Verify checks to make sure that this driver should function
	// properly. This should check that all the files it will use
	// appear to exist and so on. If everything is okay, this doesn't
	// return an error. Otherwise, this returns an error. Each vmware
	// driver should assign the VmwareMachine callback functions for locating
	// paths within this function.
	Verify() error

	/// This is to establish a connection to the guest
	CommHost(multistep.StateBag) (string, error)

	/// These methods are generally implemented by the VmwareDriver
	/// structure within this file. A driver implementation can
	/// reimplement these, though, if it wants.
	GetVmwareDriver() VmwareDriver

	// Get the guest hw address for the vm
	GuestAddress(multistep.StateBag) (string, error)

	// Get the guest ip address for the vm
	PotentialGuestIP(multistep.StateBag) ([]string, error)

	// Get the host hw address for the vm
	HostAddress(multistep.StateBag) (string, error)

	// Get the host ip address for the vm
	HostIP(multistep.StateBag) (string, error)

	// Export the vm to ovf or ova format using ovftool
	Export([]string) error

	// OvfTool
	VerifyOvfTool(bool, bool) error
}

// NewDriver returns a new driver implementation for this operating
// system, or an error if the driver couldn't be initialized.
func NewDriver(dconfig *DriverConfig, config *SSHConfig, vmName string) (Driver, error) {
	var drivers []Driver

	if dconfig.RemoteType != "" {
		esx5Driver, err := NewESX5Driver(dconfig, config, vmName)
		if err != nil {
			return nil, err
		}
		drivers = []Driver{esx5Driver}

	} else {
		switch runtime.GOOS {
		case "darwin":
			drivers = []Driver{
				NewFusion6Driver(dconfig, config),
				NewFusion5Driver(dconfig, config),
			}
		case "linux":
			fallthrough
		case "windows":
			drivers = []Driver{
				NewWorkstation10Driver(config),
				NewWorkstation9Driver(config),
				NewPlayer6Driver(config),
				NewPlayer5Driver(config),
			}
		default:
			return nil, fmt.Errorf("can't find driver for OS: %s", runtime.GOOS)
		}
	}

	errs := ""
	for _, driver := range drivers {
		err := driver.Verify()

		log.Printf("Testing against vmware driver %T, Success: %t", driver, err == nil)
		if err == nil {
			return driver, nil
		}

		log.Printf("skipping %T because it failed with the following error %s", driver, err)
		errs += "* " + err.Error() + "\n"
	}

	return nil, fmt.Errorf("driver initialization failed. fix at least one driver to continue:\n%s", errs)
}

func runAndLog(cmd *exec.Cmd) (string, string, error) {
	var stdout, stderr bytes.Buffer

	log.Printf("Executing: %s %s", cmd.Path, strings.Join(cmd.Args[1:], " "))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	stdoutString := strings.TrimSpace(stdout.String())
	stderrString := strings.TrimSpace(stderr.String())

	if _, ok := err.(*exec.ExitError); ok {
		message := stderrString
		if message == "" {
			message = stdoutString
		}

		err = fmt.Errorf("VMware error: %s", message)

		// If "unknown error" is in there, add some additional notes
		re := regexp.MustCompile(`(?i)unknown error`)
		if re.MatchString(message) {
			err = fmt.Errorf(
				"%s\n\n%s", err,
				"Packer detected a VMware 'Unknown Error'. Unfortunately VMware\n"+
					"often has extremely vague error messages such as this and Packer\n"+
					"itself can't do much about that. Please check the vmware.log files\n"+
					"created by VMware when a VM is started (in the directory of the\n"+
					"vmx file), which often contains more detailed error information.\n\n"+
					"You may need to set the command line flag --on-error=abort to\n\n"+
					"prevent Packer from cleaning up the vmx file directory.")
		}
	}

	log.Printf("stdout: %s", stdoutString)
	log.Printf("stderr: %s", stderrString)

	// Replace these for Windows, we only want to deal with Unix
	// style line endings.
	returnStdout := strings.Replace(stdout.String(), "\r\n", "\n", -1)
	returnStderr := strings.Replace(stderr.String(), "\r\n", "\n", -1)

	return returnStdout, returnStderr, err
}

func normalizeVersion(version string) (string, error) {
	i, err := strconv.Atoi(version)
	if err != nil {
		return "", fmt.Errorf("returned a non-integer version %q: %s", version, err)
	}

	return fmt.Sprintf("%02d", i), nil
}

func compareVersions(versionFound string, versionWanted string, product string) error {
	found, err := normalizeVersion(versionFound)
	if err != nil {
		return err
	}

	wanted, err := normalizeVersion(versionWanted)
	if err != nil {
		return err
	}

	if found < wanted {
		return fmt.Errorf("requires %s or later, found %s", versionWanted, versionFound)
	}

	return nil
}

// helper functions that read configuration information from a file
// read the network<->device configuration out of the specified path
func ReadNetmapConfig(path string) (NetworkMap, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return ReadNetworkMap(fd)
}

// read the dhcp configuration out of the specified path
func ReadDhcpConfig(path string) (DhcpConfiguration, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return ReadDhcpConfiguration(fd)
}

// read the VMX configuration from the specified path
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

// read the connection type out of a vmx configuration
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

// This VmwareDriver is a base class that contains default methods
// that a Driver can use or implement themselves.
type VmwareDriver struct {
	/// These methods define paths that are utilized by the driver
	/// A driver must overload these in order to point to the correct
	/// files so that the address detection (ip and ethernet) machinery
	/// works.
	DhcpLeasesPath   func(string) string
	DhcpConfPath     func(string) string
	VmnetnatConfPath func(string) string

	/// This method returns an object with the NetworkNameMapper interface
	/// that maps network to device and vice-versa.
	NetworkMapper func() (NetworkNameMapper, error)
}

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
	log.Printf("GuestAddress discovered MAC address: %s", macAddress)

	res, err := net.ParseMAC(macAddress)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

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
		log.Printf("GuestIP discovered device matching %s: %s", network, device)
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
		log.Printf("GuestIP discovered custom device matching %s: %s", network, device)
	}

	// figure out our MAC address for looking up the guest address
	MACAddress, err := d.GuestAddress(state)
	if err != nil {
		return []string{}, err
	}

	// iterate through all of the devices and collect all the dhcp lease entries
	// that we possibly cacn.
	var available_lease_entries []dhcpLeaseEntry

	for _, device := range devices {
		// figure out the correct dhcp leases
		dhcpLeasesPath := d.DhcpLeasesPath(device)
		log.Printf("Trying DHCP leases path: %s", dhcpLeasesPath)
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
			if !(now.After(entry.starts) && now.Before(entry.ends)) {
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
			available_lease_entries = append(available_lease_entries, results...)
		}

		// Now we need to map our results to get the address so we can return it.iterate through our results and figure out which one
		// is actually up...and should be relevant.
	}

	// Check if we found any lease entries that correspond to us. If so, then we
	// need to map() them in order to extract the address field to return to the
	// caller.
	if len(available_lease_entries) > 0 {
		addrs := make([]string, 0)
		for _, entry := range available_lease_entries {
			addrs = append(addrs, entry.address)
		}
		return addrs, nil
	}

	if runtime.GOOS == "darwin" {
		// We have match no vmware DHCP lease for this MAC. We'll try to match it in Apple DHCP leases.
		// As a remember, VMware is no longer able to rely on its own dhcpd server on MacOS BigSur and is
		// forced to use Apple DHCPD server instead.
		// https://communities.vmware.com/t5/VMware-Fusion-Discussions/Big-Sur-hosts-with-Fusion-Is-vmnet-dhcpd-vmnet8-leases-file/m-p/2298927/highlight/true#M140003

		// set the apple dhcp leases path
		appleDhcpLeasesPath := "/var/db/dhcpd_leases"
		log.Printf("Trying Apple DHCP leases path: %s", appleDhcpLeasesPath)

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
			available_lease_entries := make([]appleDhcpLeaseEntry, 0)
			for _, entry := range leaseEntries {
				// Next check for any where the hardware address matches.
				if bytes.Equal(hwaddr, entry.hwAddress) {
					available_lease_entries = append(available_lease_entries, entry)
				}
			}

			// Check if we found any lease entries that correspond to us. If so, then we
			// need to map() them in order to extract the address field to return to the
			// caller.
			if len(available_lease_entries) > 0 {
				addrs := make([]string, 0)
				for _, entry := range available_lease_entries {
					addrs = append(addrs, entry.ipAddress)
				}
				return addrs, nil
			}
		}
	}

	return []string{}, fmt.Errorf("none of the found device(s) %v have a DHCP lease for MAC address %s", devices, MACAddress)
}

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
		log.Printf("HostAddress discovered device matching %s: %s", network, device)
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
		log.Printf("HostAddress discovered custom device matching %s: %s", network, device)
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
			names = append(names, intf.Name)
		}
	}
	return "", fmt.Errorf("unable to find host address from devices %v, last error: %s", devices, lastError)
}

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
		log.Printf("HostIP discovered device matching %s: %s", network, device)
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
		log.Printf("HostIP discovered custom device matching %s: %s", network, device)
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

func GetOvfTool() string {
	ovftool := "ovftool"
	if runtime.GOOS == "windows" {
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
	log.Printf("Returned ovftool version: %s.", versionOutput)

	versionString := ovfToolVersionRegex.FindString(versionOutput)
	if versionString == "" {
		return errors.New("unable to determine the version of ovftool")
	}

	currentVersion, err := version.NewVersion(versionString)
	if err != nil {
		log.Printf("[WARN] Failed to parse version '%s': %v.", versionString, err)
		return fmt.Errorf("failed to parse ovftool version: %v", err)
	}

	if currentVersion.LessThan(ovfToolMinRecommended) {
		log.Printf("[WARN] The version of ovftool (%s) is below the minimum recommended version (%s). Please download the latest version from %s.", currentVersion, ovfToolMinRecommended, ovfToolDownloadURL)
		// Log a warning; do not return an error.
		// TODO: Transition this to an error in a future major release.
	}

	return nil
}

func (d *VmwareDriver) Export(args []string) error {
	ovftool := GetOvfTool()
	if ovftool == "" {
		return fmt.Errorf("error finding ovftool in path")
	}
	cmd := exec.Command(ovftool, args...)
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

func (d *VmwareDriver) VerifyOvfTool(SkipExport, _ bool) error {
	if SkipExport {
		return nil
	}

	log.Printf("Verifying that ovftool exists...")
	ovftoolPath := GetOvfTool()
	if ovftoolPath == "" {
		return fmt.Errorf("ovftool not found; install and include it in your PATH")
	}

	log.Printf("Checking ovftool version...")
	if err := CheckOvfToolVersion(ovftoolPath); err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}
