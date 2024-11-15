// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows

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

	"github.com/hashicorp/go-version"
)

// VMware Workstation Player for Linux.

func playerFindVmplayer() (string, error) {
	return exec.LookPath(appPlayer)
}

func playerFindVmrun() (string, error) {
	return exec.LookPath(appVmrun)
}

func playerFindVdiskManager() (string, error) {
	return exec.LookPath(appVdiskManager)
}

func playerFindQemuImg() (string, error) {
	return exec.LookPath(appQemuImg)
}

func playerToolsIsoPath(flavor string) string {
	return "/usr/lib/vmware/isoimages/" + flavor + ".iso"
}

// Return the base path to configuration files.
func playerInstallationPath() (s string, err error) {
	return "/etc/vmware", nil
}

// Helper function to find configuration paths
func playerFindConfigPath(device string, paths []string) string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("Error finding configuration root path: %s", err)
		return ""
	}

	devicebase := filepath.Join(base, device)
	for _, p := range paths {
		fp := filepath.Join(devicebase, p)
		if _, err := os.Stat(fp); !os.IsNotExist(err) {
			return fp
		}
	}

	log.Printf("Error finding configuration file in device path: %s", devicebase)
	return ""
}

func playerDhcpLeasesPath(device string) string {
	return playerFindConfigPath(device, GetDhcpLeasesPaths())
}

func playerVmDhcpConfPath(device string) string {
	return playerFindConfigPath(device, GetDhcpConfPaths())
}

func playerVmnetnatConfPath(device string) string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, device, "nat/nat.conf")
}

func playerNetmapConfPath() string {
	base, err := playerInstallationPath()
	if err != nil {
		log.Printf("Error finding the configuration root path: %s", err)
		return ""
	}
	return filepath.Join(base, netmapConfFile)
}

func playerVerifyVersion(requiredVersion string) error {
	if runtime.GOOS != osLinux {
		return fmt.Errorf("driver is only supported on linux and windows, not %s", runtime.GOOS)
	}

	// Using the default.
	vmxpath := "/usr/lib/vmware/bin/" + appVmx

	var stderr bytes.Buffer
	cmd := exec.Command(vmxpath, "-v")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	versionRe := regexp.MustCompile(`(?i)VMware Player (\d+)\.(\d+)\.(\d+)`)
	matches := versionRe.FindStringSubmatch(stderr.String())
	if matches == nil {
		return fmt.Errorf("error parsing version from output: %s", stderr.String())
	}

	fullVersion := fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
	log.Printf("[INFO] %s: %s", playerProductName, fullVersion)

	parsedVersionFound, err := version.NewVersion(fullVersion)
	if err != nil {
		return fmt.Errorf("invalid version found: %w", err)
	}

	parsedVersionRequired, err := version.NewVersion(requiredVersion)
	if err != nil {
		return fmt.Errorf("invalid version required: %w", err)
	}

	return compareVersionObjects(parsedVersionFound, parsedVersionRequired, playerProductName)
}
