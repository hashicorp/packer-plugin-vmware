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

// VMware Workstation Player for Windows.

func playerExecutable(executable string) (string, error) {
	path, err := exec.LookPath(executable + ".exe")
	if err == nil {
		return path, nil
	}
	return findFile(executable+".exe", playerProgramFilePaths()), nil
}

func playerFindVmplayer() (string, error) {
	return playerExecutable(appPlayer)
}

func playerFindVmrun() (string, error) {
	return playerExecutable(appVmrun)
}

func playerFindVdiskManager() (string, error) {
	return playerExecutable(appVdiskManager)
}

func playerFindQemuImg() (string, error) {
	return playerExecutable(appQemuImg)
}

func playerToolsIsoPath(flavor string) string {
	return findFile(flavor+".iso", playerProgramFilePaths())
}

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

func playerVmDhcpConfPath(device string) string {
	// Not used on Windows.
	path, err := playerDhcpConfigPathRegistry()
	if err != nil {
		log.Printf("[WARN] Unable to retrieve DHCP configuration path from registry: %s", err)
	} else if _, err := os.Stat(path); err == nil {
		return path
	}
	return findFile(dhcpVmnetConfFile, playerDataFilePaths())
}

func playerVmnetnatConfPath(device string) string {
	// Not used on Windows.
	return findFile(natVmnetConfFile, playerDataFilePaths())
}

func playerNetmapConfPath() string {
	return findFile(netmapConfFile, playerDataFilePaths())
}

func playerReadRegistryPath(key, subkey string) (string, error) {
	s, err := readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		return "", fmt.Errorf("unable to read registry key %s\\%s: %w", key, subkey, err)
	}
	return normalizePath(s), nil
}

// Read the installation path from the registry.
func playerInstallationPath() (string, error) {
	return playerReadRegistryPath(playerInstallationPathKey, "Path")
}

// Read the DHCP leases path from the registry.
func playerDhcpLeasesPathRegistry() (string, error) {
	return playerReadRegistryPath(playerRegistryKey, "LeaseFile")
}

// Read the DHCP configuration path from the registry.
func playerDhcpConfigPathRegistry() (string, error) {
	return playerReadRegistryPath(playerRegistryKey, "ConfFile")
}

// Append path if the environment variable exists.
func appendPlayerPath(paths []string, envVar, suffix string) []string {
	if value := os.Getenv(envVar); value != "" {
		paths = append(paths, filepath.Join(value, suffix))
	}
	return paths
}

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

	paths = appendPlayerPath(paths, "ProgramFiles(x86)", "VMware/VMware Player")
	paths = appendPlayerPath(paths, "ProgramFiles", "VMware/VMware Player")
	paths = appendPlayerPath(paths, "QEMU_HOME", "")
	paths = appendPlayerPath(paths, "ProgramFiles(x86)", "QEMU")
	paths = appendPlayerPath(paths, "ProgramFiles", "QEMU")
	paths = appendPlayerPath(paths, "SystemDrive", "QEMU")

	return paths
}

// Read a list of possible data paths.
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

	paths = appendPlayerPath(paths, "ProgramData", "VMware")
	paths = appendPlayerPath(paths, "ALLUSERSPROFILE", "Application Data/VMware")

	return paths
}

func playerVerifyVersion(requiredVersion string) error {
	key := `SOFTWARE\Wow6432Node\VMware, Inc.\VMware Player`
	subkey := "ProductVersion"
	productVersion, err := readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		log.Printf(`[WARN] Unable to read registry key %s\%s`, key, subkey)
		key = `SOFTWARE\VMware, Inc.\VMware Player`
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
