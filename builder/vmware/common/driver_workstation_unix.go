// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package common

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/hashicorp/go-version"
)

func workstationCheckLicense() error {

	err := workstationVerifyVersion(workstationNoLicenseVersion)
	if err == nil {
		// Reference: Free for commercial, educational, and personal use.
		log.Printf("[INFO] Skipping license check for version >= %s", workstationNoLicenseVersion)
		return nil
	}

	var errLicenseRequired = errors.New("installed version requires a license")

	if errors.Is(err, errLicenseRequired) {
		log.Printf("[INFO] Version is lower than %s. Performing license check.", workstationNoLicenseVersion)

		matches, err := filepath.Glob("/etc/vmware/license-ws-*")
		if err != nil {
			return fmt.Errorf("error finding license file: %w", err)
		}
		if len(matches) == 0 {
			return errors.New("no license found")
		}

		log.Printf("[INFO] License found: %v", matches)
		return nil
	}

	return err
}

func workstationFindVdiskManager() (string, error) {
	return exec.LookPath("vmware-vdiskmanager")
}

func workstationFindVMware() (string, error) {
	return exec.LookPath("vmware")
}

func workstationFindVmrun() (string, error) {
	return exec.LookPath("vmrun")
}

// return the base path to vmware's config on the host
func workstationVMwareRoot() (s string, err error) {
	return "/etc/vmware", nil
}

func workstationDhcpLeasesPath(device string) string {
	base, err := workstationVMwareRoot()
	if err != nil {
		log.Printf("Error finding VMware root: %s", err)
		return ""
	}

	// Build the base path to VMware configuration for specified device: `/etc/vmware/${device}`
	devicebase := filepath.Join(base, device)

	// Walk through a list of paths searching for the correct permutation...
	// ...as it appears that in >= WS14 and < WS14, the leases file may be labelled differently.

	// Docs say we should expect: dhcpd/dhcpd.leases
	paths := []string{"dhcpd/dhcpd.leases", "dhcpd/dhcp.leases", "dhcp/dhcpd.leases", "dhcp/dhcp.leases"}
	for _, p := range paths {
		fp := filepath.Join(devicebase, p)
		if _, err := os.Stat(fp); !os.IsNotExist(err) {
			return fp
		}
	}

	log.Printf("Error finding VMware DHCP Server Leases (dhcpd.leases) under device path: %s", devicebase)
	return ""
}

func workstationDhcpConfPath(device string) string {
	base, err := workstationVMwareRoot()
	if err != nil {
		log.Printf("Error finding VMware root: %s", err)
		return ""
	}

	// Build the base path to VMware configuration for specified device: `/etc/vmware/${device}`
	devicebase := filepath.Join(base, device)

	// Walk through a list of paths searching for the correct permutation...
	// ...as it appears that in >= WS14 and < WS14, the dhcp config may be labelled differently.

	// Docs say we should expect: dhcp/dhcp.conf
	paths := []string{"dhcp/dhcp.conf", "dhcp/dhcpd.conf", "dhcpd/dhcp.conf", "dhcpd/dhcpd.conf"}
	for _, p := range paths {
		fp := filepath.Join(devicebase, p)
		if _, err := os.Stat(fp); !os.IsNotExist(err) {
			return fp
		}
	}

	log.Printf("Error finding VMware DHCP Server Configuration (dhcp.conf) under device path: %s", devicebase)
	return ""
}

func workstationVmnetnatConfPath(device string) string {
	base, err := workstationVMwareRoot()
	if err != nil {
		log.Printf("Error finding VMware root: %s", err)
		return ""
	}
	return filepath.Join(base, device, "nat/nat.conf")
}

func workstationNetmapConfPath() string {
	base, err := workstationVMwareRoot()
	if err != nil {
		log.Printf("error finding vmware root: %s", err)
		return ""
	}
	return filepath.Join(base, "netmap.conf")
}

func workstationToolsIsoPath(flavor string) string {
	return "/usr/lib/vmware/isoimages/" + flavor + ".iso"
}

func workstationVerifyVersion(version string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("driver is only supported on Linux or Windows, not %s", runtime.GOOS)
	}

	//TODO(pmyjavec) there is a better way to find this, how?
	//the default will suffice for now.
	vmxpath := "/usr/lib/vmware/bin/vmware-vmx"

	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return workstationTestVersion(version, stderr.String())
}

func workstationTestVersion(requiredVersion, versionOutput string) error {
	versionRe := regexp.MustCompile(`(?i)VMware Workstation (\d+\.\d+\.\d+)`)
	matches := versionRe.FindStringSubmatch(versionOutput)
	if matches == nil {
		return fmt.Errorf("error parsing version output: %s", versionOutput)
	}
	fullVersion := matches[1]
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
