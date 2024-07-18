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

// PlayerDriver is a driver for VMware Workstation Player.
type PlayerDriver struct {
	VmwareDriver

	AppPath          string
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

func (d *PlayerDriver) Clone(dst, src string, linked bool, snapshot string) error {

	var cloneType string
	if linked {
		cloneType = "linked"
	} else {
		cloneType = "full"
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
	guiArgument := "gui"
	if headless {
		guiArgument = "nogui"
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
		return err
	}

	return nil
}

func (d *PlayerDriver) SuppressMessages(vmxPath string) error {
	return nil
}

func (d *PlayerDriver) Verify() error {
	var err error
	if d.AppPath == "" {
		if d.AppPath, err = playerFindVMware(); err != nil {
			return err
		}
	}

	if d.VmrunPath == "" {
		if d.VmrunPath, err = playerFindVmrun(); err != nil {
			return err
		}
	}

	if d.VdiskManagerPath == "" {
		d.VdiskManagerPath, err = playerFindVdiskManager()
	}

	if d.VdiskManagerPath == "" && d.QemuImgPath == "" {
		d.QemuImgPath, err = playerFindQemuImg()
	}

	if err != nil {
		return fmt.Errorf("error finding either 'vmware-vdiskmanager' or 'qemu-img' in path")
	}

	log.Printf("VMware app path: %s", d.AppPath)
	log.Printf("vmrun path: %s", d.VmrunPath)
	log.Printf("vdisk-manager path: %s", d.VdiskManagerPath)
	log.Printf("qemu-img path: %s", d.QemuImgPath)

	if _, err := os.Stat(d.AppPath); err != nil {
		return fmt.Errorf("player not found in path: %s", d.AppPath)
	}

	if _, err := os.Stat(d.VmrunPath); err != nil {
		return fmt.Errorf("'vmrun' not found in path: %s", d.VmrunPath)
	}

	if d.VdiskManagerPath != "" {
		_, err = os.Stat(d.VdiskManagerPath)
	} else {
		_, err = os.Stat(d.QemuImgPath)
	}

	if err != nil {
		return fmt.Errorf("error finding either 'vmware-vdiskmanager' or 'qemu-img' in path: %s", d.VdiskManagerPath)
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

		// If we were able to find the file (no error), then we can proceed with reading
		// the networkmapper configuration.
		if _, err := os.Stat(pathNetmap); err == nil {
			log.Printf("Located networkmapper configuration file: %s", pathNetmap)
			return ReadNetmapConfig(pathNetmap)
		}

		// If we weren't able to find the networkmapper configuration file, then fall back
		// to the networking file which might also be in the configuration directory.
		libpath, _ := playerVMwareRoot()
		pathNetworking := filepath.Join(libpath, "networking")
		if _, err := os.Stat(pathNetworking); err != nil {
			return nil, fmt.Errorf("error determining network mappings from files in path: %s", libpath)
		}

		// We were able to successfully stat the file.. So, now we can open a handle to it.
		log.Printf("Located networking configuration file: %s", pathNetworking)
		fd, err := os.Open(pathNetworking)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		// Then we pass the handle to the networking configuration parser.
		return ReadNetworkingConfig(fd)
	}
	return nil
}

func (d *PlayerDriver) ToolsIsoPath(flavor string) string {
	return playerToolsIsoPath(flavor)
}

func (d *PlayerDriver) ToolsInstall() error {
	return nil
}

func (d *PlayerDriver) GetVmwareDriver() VmwareDriver {
	return d.VmwareDriver
}
