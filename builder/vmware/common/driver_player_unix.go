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
	return "/usr/lib/vmware/isoimages/" + flavor + ".iso"
}

// playerInstallationPath returns the path to the installation path.
func playerInstallationPath() (s string, err error) {
	return "/etc/vmware", nil
}

// playerFindConfigPath finds the configuration file in the device path.
func playerFindConfigPath(device string, paths []string) string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("[WARN] Error finding %s installation path: %s", playerProductName, err)
		return ""
	}

	devicebase := filepath.Join(base, device)
	for _, p := range paths {
		fp := filepath.Join(devicebase, p)
		if _, err := os.Stat(fp); !os.IsNotExist(err) {
			return fp
		}
	}

	log.Printf("[WARN] Error finding configuration file in device path: %s", devicebase)
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
		log.Printf("[WARN] Error finding the NAT configuration file path: %s", err)
		return ""
	}
	return filepath.Join(base, device, netmapConfFile)
}

// playerNetmapConfPath returns the path to the network mapping configuration
// file.
func playerNetmapConfPath() string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("[WARN] Error finding the network mapping configuration file path: %s", err)
		return ""
	}
	return filepath.Join(base, netmapConfFile)
}

// playerVerifyVersion verifies the VMware Workstation Player version
// against the required version.
func playerVerifyVersion(requiredVersion string) error {
	if runtime.GOOS != osLinux {
		return fmt.Errorf("driver is only supported on Linux, not %s", runtime.GOOS)
	}

	// Using the default.
	vmxpath := "/usr/lib/vmware/bin/" + appVmx

	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	versionRe := regexp.MustCompile(`(?i)VMware Player (\d+)\.(\d+)\.(\d+)`)
	matches := versionRe.FindStringSubmatch(stderr.String())
	if matches == nil {
		return fmt.Errorf("error parsing version from output: %s", stderr.String())
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
