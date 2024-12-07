// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build windows

package common

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unsafe"

	"github.com/hashicorp/go-version"
	"golang.org/x/sys/windows"
)

// VMware Workstation on Windows

// workstationCheckLicense checks if a VMware Workstation license is present.
func workstationCheckLicense() error {
	// Not used on Windows.
	return nil
}

// workstationExecutable returns the path to a VMware Workstation executable.
func workstationExecutable(executable string) (string, error) {
	path, err := exec.LookPath(executable + ".exe")
	if err == nil {
		return path, nil
	}
	return findFile(executable+".exe", workstationProgramFilePaths()), nil
}

// workstationFindVMware returns the path to the VMware Workstation executable.
func workstationFindVMware() (string, error) {
	return workstationExecutable(appVmware)
}

// workstationFindVmrun returns the path to the VMware VIX executable.
func workstationFindVmrun() (string, error) {
	return workstationExecutable(appVmrun)
}

// workstationFindVdiskManager returns the path to the VMware Virtual Disk
// Manager executable.
func workstationFindVdiskManager() (string, error) {
	return workstationExecutable(appVdiskManager)
}

// workstationToolsIsoPath returns the path to the VMware Tools ISO.
func workstationToolsIsoPath(flavor string) string {
	return findFile(flavor+".iso", workstationProgramFilePaths())
}

// workstationInstallationPath reads the installation path from the registry.
func workstationInstallationPath() (string, error) {
	return workstationReadRegistryPath(workstationInstallationPathKey, "Path")
}

// workstationDhcpLeasesPath returns the path to the DHCP leases file.
func workstationDhcpLeasesPath(device string) string {
	path, err := workstationDhcpLeasesPathRegistry()
	if err != nil {
		log.Printf("[WARN] Error finding leases in registry: %s", err)
	} else if _, err := os.Stat(path); err == nil {
		return path
	}

	return findFile(dhcpVmnetLeasesFile, workstationDataFilePaths())
}

// workstationDhcpConfPath returns the path to the DHCP configuration file.
func workstationDhcpConfPath(device string) string {
	// Not used on Windows.
	return findFile(dhcpVmnetConfFile, workstationDataFilePaths())
}

// workstationNatConfPath returns the path to the NAT configuration file.
func workstationNatConfPath(device string) string {
	// Not used on Windows.
	return findFile(natVmnetConfFile, workstationDataFilePaths())
}

// workstationNetmapConfPath returns the path to the network mapping
// configuration file.
func workstationNetmapConfPath() string {
	return findFile(netmapConfFile, workstationDataFilePaths())
}

// workstationReadRegistryPath reads a registry key and subkey and normalizes
// the path.
func workstationReadRegistryPath(key, subkey string) (string, error) {
	s, err := readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		return "", fmt.Errorf("unable to read registry key %s\\%s: %w", key, subkey, err)
	}
	return normalizePath(s), nil
}

// workstationDhcpLeasesPathRegistry reads the DHCP leases path from the
// registry.
func workstationDhcpLeasesPathRegistry() (string, error) {
	return workstationReadRegistryPath(workstationRegistryKey, "LeaseFile")
}

// workstationAppendPath appends the path if the environment variable exists.
func workstationAppendPath(paths []string, envVar, suffix string) []string {
	if value := os.Getenv(envVar); value != "" {
		paths = append(paths, filepath.Join(value, suffix))
	}
	return paths
}

// workstationProgramFilesPaths returns a list of paths that are eligible to
// contain the VMware Workstation binaries.
func workstationProgramFilePaths() []string {
	path, err := workstationInstallationPath()
	if err != nil {
		log.Printf("[WARN] Unable to retrieve installation path from registry: %s", err)
	}

	paths := make([]string, 0, 5)

	if homePath := os.Getenv("VMWARE_HOME"); homePath != "" {
		paths = append(paths, homePath)
	}

	if path != "" {
		paths = append(paths, path)
	}

	paths = workstationAppendPath(paths, "ProgramFiles(x86)", "VMware/VMware Workstation")
	paths = workstationAppendPath(paths, "ProgramFiles", "VMware/VMware Workstation")

	return paths
}

// workstationDataFilePaths returns a list of paths that are eligible to
// contain configuration files.
func workstationDataFilePaths() []string {
	leasesPath, err := workstationDhcpLeasesPathRegistry()
	if err != nil {
		log.Printf("[WARN] Unable to retrieve DHCP leases path from registry: %s", err)
	}

	if leasesPath != "" {
		leasesPath = filepath.Dir(leasesPath)
	}

	paths := make([]string, 0, 5)

	if dataPath := os.Getenv("VMWARE_DATA"); dataPath != "" {
		paths = append(paths, dataPath)
	}

	if leasesPath != "" {
		paths = append(paths, leasesPath)
	}

	paths = workstationAppendPath(paths, "ProgramData", "VMware")
	paths = workstationAppendPath(paths, "ALLUSERSPROFILE", "Application Data/VMware")

	return paths
}

// workstationVerifyVersion verifies the VMware Workstation version against the
// required version.
func workstationVerifyVersion(requiredVersion string) error {
	key := `SOFTWARE\Wow6432Node\VMware, Inc.\VMware Workstation`
	subkey := "ProductVersion"
	productVersion, err := readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		log.Printf(`[WARN] Unable to read registry key %s\%s`, key, subkey)
		key = `SOFTWARE\VMware, Inc.\VMware Workstation`
		productVersion, err = readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
		if err != nil {
			return fmt.Errorf("unable to read registry key %s\\%s: %w", key, subkey, err)
		}
	}

	versionRe := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := versionRe.FindStringSubmatch(productVersion)

	if matches == nil || len(matches) < 4 {
		return fmt.Errorf("error retrieving the version from registry key %s\\%s: '%s'", key, subkey, productVersion)
	}

	fullVersion := fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
	log.Printf("[INFO] %s: %s", workstationProductName, fullVersion)

	parsedVersionFound, err := version.NewVersion(fullVersion)
	if err != nil {
		return fmt.Errorf("invalid version found: %w", err)
	}

	parsedVersionRequired, err := version.NewVersion(requiredVersion)
	if err != nil {
		return fmt.Errorf("invalid version required: %w", err)
	}

	return compareVersionObjects(parsedVersionFound, parsedVersionRequired, workstationProductName)
}

// readRegString reads a string value from the registry.
func readRegString(hive syscall.Handle, subKeyPath, valueName string) (value string, err error) {
	var h syscall.Handle
	subKeyPathPtr, err := windows.UTF16PtrFromString(subKeyPath)
	if err != nil {
		return "", err
	}

	err = syscall.RegOpenKeyEx(hive, subKeyPathPtr, 0, syscall.KEY_READ, &h)
	if err != nil {
		return
	}
	defer syscall.RegCloseKey(h)

	var typ uint32
	var bufSize uint32
	valueNamePtr, err := windows.UTF16PtrFromString(valueName)
	if err != nil {
		return "", err
	}

	err = syscall.RegQueryValueEx(
		h,
		valueNamePtr,
		nil,
		&typ,
		nil,
		&bufSize)
	if err != nil {
		return
	}

	data := make([]uint16, bufSize/2+1)
	err = syscall.RegQueryValueEx(
		h,
		valueNamePtr,
		nil,
		&typ,
		(*byte)(unsafe.Pointer(&data[0])),
		&bufSize)
	if err != nil {
		return
	}

	return syscall.UTF16ToString(data), nil
}

// normalizePath normalizes a path.
func normalizePath(path string) string {
	path = strings.Replace(path, "\\", "/", -1)
	path = strings.Replace(path, "//", "/", -1)
	path = strings.TrimRight(path, "/")
	return path
}

// findFile searches for a file in a list of paths.
func findFile(file string, paths []string) string {
	for _, path := range paths {
		path = filepath.Join(path, file)
		path = normalizePath(path)
		log.Printf("[INFO] Searching for file '%s'", path)

		if _, err := os.Stat(path); err == nil {
			log.Printf("[INFO] Found file '%s'", path)
			return path
		}
	}

	log.Printf("[WARN] File not found: '%s'", file)
	return ""
}
