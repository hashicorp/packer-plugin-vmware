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
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

// VMware Fusion

// FusionDriver is a driver for VMware Fusion for macOS.
type FusionDriver struct {
	VmwareDriver

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
	cleanDiskPath := filepath.Clean(diskPath)
	absPath, err := filepath.Abs(cleanDiskPath)
	if err != nil {
		return err
	}

	defragCmd := exec.Command(d.vdiskManagerPath(), "-d", absPath) //nolint:gosec
	if _, _, err := runAndLog(defragCmd); err != nil {
		return err
	}

	shrinkCmd := exec.Command(d.vdiskManagerPath(), "-k", absPath) //nolint:gosec
	if _, _, err := runAndLog(shrinkCmd); err != nil {
		return err
	}

	return nil
}

func (d *FusionDriver) CreateDisk(output string, size string, adapterType string, typeId string) error {
	cleanOutput := filepath.Clean(output)
	absOutput, err := filepath.Abs(cleanOutput)
	if err != nil {
		return err
	}

	cmd := exec.Command(d.vdiskManagerPath(), "-c", "-s", size, "-a", adapterType, "-t", typeId, absOutput) //nolint:gosec
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

func (d *FusionDriver) CreateSnapshot(vmxPath string, snapshotName string) error {
	cleanVmx := filepath.Clean(vmxPath)
	absVmxPath, err := filepath.Abs(cleanVmx)
	if err != nil {
		return err
	}

	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "snapshot", absVmxPath, snapshotName) //nolint:gosec
	_, _, err = runAndLog(cmd)
	return err
}

func (d *FusionDriver) IsRunning(vmxPath string) (bool, error) {
	cleanVmx := filepath.Clean(vmxPath)
	absVmxPath, err := filepath.Abs(cleanVmx)
	if err != nil {
		return false, err
	}

	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "list") //nolint:gosec
	stdout, _, err := runAndLog(cmd)
	if err != nil {
		return false, err
	}

	for _, line := range strings.Split(stdout, "\n") {
		if line == absVmxPath {
			return true, nil
		}
	}

	return false, nil
}

func (d *FusionDriver) CommHost(state multistep.StateBag) (string, error) {
	return CommHost(d.SSHConfig)(state)
}

func (d *FusionDriver) Start(vmxPath string, headless bool) error {
	cleanVmx := filepath.Clean(vmxPath)
	absVmxPath, err := filepath.Abs(cleanVmx)
	if err != nil {
		return err
	}

	guiArgument := guiArgumentNoGUI
	if !headless {
		guiArgument = guiArgumentGUI
	}

	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "start", absVmxPath, guiArgument) //nolint:gosec
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

func (d *FusionDriver) Stop(vmxPath string) error {
	cleanVmx := filepath.Clean(vmxPath)
	absVmxPath, err := filepath.Abs(cleanVmx)
	if err != nil {
		return err
	}

	cmd := exec.Command(d.vmrunPath(), "-T", "fusion", "stop", absVmxPath, "hard") //nolint:gosec
	if _, _, err := runAndLog(cmd); err != nil {
		// Check if the virtual machine is running. If not, it is stopped.
		running, runningErr := d.IsRunning(absVmxPath)
		if runningErr == nil && !running {
			return nil
		}
		return err
	}
	return nil
}

func (d *FusionDriver) SuppressMessages(vmxPath string) error {
	dir := filepath.Dir(vmxPath)
	base := filepath.Base(vmxPath)
	base = strings.ReplaceAll(base, ".vmx", "")

	plistPath := filepath.Join(dir, base+".plist")
	return os.WriteFile(plistPath, []byte(fusionSuppressPlist), 0644) //nolint:gosec
}

func (d *FusionDriver) libPath() string {
	return fusionPreferencesPath
}

func (d *FusionDriver) binaryPath(binaryName string) string {
	return filepath.Join(d.AppPath, "Contents", "Library", binaryName)
}

func (d *FusionDriver) toolsIsoPath(subdirs ...string) string {
	parts := append([]string{d.AppPath, "Contents", "Library", "isoimages"}, subdirs...)
	return filepath.Join(parts...)
}

func (d *FusionDriver) vmxPath() string {
	return d.binaryPath(appVmx)
}

func (d *FusionDriver) vmrunPath() string {
	return d.binaryPath(appVmrun)
}

func (d *FusionDriver) vdiskManagerPath() string {
	return d.binaryPath(appVdiskManager)
}

func (d *FusionDriver) isoFileName(base string) string {
	return base + ".iso"
}

func (d *FusionDriver) ToolsInstall() error {
	return nil
}

func (d *FusionDriver) Clone(dst, src string, linked bool, snapshot string) error {
	cleanDst := filepath.Clean(dst)
	absDst, err := filepath.Abs(cleanDst)
	if err != nil {
		return err
	}

	cleanSrc := filepath.Clean(src)
	absSrc, err := filepath.Abs(cleanSrc)
	if err != nil {
		return err
	}

	var cloneType string
	if linked {
		cloneType = cloneTypeLinked
	} else {
		cloneType = cloneTypeFull
	}

	args := []string{"-T", "fusion", "clone", absSrc, absDst, cloneType}
	if snapshot != "" {
		args = append(args, "-snapshot", snapshot)
	}

	cmd := exec.Command(d.vmrunPath(), args...) //nolint:gosec
	if _, _, err := runAndLog(cmd); err != nil {
		return err
	}

	return nil
}

func (d *FusionDriver) Verify() error {
	log.Printf("[INFO] Searching for %s...", fusionProductName)

	fusionVersion, err := d.getFusionVersion()

	if err != nil {
		return fmt.Errorf("error getting %s version: %s", fusionProductName, err)
	}

	log.Printf("[INFO] %s: %s", fusionProductName, fusionVersion)
	log.Printf("[INFO] Checking %s paths...", fusionProductName)

	if _, err := os.Stat(d.AppPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s.app not found at: %s", fusionProductName, d.AppPath)
		}

		return err
	}

	log.Printf("[INFO] - %s.app found at: %s", fusionProductName, d.AppPath)

	if _, err := os.Stat(d.vmxPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found at: %s", appVmx, d.vmxPath())
		}

		return err
	}

	log.Printf("[INFO] - %s found at: %s", appVmx, d.vmxPath())

	if _, err := os.Stat(d.vmrunPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found at: %s", appVmrun, d.vmrunPath())
		}

		return err
	}

	log.Printf("[INFO] - %s found at: %s", appVmrun, d.vmrunPath())

	if _, err := os.Stat(d.vdiskManagerPath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found at: %s", appVdiskManager, d.vdiskManagerPath())
		}
		return err
	}

	log.Printf("[INFO] - %s found at: %s", appVdiskManager, d.vdiskManagerPath())

	libpath := d.libPath()

	d.DhcpLeasesPath = func(device string) string {
		return "/var/db/vmware/vmnet-dhcpd-" + device + ".leases"
	}

	d.DhcpConfPath = func(device string) string {
		return filepath.Join(libpath, device, "dhcpd.conf")
	}

	d.VmnetnatConfPath = func(device string) string {
		return filepath.Join(libpath, device, "nat.conf")
	}

	d.NetworkMapper = func() (NetworkNameMapper, error) {
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

	return compareVersionObjects(fusionVersion, fusionMinVersionObj, fusionProductName)
}

func (d *FusionDriver) ToolsIsoPath(k string) string {
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = archAMD64
	default:
		arch = archARM64
	}

	log.Printf("[INFO] Selected architecture: %s", arch)
	return d.toolsIsoPath(arch, d.isoFileName(k))
}

func (d *FusionDriver) GetVmwareDriver() VmwareDriver {
	return d.VmwareDriver
}

func (d *FusionDriver) getFusionVersion() (*version.Version, error) {
	var stderr bytes.Buffer

	cleanVmxPath := filepath.Clean(d.vmxPath())
	cmd := exec.Command(cleanVmxPath, "-v") //nolint:gosec
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error getting version: %w", err)
	}

	versionMatch := productVersion.FindStringSubmatch(stderr.String())
	if versionMatch == nil {
		return nil, fmt.Errorf("error parsing version from output: %s", stderr.String())
	}

	versionStr := versionMatch[1]
	parsedVersion, err := version.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing version: %w", err)
	}

	return parsedVersion, nil
}
