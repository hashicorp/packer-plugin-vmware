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

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

// TODO: Update the to VMware Fusion 13 per the Broadcom Product Lifecycle.
const minimumFusionVersion = "6"

const fusionSuppressPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>disallowUpgrade</key>
	<true/>
</dict>
</plist>`

// FusionDriver is a driver that can run VMware Fusion 6.
type FusionDriver struct {
	VmwareDriver

	// This is the path to the "VMware Fusion.app"
	AppPath string

	// SSHConfig are the SSH settings for the Fusion VM
	SSHConfig *SSHConfig
}

func NewFusionDriver(dconfig *DriverConfig, config *SSHConfig) Driver {
	return &FusionDriver{
		AppPath:   dconfig.FusionAppPath,
		SSHConfig: config,
	}
}

func (d *FusionDriver) CompactDisk(diskPath string) error {
	defragCmd := exec.Command(d.vdiskManagerPath(), "-d", diskPath)
	if _, _, err := runAndLog(defragCmd); err != nil {
		return err
	}

	shrinkCmd := exec.Command(d.vdiskManagerPath(), "-k", diskPath)
	if _, _, err := runAndLog(shrinkCmd); err != nil {
		return err
	}

	return nil
}

func (d *FusionDriver) CreateDisk(output string, size string, adapter_type string, type_id string) error {
	cmd := exec.Command(d.vdiskManagerPath(), "-c", "-s", size, "-a", adapter_type, "-t", type_id, output)
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

func (d *FusionDriver) CreateSnapshot(vmxPath string, snapshotName string) error {
	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "snapshot", vmxPath, snapshotName)
	_, _, err := runAndLog(cmd)
	return err
}

func (d *FusionDriver) IsRunning(vmxPath string) (bool, error) {
	vmxPath, err := filepath.Abs(vmxPath)
	if err != nil {
		return false, err
	}

	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "list")
	stdout, _, err := runAndLog(cmd)
	if err != nil {
		return false, err
	}

	for _, line := range strings.Split(stdout, "\n") {
		if line == vmxPath {
			return true, nil
		}
	}

	return false, nil
}

func (d *FusionDriver) CommHost(state multistep.StateBag) (string, error) {
	return CommHost(d.SSHConfig)(state)
}

func (d *FusionDriver) Start(vmxPath string, headless bool) error {
	guiArgument := "gui"
	if headless {
		guiArgument = "nogui"
	}

	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "start", vmxPath, guiArgument)
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

func (d *FusionDriver) Stop(vmxPath string) error {
	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "stop", vmxPath, "hard")
	if _, _, err := runAndLog(cmd); err != nil {
		// Check if the VM is running. If its not, it was already stopped
		running, rerr := d.IsRunning(vmxPath)
		if rerr == nil && !running {
			return nil
		}

		return err
	}

	return nil
}

func (d *FusionDriver) SuppressMessages(vmxPath string) error {
	dir := filepath.Dir(vmxPath)
	base := filepath.Base(vmxPath)
	base = strings.Replace(base, ".vmx", "", -1)

	plistPath := filepath.Join(dir, base+".plist")
	return os.WriteFile(plistPath, []byte(fusionSuppressPlist), 0644)
}

func (d *FusionDriver) vdiskManagerPath() string {
	return filepath.Join(d.AppPath, "Contents", "Library", "vmware-vdiskmanager")
}

func (d *FusionDriver) vmrunPath() string {
	return filepath.Join(d.AppPath, "Contents", "Library", "vmrun")
}

func (d *FusionDriver) ToolsInstall() error {
	return nil
}

func (d *FusionDriver) Clone(dst, src string, linked bool, snapshot string) error {

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

		return err
	}

	return nil
}

func (d *FusionDriver) Verify() error {

	if _, err := os.Stat(d.AppPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Fusion application not found at path: %s", d.AppPath)
		}

		return err
	}

	vmxpath := filepath.Join(d.AppPath, "Contents", "Library", "vmware-vmx")
	if _, err := os.Stat(vmxpath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vmware-vmx could not be found at path: %s",
				vmxpath)
		}

		return err
	}

	if _, err := os.Stat(d.vmrunPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"Critical application 'vmrun' not found at path: %s", d.vmrunPath())

		}

		return err
	}

	if _, err := os.Stat(d.vdiskManagerPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"Critical application vdisk manager not found at path: %s",
				d.vdiskManagerPath())
		}

		return err
	}

	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}

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
		return fmt.Errorf(
			"Couldn't find VMware version in output: %s", stderr.String())
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
			return nil, fmt.Errorf("Could not find networking conf file: %s", pathNetworking)
		}
		log.Printf("Located networkmapper configuration file using Fusion6: %s", pathNetworking)

		fd, err := os.Open(pathNetworking)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		return ReadNetworkingConfig(fd)
	}

	return compareVersions(matches[1], minimumFusionVersion, "Fusion Professional")
}

func (d *FusionDriver) ToolsIsoPath(k string) string {
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

func (d *FusionDriver) GetVmwareDriver() VmwareDriver {
	return d.VmwareDriver
}
