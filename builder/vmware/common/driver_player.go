// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

// VMware Workstation Player

// PlayerDriver is a driver for VMware Workstation Player.
type PlayerDriver struct {
	VmwareDriver

	// The path to VMware Workstation Player.
	AppPath string

	VdiskManagerPath string
	QemuImgPath      string
	VmrunPath        string

	SSHConfig *SSHConfig
}

// NewPlayerDriver creates a new PlayerDriver.
func NewPlayerDriver(config *SSHConfig) Driver {
	return &PlayerDriver{
		SSHConfig: config,
	}
}

// GetVmwareDriver returns the VmwareDriver.
func (d *PlayerDriver) GetVmwareDriver() VmwareDriver {
	return d.VmwareDriver
}

// Clone clones a virtual machine.
func (d *PlayerDriver) Clone(dst, src string, linked bool, snapshot string) error {
	var cloneType string

	if linked {
		cloneType = cloneTypeLinked
	} else {
		cloneType = cloneTypeFull
	}

	args := []string{"-T", "ws", "clone", src, dst, cloneType}
	if snapshot != "" {
		args = append(args, "-snapshot", snapshot)
	}
	cmd := exec.Command(d.VmrunPath, args...)
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

// CompactDisk compacts a virtual machine disk based on the disk path.
func (d *PlayerDriver) CompactDisk(diskPath string) error {
	if d.QemuImgPath != "" {
		return d.qemuCompactDisk(diskPath)
	}

	defragCmd := exec.Command(d.VdiskManagerPath, "-d", diskPath)
	if _, _, err := runAndLog(defragCmd); err != nil {
		return err
	}

	shrinkCmd := exec.Command(d.VdiskManagerPath, "-k", diskPath)
	if _, _, err := runAndLog(shrinkCmd); err != nil {
		return err
	}

	return nil
}

// qemuCompactDisk compacts a virtual machine disk using qemu-img.
func (d *PlayerDriver) qemuCompactDisk(diskPath string) error {
	cmd := exec.Command(d.QemuImgPath, "convert", "-f", "vmdk", "-O", "vmdk", "-o", "compat6", diskPath, diskPath+".new")
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	if err := os.Remove(diskPath); err != nil {
		return err
	}

	if err := os.Rename(diskPath+".new", diskPath); err != nil {
		return err
	}

	return nil
}

// CreateDisk creates a virtual machine disk based on the output path, size,
// adapter type, and type ID.
func (d *PlayerDriver) CreateDisk(output string, size string, adapter_type string, type_id string) error {
	var cmd *exec.Cmd
	if d.QemuImgPath != "" {
		cmd = exec.Command(d.QemuImgPath, "create", "-f", "vmdk", "-o", "compat6", output, size)
	} else {
		cmd = exec.Command(d.VdiskManagerPath, "-c", "-s", size, "-a", adapter_type, "-t", type_id, output)
	}
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

// CreateSnapshot creates a snapshot of a virtual machine based on the .vmx
// file path and snapshot name.
func (d *PlayerDriver) CreateSnapshot(vmxPath string, snapshotName string) error {
	cmd := exec.Command(d.VmrunPath, "-T", "player", "snapshot", vmxPath, snapshotName)
	_, _, err := runAndLog(cmd)
	return err
}

// IsRunning checks if a virtual machine is running based on the .vmx file path.
func (d *PlayerDriver) IsRunning(vmxPath string) (bool, error) {
	vmxPath, err := filepath.Abs(vmxPath)
	if err != nil {
		return false, err
	}

	cmd := exec.Command(d.VmrunPath, "-T", "player", "list")
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

// CommHost returns the host address based on the SSH configuration.
func (d *PlayerDriver) CommHost(state multistep.StateBag) (string, error) {
	return CommHost(d.SSHConfig)(state)
}

// Start powers on a virtual machine based on the .vmx file path and mode
// (headless or GUI).
func (d *PlayerDriver) Start(vmxPath string, headless bool) error {
	guiArgument := guiArgumentNoGUI
	if !headless {
		guiArgument = guiArgumentGUI
	}

	cmd := exec.Command(d.VmrunPath, "-T", "player", "start", vmxPath, guiArgument)
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

// Stop powers off a virtual machine based on the .vmx file path.
func (d *PlayerDriver) Stop(vmxPath string) error {
	cmd := exec.Command(d.VmrunPath, "-T", "player", "stop", vmxPath, "hard")
	if _, _, err := runAndLog(cmd); err != nil {
		// Check if the virtual machine is running. If not, it is stopped.
		running, runningErr := d.IsRunning(vmxPath)
		if runningErr == nil && !running {
			return nil
		}

		return err
	}

	return nil
}

// SuppressMessages suppresses messages for a virtual machine based on the .vmx
// file path.
func (d *PlayerDriver) SuppressMessages(vmxPath string) error {
	return nil
}

// Verify checks if the VMware Workstation Player installation is valid.
func (d *PlayerDriver) Verify() error {
	var err error

	log.Printf("[INFO] Searching for %s...", playerProductName)

	if err := playerVerifyVersion(playerMinVersionObj.String()); err != nil {
		return fmt.Errorf("version verification failed: %s", err)
	}

	components := map[string]*string{
		appPlayer: &d.AppPath,
		appVmrun:  &d.VmrunPath,
	}

	for name, path := range components {
		if *path == "" {
			var finderFunc func() (string, error)
			switch name {
			case appPlayer:
				finderFunc = playerFindVmplayer
			case appVmrun:
				finderFunc = playerFindVmrun
			default:
				return fmt.Errorf("unknown component: %s", name)
			}

			if foundPath, err := finderFunc(); err != nil {
				return fmt.Errorf("%s not found: %s", name, err)
			} else {
				*path = foundPath
				log.Printf("[INFO] - %s found at: %s", name, *path)
			}
		}
	}

	if d.VdiskManagerPath == "" {
		d.VdiskManagerPath, err = playerFindVdiskManager()
	}

	if d.VdiskManagerPath == "" && d.QemuImgPath == "" {
		d.QemuImgPath, err = playerFindQemuImg()
	}

	if err != nil {
		return fmt.Errorf("error finding either %s or %s: %s", appVdiskManager, appQemuImg, err)
	}

	for name, path := range map[string]string{
		appVdiskManager: d.VdiskManagerPath,
		appQemuImg:      d.QemuImgPath,
	} {
		if path != "" {
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("%s not found at: %s", name, path)
				}
				return err
			}
			log.Printf("[INFO] - %s found at: %s", name, path)
		}
	}

	d.DhcpLeasesPath = func(device string) string {
		return playerDhcpLeasesPath(device)
	}

	d.DhcpConfPath = func(device string) string {
		return playerDhcpConfPath(device)
	}

	d.VmnetnatConfPath = func(device string) string {
		return playerNatConfPath(device)
	}

	d.NetworkMapper = func() (NetworkNameMapper, error) {
		pathNetmap := playerNetmapConfPath()

		if _, err := os.Stat(pathNetmap); err == nil {
			log.Printf("[INFO] Found: %s", pathNetmap)
			return ReadNetmapConfig(pathNetmap)
		}

		libpath, _ := playerInstallationPath()
		pathNetworking := filepath.Join(libpath, "networking")
		if _, err := os.Stat(pathNetworking); err != nil {
			return nil, fmt.Errorf("not found: %s", pathNetworking)
		}

		log.Printf("[INFO] Found: %s", pathNetworking)
		fd, err := os.Open(pathNetworking)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		return ReadNetworkingConfig(fd)
	}

	return nil
}

// ToolsIsoPath returns the path to the VMware Tools ISO based on the flavor.
func (d *PlayerDriver) ToolsIsoPath(flavor string) string {
	return playerToolsIsoPath(flavor)
}

// ToolsInstall installs VMware Tools.
func (d *PlayerDriver) ToolsInstall() error {
	return nil
}
