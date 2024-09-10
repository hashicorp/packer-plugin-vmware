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

	"golang.org/x/sys/windows"
)

func workstationCheckLicense() error {
	// Not used on Windows.
	return nil
}

func workstationFindVdiskManager() (string, error) {
	path, err := exec.LookPath("vmware-vdiskmanager.exe")
	if err == nil {
		return path, nil
	}

	return findFile("vmware-vdiskmanager.exe", workstationProgramFilePaths()), nil
}

func workstationFindVMware() (string, error) {
	path, err := exec.LookPath("vmware.exe")
	if err == nil {
		return path, nil
	}

	return findFile("vmware.exe", workstationProgramFilePaths()), nil
}

func workstationFindVmrun() (string, error) {
	path, err := exec.LookPath("vmrun.exe")
	if err == nil {
		return path, nil
	}

	return findFile("vmrun.exe", workstationProgramFilePaths()), nil
}

func workstationToolsIsoPath(flavor string) string {
	return findFile(flavor+".iso", workstationProgramFilePaths())
}

// Read the DHCP leases path from the registry.
func workstationDhcpLeasesPath(device string) string {
	path, err := workstationDhcpLeasesPathRegistry()
	if err != nil {
		log.Printf("Error finding leases in registry: %s", err)
	} else if _, err := os.Stat(path); err == nil {
		return path
	}

	return findFile("vmnetdhcp.leases", workstationDataFilePaths())
}

// Read the DHCP configuration path from the registry.
func workstationDhcpConfPath(device string) string {
	// Not used on Windows.
	return findFile("vmnetdhcp.conf", workstationDataFilePaths())
}

func workstationVmnetnatConfPath(device string) string {
	// Not used on Windows.
	return findFile("vmnetnat.conf", workstationDataFilePaths())
}

func workstationNetmapConfPath() string {
	return findFile("netmap.conf", workstationDataFilePaths())
}

// Read the installation path from the registry.
func workstationVMwareRoot() (s string, err error) {
	key := `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\vmware.exe`
	subkey := "Path"
	s, err = readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		log.Printf(`Unable to read registry key %s\%s`, key, subkey)
		return
	}

	return normalizePath(s), nil
}

// Read the DHCP leases path from the registry.
func workstationDhcpLeasesPathRegistry() (s string, err error) {
	key := "SYSTEM\\CurrentControlSet\\services\\VMnetDHCP\\Parameters"
	subkey := "LeaseFile"
	s, err = readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		log.Printf(`Unable to read registry key %s\%s`, key, subkey)
		return
	}

	return normalizePath(s), nil
}

// workstationProgramFilesPaths returns a list of paths that are eligible
// to contain the VMware Workstation binaries.
func workstationProgramFilePaths() []string {
	path, err := workstationVMwareRoot()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
	}

	paths := make([]string, 0, 5)
	if os.Getenv("VMWARE_HOME") != "" {
		paths = append(paths, os.Getenv("VMWARE_HOME"))
	}

	if path != "" {
		paths = append(paths, path)
	}

	if os.Getenv("ProgramFiles(x86)") != "" {
		paths = append(paths,
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "/VMware/VMware Workstation"))
	}

	if os.Getenv("ProgramFiles") != "" {
		paths = append(paths,
			filepath.Join(os.Getenv("ProgramFiles"), "/VMware/VMware Workstation"))
	}

	return paths
}

// workstationDataFilePaths returns a list of paths that are eligible
// to contain configuration files.
func workstationDataFilePaths() []string {
	leasesPath, err := workstationDhcpLeasesPathRegistry()
	if err != nil {
		log.Printf("Error retrieving DHCP leases path from registry: %s", err)
	}

	if leasesPath != "" {
		leasesPath = filepath.Dir(leasesPath)
	}

	paths := make([]string, 0, 5)
	if os.Getenv("VMWARE_DATA") != "" {
		paths = append(paths, os.Getenv("VMWARE_DATA"))
	}

	if leasesPath != "" {
		paths = append(paths, leasesPath)
	}

	if os.Getenv("ProgramData") != "" {
		paths = append(paths,
			filepath.Join(os.Getenv("ProgramData"), "/VMware"))
	}

	if os.Getenv("ALLUSERSPROFILE") != "" {
		paths = append(paths,
			filepath.Join(os.Getenv("ALLUSERSPROFILE"), "/Application Data/VMware"))
	}

	return paths
}

func workstationVerifyVersion(version string) error {
	key := `SOFTWARE\Wow6432Node\VMware, Inc.\VMware Workstation`
	subkey := "ProductVersion"
	productVersion, err := readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		log.Printf(`Unable to read registry key %s\%s`, key, subkey)
		key = `SOFTWARE\VMware, Inc.\VMware Workstation`
		productVersion, err = readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
		if err != nil {
			log.Printf(`Unable to read registry key %s\%s`, key, subkey)
			return err
		}
	}

	versionRe := regexp.MustCompile(`^(\d+)\.`)
	matches := versionRe.FindStringSubmatch(productVersion)
	if matches == nil {
		return fmt.Errorf("error retrieving the version from registry key %s\\%s: '%s'", key, subkey, productVersion)
	}
	log.Printf("VMware Workstation: %s", matches[1])

	return compareVersions(matches[1], version, "Workstation")
}

// Read Windows registry data.
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

func normalizePath(path string) string {
	path = strings.Replace(path, "\\", "/", -1)
	path = strings.Replace(path, "//", "/", -1)
	path = strings.TrimRight(path, "/")
	return path
}

func findFile(file string, paths []string) string {
	for _, path := range paths {
		path = filepath.Join(path, file)
		path = normalizePath(path)
		log.Printf("Searching for file '%s'", path)

		if _, err := os.Stat(path); err == nil {
			log.Printf("Found file '%s'", path)
			return path
		}
	}

	log.Printf("File not found: '%s'", file)
	return ""
}
