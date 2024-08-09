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

// TODO: Update to best effort comply with the Broadcom Product Lifecycle.
const (
	minimumFusionVersion = "6"
	isoPathChangeVersion = "13"
	archAMD64            = "x86_x64"
	archARM64            = "arm64"
)

const fusionSuppressPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>disallowUpgrade</key>
    <true/>
</dict>
</plist>`

// FusionDriver is a driver for VMware Fusion for macOS.
type FusionDriver struct {
	VmwareDriver

	// This is the path to the "VMware Fusion.app"
	AppPath string

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
		// Check if the virtual machine is running. If not, it is stopped.
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

func (d *FusionDriver) libPath() string {
	return filepath.Join("/", "Library", "Preferences", "VMware Fusion")
}

func (d *FusionDriver) binaryPath(binaryName string) string {
	return filepath.Join(d.AppPath, "Contents", "Library", binaryName)
}

func (d *FusionDriver) toolsIsoPath(subdirs ...string) string {
	parts := append([]string{d.AppPath, "Contents", "Library", "isoimages"}, subdirs...)
	return filepath.Join(parts...)
}

func (d *FusionDriver) vmxPath() string {
	return d.binaryPath("vmware-vmx")
}

func (d *FusionDriver) vmrunPath() string {
	return d.binaryPath("vmrun")
}

func (d *FusionDriver) vdiskManagerPath() string {
	return d.binaryPath("vmware-vdiskmanager")
}

func (d *FusionDriver) isoFileName(base string) string {
	return base + ".iso"
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
			return fmt.Errorf("application not found at path: %s", d.AppPath)
		}

		return err
	}

	if _, err := os.Stat(d.vmxPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vmware-vmx not found at path: %s", d.vmxPath())
		}

		return err
	}

	if _, err := os.Stat(d.vmrunPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vmrun not found at path: %s", d.vmrunPath())
		}

		return err
	}

	if _, err := os.Stat(d.vdiskManagerPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vmware-vdiskmanager not found at path: %s", d.vdiskManagerPath())
		}

		return err
	}

	var stderr bytes.Buffer
	cmd := exec.Command(d.vmxPath(), "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	techPreviewRe := regexp.MustCompile(`(?i)VMware [a-z0-9-]+ e\.x\.p `)
	matches := techPreviewRe.FindStringSubmatch(stderr.String())
	if matches != nil {
		log.Printf("VMware Fusion: e.x.p (Tech Preview)")
		return nil
	}

	// Example: VMware Fusion 13.5.2 build-23775688 Release
	versionRe := regexp.MustCompile(`(?i)VMware [a-z0-9-]+ (\d+)\.`)
	matches = versionRe.FindStringSubmatch(stderr.String())
	if matches == nil {
		return fmt.Errorf("error parsing version from output: %s", stderr.String())
	}
	log.Printf("VMware Fusion: %s", matches[1])

	libpath := d.libPath()

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
			return nil, fmt.Errorf("unable to locate networking configuration file: %s", pathNetworking)
		}

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
	// VMware Fusion 13 changes the VMware Tools ISO location.
	var stderr bytes.Buffer
	cmd := exec.Command(d.vmxPath(), "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[WARN] Failed to return version: %v. Continuing with default path.", err)
		return d.toolsIsoPath(archAMD64, d.isoFileName(k))
	}

	versionRe := regexp.MustCompile(`(?i)VMware [a-z0-9-]+ (\d+)\.`)
	matches := versionRe.FindStringSubmatch(stderr.String())
	if len(matches) > 0 && (matches[1] < isoPathChangeVersion) {
		return d.toolsIsoPath(d.isoFileName(k))
	}

	if k == "windows" && runtime.GOARCH == archARM64 {
		return d.toolsIsoPath(archARM64, d.isoFileName(k))
	}

	return d.toolsIsoPath(archAMD64, d.isoFileName(k))
}

func (d *FusionDriver) GetVmwareDriver() VmwareDriver {
	return d.VmwareDriver
}
