// Copyright IBM Corp. 2013, 2025
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

// These variables are defined to silence unused constant warnings.
// They reference the Windows-only registry constants that are not used in Unix environments.
var (
	_ = workstationInstallationPathKey
	_ = workstationDhcpRegistryKey
)

// workstationCheckLicense checks for the presence of a VMware Workstation
// license file.
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
	return filepath.Join(linuxIsosPath, flavor+".iso")
}

// workstationInstallationPath reads the installation path.
func workstationInstallationPath() (s string, err error) {
	return linuxDefaultPath, nil
}

// workstationFindConfigPath finds the configuration file in the device path.
func workstationFindConfigPath(device string, paths []string) string {
	base, err := workstationInstallationPath()
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
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, device, "nat/nat.conf")
}

// workstationNetmapConfPath returns the path to the network mapping
// configuration file.
func workstationNetmapConfPath() string {
	base, err := workstationInstallationPath()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, netmapConfFile)
}

// workstationVerifyVersion verifies the VMware Workstation version against the
// required version using workstationTestVersion.
func workstationVerifyVersion(version string) error {
	if runtime.GOOS != osLinux {
		return fmt.Errorf("driver is only supported on Linux, not %s", runtime.GOOS)
	}

	vmxPath := filepath.Join(linuxAppPath, appVmx)

	var stderr bytes.Buffer
	cmd := exec.Command(vmxPath, "-v")
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
