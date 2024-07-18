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
)

// VMware Workstation Player for Linux.

func playerFindVdiskManager() (string, error) {
	return exec.LookPath("vmware-vdiskmanager")
}

func playerFindQemuImg() (string, error) {
	return exec.LookPath("qemu-img")
}

func playerFindVMware() (string, error) {
	return exec.LookPath("vmplayer")
}

func playerFindVmrun() (string, error) {
	return exec.LookPath("vmrun")
}

func playerToolsIsoPath(flavor string) string {
	return "/usr/lib/vmware/isoimages/" + flavor + ".iso"
}

// Return the base path to configuration files.
func playerVMwareRoot() (s string, err error) {
	return "/etc/vmware", nil
}

func playerDhcpLeasesPath(device string) string {
	base, err := playerVMwareRoot()
	if err != nil {
		log.Printf("Error finding configuration root path: %s", err)
		return ""
	}

	// Build the base path to configuration for specified device:
	// `/etc/vmware/${device}`
	devicebase := filepath.Join(base, device)

	// Iterate through a list of paths searching for the correct permutation.
	// The default is dhcpd.leases, per the product documentation, but this
	// will check for a few variations.
	paths := []string{"dhcpd/dhcpd.leases", "dhcpd/dhcp.leases", "dhcp/dhcpd.leases", "dhcp/dhcp.leases"}
	for _, p := range paths {
		fp := filepath.Join(devicebase, p)
		if _, err := os.Stat(fp); !os.IsNotExist(err) {
			return fp
		}
	}

	log.Printf("Error finding 'dhcpd.leases' in device path: %s", devicebase)
	return ""
}

func playerVmDhcpConfPath(device string) string {
	base, err := playerVMwareRoot()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}

	// Build the base path to configuration for specified device:
	// `/etc/vmware/${device}`
	devicebase := filepath.Join(base, device)

	// Iterate through a list of paths searching for the correct permutation.
	// The default is dhcp/dhcp.conf, per the product documentation, but this
	// will check for a few variations.
	paths := []string{"dhcp/dhcp.conf", "dhcp/dhcpd.conf", "dhcpd/dhcp.conf", "dhcpd/dhcpd.conf"}
	for _, p := range paths {
		fp := filepath.Join(devicebase, p)
		if _, err := os.Stat(fp); !os.IsNotExist(err) {
			return fp
		}
	}

	log.Printf("Error finding 'dhcp.conf' in device path: %s", devicebase)
	return ""
}

func playerVmnetnatConfPath(device string) string {
	base, err := playerVMwareRoot()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, device, "nat/nat.conf")
}

func playerNetmapConfPath() string {
	base, err := playerVMwareRoot()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, "netmap.conf")
}

func playerVerifyVersion(version string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("driver is only supported on linux and windows, not %s", runtime.GOOS)
	}

	// Using the default.
	vmxpath := "/usr/lib/vmware/bin/vmware-vmx"

	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	versionRe := regexp.MustCompile(`(?i)VMware Player (\d+)\.`)
	matches := versionRe.FindStringSubmatch(stderr.String())
	if matches == nil {
		return fmt.Errorf("error parsing version output: %s", stderr.String())
	}
	log.Printf("Detected VMware Workstation Player version: %s", matches[1])

	return compareVersions(matches[1], version, "Player")
}
