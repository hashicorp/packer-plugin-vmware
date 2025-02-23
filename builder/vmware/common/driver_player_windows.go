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
	"syscall"

	"github.com/hashicorp/go-version"
)

// VMware Workstation Player on Windows

// Registry paths for different VMware Workstation Player installation
// locations on Windows operating systems.
var playerInstallationRegistryPaths = []string{
	`SOFTWARE\Wow6432Node\VMware, Inc.\VMware Player`,
	`SOFTWARE\VMware, Inc.\VMware Player`,
}

// playerExecutable returns the path to a VMware Workstation Player executable.
func playerExecutable(executable string) (string, error) {
	path, err := exec.LookPath(executable + ".exe")
	if err == nil {
		return path, nil
	}
	return findFile(executable+".exe", playerProgramFilePaths()), nil
}

// playerFindVmplayer returns the path to the VMware Workstation Player executable.
func playerFindVmplayer() (string, error) {
	return playerExecutable(appPlayer)
}

// playerFindVmrun returns the path to the VMware VIX executable.
func playerFindVmrun() (string, error) {
	return playerExecutable(appVmrun)
}

// playerFindVdiskManager returns the path to the VMware Virtual Disk Manager
// executable.
func playerFindVdiskManager() (string, error) {
	return playerExecutable(appVdiskManager)
}

// playerFindQemuImg returns the path to the QEMU image utility.
func playerFindQemuImg() (string, error) {
	return playerExecutable(appQemuImg)
}

// playerToolsIsoPath returns the path to the VMware Tools ISO.
func playerToolsIsoPath(flavor string) string {
	return findFile(flavor+".iso", playerProgramFilePaths())
}

// playerInstallationPath returns the path to the VMware Workstation Player
// installation.
func playerDhcpLeasesPath(device string) string {
	// Not used on Windows.
	path, err := playerDhcpLeasesPathRegistry()
	if err != nil {
		log.Printf("[WARN] Unable to retrieve DHCP leases path from registry: %s", err)
	} else if _, err := os.Stat(path); err == nil {
		return path
	}
	return findFile(dhcpVmnetLeasesFile, playerDataFilePaths())
}

// playerDhcpConfPath returns the path to the DHCP configuration file.
func playerDhcpConfPath(device string) string {
	// Not used on Windows.
	path, err := playerDhcpConfPathRegistry()
	if err != nil {
		log.Printf("[WARN] Unable to retrieve DHCP configuration path from registry: %s", err)
	} else if _, err := os.Stat(path); err == nil {
		return path
	}
	return findFile(dhcpVmnetConfFile, playerDataFilePaths())
}

// playerNatConfPath returns the path to the NAT configuration file.
func playerNatConfPath(device string) string {
	// Not used on Windows.
	return findFile(natVmnetConfFile, playerDataFilePaths())
}

// playerNetmapConfPath returns the path to the network mapping configuration
// file.
func playerNetmapConfPath() string {
	return findFile(netmapConfFile, playerDataFilePaths())
}

// playerReadRegistryPath reads a registry key and subkey and normalizes the
// path.
func playerReadRegistryPath(key, subkey string) (string, error) {
	s, err := readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		return "", fmt.Errorf("unable to read registry key %s\\%s: %w", key, subkey, err)
	}
	return normalizePath(s), nil
}

// playerInstallationPath reads the installation path from the registry.
func playerInstallationPath() (string, error) {
	return playerReadRegistryPath(playerInstallationPathKey, "Path")
}

// playerDhcpLeasesPathRegistry reads the DHCP leases path from the registry.
func playerDhcpLeasesPathRegistry() (string, error) {
	return playerReadRegistryPath(playerDhcpRegistryKey, "LeaseFile")
}

// playerDhcpConfPathRegistry reads the DHCP configuration path from the
// registry.
func playerDhcpConfPathRegistry() (string, error) {
	return playerReadRegistryPath(playerDhcpRegistryKey, "ConfFile")
}

// playerAppendPath appends the path if the environment variable exists.
func playerAppendPath(paths []string, envVar, suffix string) []string {
	if value := os.Getenv(envVar); value != "" {
		paths = append(paths, filepath.Join(value, suffix))
	}
	return paths
}

// playerProgramFilesPaths returns a list of paths that are eligible
// to contain the VMware Workstation Player binaries.
func playerProgramFilePaths() []string {
	path, err := playerInstallationPath()
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

	paths = playerAppendPath(paths, "ProgramFiles(x86)", "VMware/VMware Player")
	paths = playerAppendPath(paths, "ProgramFiles", "VMware/VMware Player")
	paths = playerAppendPath(paths, "QEMU_HOME", "")
	paths = playerAppendPath(paths, "ProgramFiles(x86)", "QEMU")
	paths = playerAppendPath(paths, "ProgramFiles", "QEMU")
	paths = playerAppendPath(paths, "SystemDrive", "QEMU")

	return paths
}

// playerDataFilePaths returns a list of paths that are eligible
// to contain configuration files.
func playerDataFilePaths() []string {
	leasesPath, err := playerDhcpLeasesPathRegistry()
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

	paths = playerAppendPath(paths, "ProgramData", "VMware")
	paths = playerAppendPath(paths, "ALLUSERSPROFILE", "Application Data/VMware")

	return paths
}

// playerVerifyVersion verifies the VMware Workstation Player version against
// the required version.
func playerVerifyVersion(requiredVersion string) error {
	productVersion, err := playerGetVersionFromRegistry()
	if err != nil {
		return err
	}

	return playerTestVersion(requiredVersion, productVersion)
}

// playerGetVersionFromRegistry retrieves the VMware Workstation Player version
// from the Windows registry.
func playerGetVersionFromRegistry() (string, error) {
	subkey := "ProductVersion"
	for _, path := range playerInstallationRegistryPaths {
		productVersion, err := readRegString(syscall.HKEY_LOCAL_MACHINE, path, subkey)
		if err == nil {
			return productVersion, nil
		}
		log.Printf(`[WARN] Unable to read registry key %s\%s`, path, subkey)
	}
	return "", fmt.Errorf("unable to read any valid registry key for VMware Player")
}

// playerTestVersion checks if the product version matches the required version.
func playerTestVersion(requiredVersion, productVersion string) error {
	versionRe := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := versionRe.FindStringSubmatch(productVersion)
	if matches == nil || len(matches) < 4 {
		return fmt.Errorf("error parsing product version: '%s'", productVersion)
	}

	fullVersion := fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
	log.Printf("[INFO] %s: %s", playerProductName, fullVersion)

	parsedVersionFound, err := version.NewVersion(fullVersion)
	if err != nil {
		return fmt.Errorf("invalid version found: %w", err)
	}

	parsedVersionRequired, err := version.NewVersion(requiredVersion)
	if err != nil {
		return fmt.Errorf("invalid version required: %w", err)
	}

	return compareVersionObjects(parsedVersionFound, parsedVersionRequired, playerProductName)
}
