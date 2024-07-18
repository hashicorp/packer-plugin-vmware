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

const (
	// VMware Workstation Player application name.
	playerProductName = "VMware Workstation Player"

	// VMware Workstation Player versions.
	// TODO: Update to best effort comply with the Broadcom Product Lifecycle.
	minimumPlayerVersion = "6.0.0"
)

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

func NewPlayerDriver(config *SSHConfig) Driver {
	return &PlayerDriver{
		SSHConfig: config,
	}
}

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

func (d *PlayerDriver) CreateSnapshot(vmxPath string, snapshotName string) error {
	cmd := exec.Command(d.VmrunPath, "-T", "player", "snapshot", vmxPath, snapshotName)
	_, _, err := runAndLog(cmd)
	return err
}

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

func (d *PlayerDriver) CommHost(state multistep.StateBag) (string, error) {
	return CommHost(d.SSHConfig)(state)
}

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

func (d *PlayerDriver) SuppressMessages(vmxPath string) error {
	return nil
}

func (d *PlayerDriver) Verify() error {
	var err error

	log.Printf("[INFO] Checking %s paths...", playerProductName)

	if d.AppPath == "" {
		if d.AppPath, err = playerFindVmplayer(); err != nil {
			return fmt.Errorf("%s not found: %s", playerProductName, err)
		}
	}
	log.Printf("[INFO] - %s app path: %s", playerProductName, d.AppPath)

	if d.VmrunPath == "" {
		if d.VmrunPath, err = playerFindVmrun(); err != nil {
			return fmt.Errorf("%s not found: %s", appVmrun, err)
		}
	}
	log.Printf("[INFO] - %s found at: %s", appVmrun, d.VmrunPath)

	if d.VdiskManagerPath == "" {
		d.VdiskManagerPath, err = playerFindVdiskManager()
	}

	if d.VdiskManagerPath == "" && d.QemuImgPath == "" {
		d.QemuImgPath, err = playerFindQemuImg()
	}

	if err != nil {
		return fmt.Errorf("error finding either %s or %s: %s", appVdiskManager, appQemuImg, err)
	}

	log.Printf("[INFO] - %s found at: %s", appVdiskManager, d.VdiskManagerPath)
	log.Printf("[INFO] - %s found at: %s", appQemuImg, d.QemuImgPath)

	if _, err := os.Stat(d.AppPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found at: %s", playerProductName, d.AppPath)
		}
		return err
	}
	log.Printf("[INFO] - %s found at: %s", playerProductName, d.AppPath)

	if _, err := os.Stat(d.VmrunPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found at: %s", appVmrun, d.VmrunPath)
		}
		return err
	}
	log.Printf("[INFO] - %s found at: %s", appVmrun, d.VmrunPath)

	if d.VdiskManagerPath != "" {
		if _, err := os.Stat(d.VdiskManagerPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s not found at: %s", appVdiskManager, d.VdiskManagerPath)
			}
			return err
		}
		log.Printf("[INFO] - %s found at: %s", appVdiskManager, d.VdiskManagerPath)
	} else {
		if _, err := os.Stat(d.QemuImgPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("%s not found at: %s", appQemuImg, d.QemuImgPath)
			}
			return err
		}
		log.Printf("[INFO] - %s found at: %s", appQemuImg, d.QemuImgPath)
	}

	// Assigning the path callbacks to VmwareDriver
	d.VmwareDriver.DhcpLeasesPath = func(device string) string {
		return playerDhcpLeasesPath(device)
	}

	d.VmwareDriver.DhcpConfPath = func(device string) string {
		return playerVmDhcpConfPath(device)
	}

	d.VmwareDriver.VmnetnatConfPath = func(device string) string {
		return playerVmnetnatConfPath(device)
	}

	d.VmwareDriver.NetworkMapper = func() (NetworkNameMapper, error) {
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

	return playerVerifyVersion(minimumPlayerVersion)
}

func (d *PlayerDriver) ToolsIsoPath(flavor string) string {
	return playerToolsIsoPath(flavor)
}

func (d *PlayerDriver) ToolsInstall() error {
	return nil
}

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

func (d *PlayerDriver) GetVmwareDriver() VmwareDriver {
	return d.VmwareDriver
}
