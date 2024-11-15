// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build windows

package common

import (
	"fmt"
	"log"
	"regexp"
	"syscall"

	"github.com/hashicorp/go-version"
)

func workstationVerifyVersion(requiredVersion string) error {
	key := `SOFTWARE\Wow6432Node\VMware, Inc.\VMware Workstation`
	subkey := "ProductVersion"
	productVersion, err := readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
	if err != nil {
		log.Printf(`[WARN] Unable to read registry key %s\%s`, key, subkey)
		key = `SOFTWARE\VMware, Inc.\VMware Workstation`
		productVersion, err = readRegString(syscall.HKEY_LOCAL_MACHINE, key, subkey)
		if err != nil {
			log.Printf(`[WARN] Unable to read registry key %s\%s`, key, subkey)
			return err
		}
	}

	versionRe := regexp.MustCompile(`^(\d+\.\d+\.\d+)`)
	matches := versionRe.FindStringSubmatch(productVersion)
	if matches == nil {
		return fmt.Errorf(
			`Could not find a VMware Workstation version in registry key %s\%s: '%s'`, key, subkey, productVersion)
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
