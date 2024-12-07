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

// VMware Workstation on Linux

// workstationCheckLicense checks if a VMware Workstation license is present.
func workstationCheckLicense() error {
	matches, err := filepath.Glob("/etc/vmware/license-ws-*")
	if err != nil {
		return fmt.Errorf("error finding license: %s", err)
	}

	if len(matches) == 0 {
		return errors.New("no license found")
	}

	return nil
}

// workstationFindVmrun returns the path to the VMware VIX executable.
func workstationFindVmrun() (string, error) {
	return exec.LookPath(appVmrun)
}

// workstationFindVdiskManager returns the path to the VMware Virtual Disk
// Manager executable.
func workstationFindVdiskManager() (string, error) {
	return exec.LookPath(appVdiskManager)
}

// workstationFindVMware returns the path to the VMware Workstation executable.
func workstationFindVMware() (string, error) {
	return exec.LookPath(appVmware)
}

// workstationToolsIsoPath returns the path to the VMware Tools ISO.
func workstationToolsIsoPath(flavor string) string {
	return "/usr/lib/vmware/isoimages/" + flavor + ".iso"
}

// workstationInstallationPath reads the installation path.
func workstationInstallationPath() (s string, err error) {
	return "/etc/vmware", nil
}

// workstationFindConfigPath finds the configuration file in the device path.
func workstationFindConfigPath(device string, paths []string) string {
	base, err := workstationInstallationPath()
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

// workstationDhcpLeasesPath returns the path to the DHCP leases file.
func workstationDhcpLeasesPath(device string) string {
	return workstationFindConfigPath(device, GetDhcpLeasesPaths())
}

// workstationDhcpConfPath returns the path to the DHCP configuration file.
func workstationDhcpConfPath(device string) string {
	return workstationFindConfigPath(device, GetDhcpConfPaths())
}

// workstationNatConfPath returns the path to the NAT configuration file.
func workstationNatConfPath(device string) string {
	base, err := workstationInstallationPath()
	if err != nil {
		log.Printf("[WARN] Error finding the network mapping configuration path: %s", err)
		return ""
	}
	return filepath.Join(base, device, netmapConfFile)
}

// workstationNetmapConfPath returns the path to the network mapping
// configuration file.
func workstationNetmapConfPath() string {
	base, err := workstationInstallationPath()
	if err != nil {
		log.Printf("[WARN] Error finding the network mapping configuration file path: %s", err)
		return ""
	}
	return filepath.Join(base, netmapConfFile)
}

// workstationVerifyVersion verifies the VMware Workstation version against the
// required version using workstationTestVersion.
func workstationVerifyVersion(version string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("driver is only supported on Linux, not %s", runtime.GOOS)
	}

	vmxpath := "/usr/lib/vmware/bin/vmware-vmx"

	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return workstationTestVersion(version, stderr.String())
}

// workstationTestVersion verifies the VMware Workstation version against the
// required version.
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
