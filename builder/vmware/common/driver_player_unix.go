// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package common

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/hashicorp/go-version"
)

// VMware Workstation Player on Linux

// playerFindVmplayer returns the path to the VMware Workstation Player executable.
func playerFindVmplayer() (string, error) {
	return exec.LookPath(appPlayer)
}

// playerFindVmrun returns the path to the VMware VIX executable.
func playerFindVmrun() (string, error) {
	return exec.LookPath(appVmrun)
}

// playerFindVdiskManager returns the path to the VMware Virtual Disk Manager
// executable.
func playerFindVdiskManager() (string, error) {
	return exec.LookPath(appVdiskManager)
}

// playerFindQemuImg returns the path to the QEMU image utility.
func playerFindQemuImg() (string, error) {
	return exec.LookPath(appQemuImg)
}

// playerToolsIsoPath returns the path to the VMware Tools ISO.
func playerToolsIsoPath(flavor string) string {
	return filepath.Join(linuxIsosPath, flavor+".iso")
}

// playerInstallationPath returns the path to the installation path.
func playerInstallationPath() (s string, err error) {
	return linuxDefaultPath, nil
}

// playerFindConfigPath finds the configuration file in the device path.
func playerFindConfigPath(device string, paths []string) string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("Error finding configuration root path: %s", err)
		return ""
	}

	deviceBase := filepath.Join(base, device)
	for _, p := range paths {
		fp := filepath.Join(deviceBase, p)
		if _, err := os.Stat(fp); !os.IsNotExist(err) {
			return fp
		}
	}

	log.Printf("Error finding configuration file in device path: %s", deviceBase)
	return ""
}

// playerDhcpLeasesPath returns the path to the DHCP leases file.
func playerDhcpLeasesPath(device string) string {
	return playerFindConfigPath(device, GetDhcpLeasesPaths())
}

// playerDhcpConfPath returns the path to the DHCP configuration file.
func playerDhcpConfPath(device string) string {
	return playerFindConfigPath(device, GetDhcpConfPaths())
}

// playerNatConfPath returns the path to the NAT configuration file.
func playerNatConfPath(device string) string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, device, "nat/nat.conf")
}

// playerNetmapConfPath returns the path to the network mapping configuration
// file.
func playerNetmapConfPath() string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, netmapConfFile)
}

// playerVerifyVersion verifies the VMware Workstation Player version against the required version.
func playerVerifyVersion(requiredVersion string) error {
	if runtime.GOOS != osLinux {
		return fmt.Errorf("driver is only supported on linux and windows, not %s", runtime.GOOS)
	}

	vmxPath := filepath.Join(linuxAppPath, appVmx)

	var stderr bytes.Buffer
	cmd := exec.Command(vmxPath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return playerTestVersion(requiredVersion, stderr.String())
}

// playerTestVersion verifies the VMware Workstation Player version against the required version.
func playerTestVersion(requiredVersion, versionOutput string) error {
	// Define the version regex pattern.
	versionRe := regexp.MustCompile(`(?i)VMware Player (\d+\.\d+\.\d+)`)
	matches := versionRe.FindStringSubmatch(versionOutput)
	if matches == nil {
		return fmt.Errorf("error parsing version output: %s", versionOutput)
	}

	// Extract the full version string.
	fullVersion := matches[1]
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
