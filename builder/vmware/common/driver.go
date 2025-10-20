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
	// Builder ID.
	builderId = "vmware.desktop"

	// Artifact configuration keys.
	artifactConfFormat     = "artifact.conf.format"
	artifactConfSkipExport = "artifact.conf.skip_export"

	// VMware Fusion.
	fusionProductName     = "VMware Fusion"
	fusionMinVersion      = "13.6.0"
	fusionAppPath         = "/Applications/VMware Fusion.app"
	fusionAppPathVariable = "FUSION_APP_PATH"
	fusionPreferencesPath = "/Library/Preferences/VMware Fusion"
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
	workstationMinVersion       = "17.6.0"
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

	// Version regular expressions.
	productVersionRegex = `(?i)VMware [a-z0-9-]+ (\d+\.\d+\.\d+)`
	ovfToolVersionRegex = `\d+\.\d+\.\d+`

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

	// FirmwareTypeBios represents a constant for the BIOS firmware type identifier.
	FirmwareTypeBios = "bios"
	// FirmwareTypeUEFI represents a constant for the UEFI firmware type identifier.
	FirmwareTypeUEFI = "efi"
	// FirmwareTypeUEFISecure represents a constant for the UEFI firmware with secure boot type identifier.
	FirmwareTypeUEFISecure = "efi-secure"

	// Network adapter types.
	networkAdapterE1000   = "e1000"
	networkAdapterE1000E  = "e1000e"
	networkAdapterVmxnet3 = "vmxnet3"

	// CD-ROM adapter types.
	cdromAdapterIde  = "ide"
	cdromAdapterSata = "sata"
	cdromAdapterScsi = "scsi"

	// USB version types.
	UsbVersion20 = "2.0"
	UsbVersion31 = "3.1"
	UsbVersion32 = "3.2"

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

	// Tools mode constants
	toolsModeUpload  = "upload"
	toolsModeAttach  = "attach"
	toolsModeDisable = "disable"

	// Tools flavors.
	toolsFlavorMacOS   = osMacOS
	toolsFlavorLinux   = osLinux
	toolsFlavorWindows = osWindows

	// MinimumHardwareVersion specifies the minimum supported hardware version required for compatibility.
	MinimumHardwareVersion = 19
	// DefaultHardwareVersion specifies the default virtual hardware version used during virtual machine creation.
	DefaultHardwareVersion = 21
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
	// DefaultGuestOsTypeAmd64 specifies the default guest operating system type for a virtual machine running on AMD64.
	DefaultGuestOsTypeAmd64 = "other-64"
	// DefaultGuestOsTypeArm64 specifies the default guest operating system type for a virtual machine running on ARM64.
	DefaultGuestOsTypeArm64 = "arm-other-64"
	// FallbackGuestOsType specifies the default guest operating system type for a virtual machine if the runtime
	// architecture is not AMD64 or ARM64.
	FallbackGuestOsType = "other"
	// DefaultNetworkType specifies the default network type for a virtual machine.
	DefaultNetworkType = "nat"
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

// The allowed network adapter types for a virtual machine.
var allowedNetworkAdapterTypes = []string{
	networkAdapterVmxnet3,
	networkAdapterE1000E,
	networkAdapterE1000,
}

// AllowedCdromAdapterTypes defines the allowed CD-ROM adapter types for a virtual machine.
var AllowedCdromAdapterTypes = []string{
	cdromAdapterIde,
	cdromAdapterSata,
	cdromAdapterScsi,
}

// AllowedUsbVersions defines the allowed USB versions for a virtual machine.
var AllowedUsbVersions = []string{
	UsbVersion20,
	UsbVersion31,
	UsbVersion32,
}

// The allowed values for the `ToolsMode`.
var allowedToolsModeValues = []string{
	toolsModeUpload,
	toolsModeAttach,
	toolsModeDisable,
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
				"Packer detected an error from the desktop hypervisor "+
					"Unfortunately, the error message provided is "+
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
	// that maps network to device and vice versa.
	NetworkMapper func() (NetworkNameMapper, error)
}

// GuestAddress retrieves the MAC address of a guest virtual machine from the .vmx configuration.
func (d *VmwareDriver) GuestAddress(state multistep.StateBag) (string, error) {
	vmxPath := state.Get("vmx_path").(string)

	log.Println("[INFO] Looking up DHCP leases...")
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
	log.Printf("[INFO] MAC address: %s", macAddress)

	res, err := net.ParseMAC(macAddress)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

// PotentialGuestIP identifies potential guest IP addresses for a virtual machine using DHCP leases and MAC address.
func (d *VmwareDriver) PotentialGuestIP(state multistep.StateBag) ([]string, error) {
	// Get the network devices.
	devices, err := d.getNetworkDevices(state)
	if err != nil {
		return nil, err
	}

	// Get the MAC address of the guest.
	MACAddress, err := d.GuestAddress(state)
	if err != nil {
		return nil, err
	}

	// Parse the MAC address.
	hwaddr, err := net.ParseMAC(MACAddress)
	if err != nil {
		return nil, err
	}

	// Search the desktop hypervisor for DHCP leases.
	if addrs := d.getDhcpLeasesHypervisor(devices, hwaddr); len(addrs) > 0 {
		return addrs, nil
	}

	// If VMware Fusion on macOS, search the Apple DHCP leases as a fallback.
	if runtime.GOOS == osMacOS {
		if addrs := d.getDhcpLeasesMacos(hwaddr); len(addrs) > 0 {
			return addrs, nil
		}
	}

	return nil, fmt.Errorf("none of the found device(s) %v have a DHCP lease for MAC address %s", devices, MACAddress)
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
		log.Printf("[INFO] Discovered device matching %s: %s", network, device)
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
		log.Printf("[INFO] Discovered custom device matching %s: %s", network, device)
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
		log.Printf("[INFO] Discovered device matching %s: %s", network, device)
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
		log.Printf("[INFO] Discovered custom device matching %s: %s", network, device)
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

// CheckOvfToolVersion checks the version of the VMware OVF Tool.
func CheckOvfToolVersion(ovftoolPath string) error {
	output, err := exec.Command(ovftoolPath, "--version").CombinedOutput()
	if err != nil {
		log.Printf("[WARN] Failed to run 'ovftool --version': %v.", err)
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

// getNetworkDevices retrieves a list of network device names associated with a specific network from the given state.
func (d *VmwareDriver) getNetworkDevices(state multistep.StateBag) ([]string, error) {
	netmap, err := d.NetworkMapper()
	if err != nil {
		return nil, err
	}

	network := state.Get("vmnetwork").(string)
	devices, err := netmap.NameIntoDevices(network)

	if err != nil || network == "custom" {
		vmxPath := state.Get("vmx_path").(string)
		vmxData, err := readVMXConfig(vmxPath)
		if err != nil {
			return nil, err
		}

		device, err := readCustomDeviceName(vmxData)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
		log.Printf("[INFO] Discovered custom device matching %s: %s", network, device)
	}

	return devices, nil
}

// getDhcpLeasesHypervisor retrieves a list of IP addresses from the desktop hypervisor DHCP leases that match the given
// hardware address.
func (d *VmwareDriver) getDhcpLeasesHypervisor(devices []string, hwaddr net.HardwareAddr) []string {
	var addresses []string

	for _, device := range devices {
		leasesPath := d.DhcpLeasesPath(device)
		log.Printf("[INFO] Reading the desktop hypervisor DHCP leases path: %s", leasesPath)

		if leasesPath == "" {
			log.Printf("[WARN] No desktop hypervisor DHCP leases path found for device %s", device)
			continue
		}

		leases := d.parseDhcpLeases(leasesPath, hwaddr)
		addresses = append(addresses, leases...)
	}

	return addresses
}

// getDhcpLeasesMacos retrieves a list of IP addresses from macOS DHCP leases that match the given hardware address.
func (d *VmwareDriver) getDhcpLeasesMacos(hwaddr net.HardwareAddr) []string {
	leasesPath := "/var/db/dhcpd_leases"
	log.Printf("[INFO] Reading the macOS DHCP leases path: %s", leasesPath)

	fh, err := os.Open(leasesPath)
	if err != nil {
		log.Printf("[WARN] Failed to read the macOS DHCP leases file %s: %s", leasesPath, err.Error())
		return nil
	}
	defer func() {
		if err := fh.Close(); err != nil {
			log.Printf("[WARN] Failed to close the macOS DHCP leases file %s: %v", leasesPath, err)
		}
	}()

	leases, err := ReadAppleDhcpdLeaseEntries(fh)
	if err != nil {
		log.Printf("[WARN] Failed to read Apple DHCP lease entries: %s", err.Error())
		return nil
	}

	var addresses []string
	for _, entry := range leases {
		if bytes.Equal(hwaddr, entry.hwAddress) {
			addresses = append(addresses, entry.ipAddress)
		}
	}

	return addresses
}

// parseDhcpLeases parses a DHCP lease file to retrieve a list of IP addresses associated with a specific hardware address.
func (d *VmwareDriver) parseDhcpLeases(filePath string, hwaddr net.HardwareAddr) []string {
	fh, err := os.Open(filePath)
	if err != nil {
		log.Printf("[WARN] Failed to read DHCP lease path file %s: %s", filePath, err.Error())
		return nil
	}
	defer func() {
		if cerr := fh.Close(); cerr != nil {
			log.Printf("[WARN] Failed to close DHCP lease file %s: %v", filePath, cerr)
		}
	}()

	leases, err := ReadDhcpdLeaseEntries(fh)
	if err != nil {
		log.Printf("[WARN] Failed to read DHCP lease entries from %s: %s", filePath, err.Error())
		return nil
	}

	return d.getLeaseAddresses(leases, hwaddr)
}

// getLeaseAddresses returns a list of IP addresses associated with a given hardware address from DHCP lease entries.
func (d *VmwareDriver) getLeaseAddresses(leaseEntries []dhcpLeaseEntry, hwaddr net.HardwareAddr) []string {
	now := time.Now().UTC()
	var strictMatches, looseMatches []string

	for _, entry := range leaseEntries {
		if !bytes.Equal(hwaddr, entry.ether) {
			continue
		}

		// Check if lease is still valid (strict matching).
		if now.After(entry.starts) && now.Before(entry.ends) {
			strictMatches = append(strictMatches, entry.address)
		} else {
			// Collect for loose matching fallback.
			looseMatches = append(looseMatches, entry.address)
		}
	}

	// Return strict matches if found; otherwise fall back to loose matches.
	if len(strictMatches) > 0 {
		return strictMatches
	}

	if len(looseMatches) > 0 {
		log.Printf("[INFO] Failed to find an exact match for DHCP lease. Falling back to loose matching for hardware address %v", hwaddr)
		return looseMatches
	}

	return nil
}
