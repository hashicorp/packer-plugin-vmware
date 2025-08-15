// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

const (
	// VMware Fusion.
	fusionProductName     = "VMware Fusion"
	fusionMinVersion      = "13.5.0"
	fusionAppPath         = "/Applications/VMware Fusion.app"
	fusionAppPathVariable = "FUSION_APP_PATH"
	fusionSuppressPlist   = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>disallowUpgrade</key>
    <true/>
</dict>
</plist>`

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
	appVdiskManager = "vmware-vdiskmanager"
	appVmrun        = "vmrun"
	appVmware       = "vmware"
	appVmx          = "vmware-vmx"

	// Connection testing constants.
	sshConnectionTimeout      = 2 * time.Second
	sshConcurrentTestsMaximum = 10
	sshDefaultPort            = 22

	// Version regular expressions.
	productVersionRegex   = `(?i)VMware [a-z0-9-]+ (\d+\.\d+\.\d+)`
	technicalPreviewRegex = `(?i)VMware [a-z0-9-]+ e\.x\.p `
	ovfToolVersionRegex   = `\d+\.\d+\.\d+`

	// File names.
	dhcpVmnetConfFile   = "vmnetdhcp.conf"
	dhcpVmnetLeasesFile = "vmnetdhcp.leases"
	natVmnetConfFile    = "vmnetnat.conf"
	netmapConfFile      = "netmap.conf"

	// VNC settings.
	defaultVNCPortMin     = 5900
	defaultVNCPortMax     = 6000
	defaultVNCBindAddress = "127.0.0.1"

	// DefaultNamePrefix is the default prefix used for naming resources in the system.
	DefaultNamePrefix = "packer"

	// FirmwareTypeBios represents a constant for the BIOS firmware type identifier..
	FirmwareTypeBios = "bios"
	// FirmwareTypeUEFI represents a constant for the UEFI firmware type identifier.
	FirmwareTypeUEFI = "efi"
	// FirmwareTypeUEFISecure represents a constant for the UEFI firmware with secure boot type identifier.
	FirmwareTypeUEFISecure = "efi-secure"

	// Shutdown operation timings.
	shutdownPollInterval     = 150 * time.Millisecond
	shutdownLockTimeout      = 120 * time.Second
	shutdownLockPollInterval = 150 * time.Millisecond
	shutdownCleanupDelay     = 5 * time.Second

	// ExportFormatOvf defines the export format as "ovf" for Open Virtualization Format.
	ExportFormatOvf = "ovf"
	// exportFormatOva defines the export format as "ova" for Open Virtualization Appliance.
	exportFormatOva = "ova"
	// ExportFormatVmx defines the export format as "vmx" for Virtual Machine eXchange.
	ExportFormatVmx = "vmx"

	// Tools flavors.
	toolsFlavorMacOS   = osMacOS
	toolsFlavorLinux   = osLinux
	toolsFlavorWindows = osWindows

	// DefaultMemorySize specifies the default memory size (in MB) for a virtual machine configuration.
	DefaultMemorySize = 512
	// DefaultDiskSize specifies the default size, in megabytes, allocated for a virtual machine's primary disk.
	DefaultDiskSize = 40000
	// DefaultDiskType specifies the default disk type for a virtual machine's primary disk.
	DefaultDiskType = "1" // Growable virtual disk split in 2GB files. Must be a string, not an int.
	// defaultDiskName specifies the default disk name for a virtual machine's primary disk.
	defaultDiskName = "disk"
	// defaultDiskAdapterType specifies the default disk adapter type for a virtual machine's primary disk.'
	defaultDiskAdapterType = "lsilogic"
	// DefaultGuestOsType specifies the default guest operating system type for a virtual machine.
	DefaultGuestOsType = "other"
	// DefaultNetworkType specifies the default network type for a virtual machine.
	DefaultNetworkType = "nat"
	// DefaultNetworkAdapterType specifies the default network adapter type for a virtual machine.
	DefaultNetworkAdapterType = "e1000"
)

// Versions for supported or required components.
var (
	fusionMinVersionObj      = version.Must(version.NewVersion(fusionMinVersion))
	workstationMinVersionObj = version.Must(version.NewVersion(workstationMinVersion))
	ovfToolMinVersionObj     = version.Must(version.NewVersion(ovfToolMinVersion))
)

// The allowed export formats for a virtual machine.
var allowedExportFormats = []string{
	ExportFormatOvf,
	exportFormatOva,
	ExportFormatVmx,
}

// The allowed firmware types for a virtual machine.
var allowedFirmwareTypes = []string{
	FirmwareTypeBios,
	FirmwareTypeUEFI,
	FirmwareTypeUEFISecure,
}

// The allowed values for the `ToolsUploadFlavor`.
var allowedToolsFlavorValues = []string{
	toolsFlavorMacOS,
	toolsFlavorLinux,
	toolsFlavorWindows,
}

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

// The file extensions to retain when cleaning up files in a virtual machine environment.
var skipCleanFileExtensions = []string{
	".nvram",
	".vmdk",
	".vmsd",
	".vmx",
	".vmxf",
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
		case osMacOS:
			drivers = []Driver{
				NewFusionDriver(dconfig, config),
			}
		case osLinux:
			fallthrough
		case osWindows:
			drivers = []Driver{
				NewWorkstationDriver(config),
			}
		default:
			return nil, fmt.Errorf("error finding a driver for %s", runtime.GOOS)
		}
	}

	errs := ""
	for _, driver := range drivers {
		err := driver.Verify()

		log.Printf("[INFO] Using driver %T, Success: %t", driver, err == nil)
		if err == nil {
			return driver, nil
		}

		log.Printf("[INFO] Skipping %T because it failed with the following error %s", driver, err)
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

// testIPConnectivity tests if a single IP address is reachable on the specified port.
func testIPConnectivity(ip string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// findFirstReachableIP tests multiple IP addresses concurrently and returns the first reachable one.
func findFirstReachableIP(ips []string, port int, timeout time.Duration) (string, error) {
	if len(ips) == 0 {
		return "", errors.New("no IPs to test")
	}

	// For single IP, test directly
	if len(ips) == 1 {
		if testIPConnectivity(ips[0], port, timeout) {
			return ips[0], nil
		}
		return "", fmt.Errorf("IP %s is not reachable on port %d", ips[0], port)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Duration(len(ips)))
	defer cancel()

	resultChan := make(chan string, 1)
	ipChan := make(chan string, len(ips))
	var wg sync.WaitGroup

	// Limit concurrent goroutines to sshConcurrentTestsMaximum
	workerCount := len(ips)
	if workerCount > sshConcurrentTestsMaximum {
		workerCount = sshConcurrentTestsMaximum
	}

	// Start limited number of worker goroutines
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				if testIPConnectivity(ip, port, timeout) {
					select {
					case resultChan <- ip:
						return // Exit worker after finding first reachable IP
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Send IPs to workers
	go func() {
		defer close(ipChan)
		for _, ip := range ips {
			select {
			case ipChan <- ip:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for completion in background
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	select {
	case ip := <-resultChan:
		if ip != "" {
			return ip, nil
		}
		return "", fmt.Errorf("no reachable IPs found among %v on port %d", ips, port)
	case <-ctx.Done():
		return "", fmt.Errorf("timeout waiting for reachable IP among %v on port %d", ips, port)
	}
}

// filterAndSortLeaseEntries sorts DHCP lease entries by preference (most recent first) and returns IP addresses.
func filterAndSortLeaseEntries(entries []dhcpLeaseEntry) []string {
	if len(entries) == 0 {
		return nil
	}

	// Sort by lease start time (most recent first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].starts.After(entries[j].starts)
	})

	// Extract IP addresses
	addrs := make([]string, len(entries))
	for i, entry := range entries {
		addrs[i] = entry.address
	}
	return addrs
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

	log.Printf("[INFO] Looking up IP information...")
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
	log.Printf("[INFO] Discovered MAC address: %s", macAddress)

	res, err := net.ParseMAC(macAddress)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

// PotentialGuestIP identifies potential guest IP addresses for a virtual machine using DHCP leases and MAC address.
func (d *VmwareDriver) PotentialGuestIP(state multistep.StateBag) ([]string, error) {

	// Get the network mapper.
	netmap, err := d.NetworkMapper()
	if err != nil {
		return []string{}, err
	}

	// Convert the stashed network to a device.
	network := state.Get("vmnetwork").(string)
	devices, err := netmap.NameIntoDevices(network)

	// Log the results to see what was detected.
	for _, device := range devices {
		log.Printf("[INFO] Discovered device matching %s: %s", network, device)
	}

	// Unable to find the device; it may be a custom one.
	// Check to see if it's in the .vmx configuration.
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

	// Determine the MAC address for looking up the guest address.
	GuestAddress, err := d.GuestAddress(state)
	if err != nil {
		return []string{}, err
	}

	// Parse the MAC address once and reuse it throughout the function
	macAddress, err := net.ParseMAC(GuestAddress)
	if err != nil {
		return []string{}, fmt.Errorf("invalid MAC address %s: %v", GuestAddress, err)
	}

	// Iterate through all of the devices and collect all the DHCP lease entries.
	var availableLeaseEntries []dhcpLeaseEntry

	for _, device := range devices {
		// Determine the path to the DHCP leases file for the device.
		dhcpLeasesPath := d.DhcpLeasesPath(device)
		log.Printf("[INFO] Trying DHCP leases path: %s", dhcpLeasesPath)
		if dhcpLeasesPath == "" {
			return []string{}, fmt.Errorf("no DHCP leases path found for device %s", device)
		}

		// Open the path to the DHCP leases file.
		fh, err := os.Open(dhcpLeasesPath)
		if err != nil {
			log.Printf("[ERROR] Error while reading DHCP lease path file %s: %s", dhcpLeasesPath, err.Error())
			continue
		}

		// Read the contents of the DHCP leases file.
		leaseEntries, err := ReadDhcpdLeaseEntries(fh)
		// Close the file immediately after reading.
		fh.Close()
		if err != nil {
			return []string{}, err
		}

		// Iterate through the lease entries and find the ones are within the scope and match the MAC address.
		results := make([]dhcpLeaseEntry, 0)
		now := time.Now().UTC()

		for _, entry := range leaseEntries {
			// Check if the lease entry is valid.
			if !now.After(entry.starts) || !now.Before(entry.ends) {
				continue
			}

			// Check if the lease entry is within the scope.
			if !bytes.Equal(macAddress, entry.ether) {
				continue
			}

			// Store the result that matches the MAC address.
			results = append(results, entry)
		}

		// No result found; fallback to expired lease matching.
		// Look for any entries where the hardware address matches, ignoring lease time bounds.
		if len(results) == 0 {
			log.Printf("[WARN] No active DHCP leases found. Searching expired leases for hardware address %s", GuestAddress)
			for _, entry := range leaseEntries {
				if bytes.Equal(macAddress, entry.ether) {
					results = append(results, entry)
				}
			}
		}

		// If a result was found, add it to the current list of lease entries.
		if len(results) > 0 {
			availableLeaseEntries = append(availableLeaseEntries, results...)
		}

		// Map the results to get the address to return to the caller.
		// Iterate through the results to determine which one is online and potentially relevant.
	}

	// Check if any of the lease entries correspond to the MAC address.
	// If so, test connectivity and return the first working IP.
	if len(availableLeaseEntries) > 0 {
		// Sort entries by preference (most recent leases first).
		addrs := filterAndSortLeaseEntries(availableLeaseEntries)

		log.Printf("[INFO] Found %d potential IP addresses, testing connectivity...", len(addrs))

		// Try to find the first reachable IP with concurrent testing.
		if workingIP, err := findFirstReachableIP(addrs, sshDefaultPort, sshConnectionTimeout); err == nil {
			log.Printf("[INFO] Found reachable IP: %s", workingIP)
			return []string{workingIP}, nil
		} else {
			log.Printf("[WARN] No IPs were reachable on port %d: %v", sshDefaultPort, err)
			// Fall back to returning all IPs for backward compatibility.
			return addrs, nil
		}
	}

	<<<<<<< Updated upstream
	if runtime.GOOS == osMacOS {
		// We have match no vmware DHCP lease for this MAC. We'll try to match it in Apple DHCP leases.
		// As a remember, VMware is no longer able to rely on its own dhcpd server on MacOS BigSur and is
		// forced to use Apple DHCPD server instead.
		=======
		if runtime.GOOS == "darwin" {
			// No match found for the MAC address.
			// Attempt to match the MAC address to an entry in the Apple DHCP leases file.
			// As of macOS BigSur, VMware Fusion is no longer able to rely on its own dhcpd server and uses Apple dhcpd.
			>>>>>>> Stashed changes

			// Set the path to the Apple DHCP leases file.
			appleDhcpLeasesPath := "/var/db/dhcpd_leases"
			log.Printf("[INFO] Trying Apple DHCP leases path: %s", appleDhcpLeasesPath)

			// Open the Apple DHCP leases file.
			fh, err := os.Open(appleDhcpLeasesPath)
			if err != nil {
				log.Printf("[ERROR] Error while reading Apple DHCP lease path file %s: %s", appleDhcpLeasesPath, err.Error())
			} else {
				// Read the contents of the Apple DHCP leases file.
				appleLeaseEntries, err := ReadAppleDhcpdLeaseEntries(fh)
				// Close the file immediately after reading.
				fh.Close()
				if err != nil {
					return []string{}, err
				}

				// Iterate through the Apple lease entries and find the ones that match the MAC address.
				matchingAppleEntries := make([]appleDhcpLeaseEntry, 0)
				for _, entry := range appleLeaseEntries {
					// Check for hardware address match using the already parsed macAddress
					if bytes.Equal(macAddress, entry.hwAddress) {
						matchingAppleEntries = append(matchingAppleEntries, entry)
					}
				}

				// Check if any of the Apple lease entries correspond to the MAC address.
				// If so, test connectivity and return the first working IP.
				if len(matchingAppleEntries) > 0 {
					appleAddrs := make([]string, 0, len(matchingAppleEntries))
					for _, entry := range matchingAppleEntries {
						appleAddrs = append(appleAddrs, entry.ipAddress)
					}

					log.Printf("[INFO] Found %d potential Apple DHCP IP addresses, testing connectivity...", len(appleAddrs))

					// Try to find the first reachable IP with concurrent testing.
					if workingIP, err := findFirstReachableIP(appleAddrs, sshDefaultPort, sshConnectionTimeout); err == nil {
						log.Printf("[INFO] Found reachable Apple DHCP IP: %s", workingIP)
						return []string{workingIP}, nil
					} else {
						log.Printf("[WARN] No Apple DHCP IPs were reachable on port %d: %v", sshDefaultPort, err)
						// Fall back to returning all IPs for backward compatibility.
						return appleAddrs, nil
					}
				}
			}
		}

		return []string{}, fmt.Errorf("none of the found device(s) %v have a DHCP lease for MAC address %s", devices, GuestAddress)
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
		return fmt.Errorf("ovftool version %s is incompatible; requires version %s or later, download from %s", currentVersion, ovfToolMinVersionObj, ovfToolDownloadURL)
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
