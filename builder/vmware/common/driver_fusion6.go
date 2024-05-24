// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	"strings"
)

const VMWARE_FUSION_VERSION = "6"

// Fusion6Driver is a driver that can run VMware Fusion 6.
type Fusion6Driver struct {
	Fusion5Driver
}

func NewFusion6Driver(dconfig *DriverConfig, config *SSHConfig) Driver {
	return &Fusion6Driver{
		Fusion5Driver: Fusion5Driver{
			AppPath:   dconfig.FusionAppPath,
			SSHConfig: config,
		},
	}
}

func (d *Fusion6Driver) Clone(dst, src string, linked bool, snapshot string) error {

	var cloneType string
	if linked {
		cloneType = "linked"
	} else {
		cloneType = "full"
	}

	args := []string{"-T", "fusion", "clone", src, dst, cloneType}
	if snapshot != "" {
		args = append(args, "-snapshot", snapshot)
	}
	cmd := exec.Command(d.vmrunPath(), args...)
	if _, _, err := runAndLog(cmd); err != nil {
		if strings.Contains(err.Error(), "parameters was invalid") {
			return fmt.Errorf("linked clones are not supported on this version")
		}

		return err
	}

	return nil
}

func (d *Fusion6Driver) Verify() error {
	if err := d.Fusion5Driver.Verify(); err != nil {
		return err
	}

	vmxpath := filepath.Join(d.AppPath, "Contents", "Library", "vmware-vmx")
	if _, err := os.Stat(vmxpath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("'vmware-vmx' not found in path: %s", vmxpath)
		}

		return err
	}

	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Example: VMware Fusion e.x.p build-6048684 Release
	techPreviewRe := regexp.MustCompile(`(?i)VMware [a-z0-9-]+ e\.x\.p `)
	matches := techPreviewRe.FindStringSubmatch(stderr.String())
	if matches != nil {
		log.Printf("Detected VMware version: e.x.p (Tech Preview)")
		return nil
	}

	// Example: VMware Fusion 7.1.3 build-3204469 Release
	versionRe := regexp.MustCompile(`(?i)VMware [a-z0-9-]+ (\d+)\.`)
	matches = versionRe.FindStringSubmatch(stderr.String())
	if matches == nil {
		return fmt.Errorf("error parsing version output: %s", stderr.String())
	}
	log.Printf("Detected VMware version: %s", matches[1])

	libpath := filepath.Join("/", "Library", "Preferences", "VMware Fusion")

	d.VmwareDriver.DhcpLeasesPath = func(device string) string {
		return "/var/db/vmware/vmnet-dhcpd-" + device + ".leases"
	}
	d.VmwareDriver.DhcpConfPath = func(device string) string {
		return filepath.Join(libpath, device, "dhcpd.conf")
	}

	d.VmwareDriver.VmnetnatConfPath = func(device string) string {
		return filepath.Join(libpath, device, "nat.conf")
	}
	d.VmwareDriver.NetworkMapper = func() (NetworkNameMapper, error) {
		pathNetworking := filepath.Join(libpath, "networking")
		if _, err := os.Stat(pathNetworking); err != nil {
			return nil, fmt.Errorf("error finding networking configuration file: %s", pathNetworking)
		}
		log.Printf("Located networkmapper configuration file: %s", pathNetworking)

		fd, err := os.Open(pathNetworking)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		return ReadNetworkingConfig(fd)
	}

	return compareVersions(matches[1], VMWARE_FUSION_VERSION, "Fusion Professional")
}

func (d *Fusion6Driver) ToolsIsoPath(k string) string {
	// Fusion 13.x.x changes tools iso location
	vmxpath := filepath.Join(d.AppPath, "Contents", "Library", "vmware-vmx")
	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[DEBUG] failed to execute vmware-vmx command to get version %v", err)
		log.Printf("[DEBUG] continuing with default iso path for fusion6+.")
		return filepath.Join(d.AppPath, "Contents", "Library", "isoimages", "x86_x64", k+".iso")
	}
	versionRe := regexp.MustCompile(`(?i)VMware [a-z0-9-]+ (\d+)\.`)
	matches := versionRe.FindStringSubmatch(stderr.String())
	if len(matches) > 0 && (matches[1] < "13") {
		return filepath.Join(d.AppPath, "Contents", "Library", "isoimages", k+".iso")
	}
	if k == "windows" && runtime.GOARCH == "arm64" {
		return filepath.Join(d.AppPath, "Contents", "Library", "isoimages", "arm64", k+".iso")
	}

	return filepath.Join(d.AppPath, "Contents", "Library", "isoimages", "x86_x64", k+".iso")
}

func (d *Fusion6Driver) GetVmwareDriver() VmwareDriver {
	return d.Fusion5Driver.VmwareDriver
}
