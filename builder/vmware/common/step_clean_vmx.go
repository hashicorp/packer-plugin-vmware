// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepCleanVMX cleans up the VMX configuration by removing temporary build devices.
type StepCleanVMX struct {
	RemoveEthernetInterfaces bool
	VNCEnabled               bool
}

// Run executes the VMX cleanup step, removing temporary devices and configurations.
func (s StepCleanVMX) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	vmxPath := state.Get("vmx_path").(string)

	ui.Say("Cleaning VMX prior to finishing up...")

	vmxData, err := ReadVMX(vmxPath)
	if err != nil {
		state.Put("error", fmt.Errorf("error reading VMX: %s", err))
		return multistep.ActionHalt
	}

	// Grab our list of devices added during the build out of the statebag
	for _, device := range state.Get("temporaryDevices").([]string) {
		// Instead of doing this in one pass which would be more efficient,
		// we do it per device-type so that the logic appears to be the same
		// as the prior implementation.

		// Walk through all the devices that were temporarily added and figure
		// out which type it is in order to figure out how to disable it.
		// Right now only floppy, cdrom devices, ethernet, and devices that use
		// ".present" are supported.
		if strings.HasPrefix(device, "floppy") {
			// We can identify a floppy device because it begins with "floppy"
			ui.Sayf("Unmounting %s from VMX...", device)

			// Delete the floppy%d entries so the floppy is no longer mounted
			for k := range vmxData {
				if strings.HasPrefix(k, fmt.Sprintf("%s.", device)) {
					log.Printf("[INFO] Deleting key for floppy device: %s", k)
					delete(vmxData, k)
				}
			}
			vmxData[fmt.Sprintf("%s.present", device)] = "FALSE"

		} else if strings.HasPrefix(vmxData[fmt.Sprintf("%s.devicetype", device)], "cdrom-") {
			// We can identify something is a cdrom if it has a ".devicetype"
			// attribute that begins with "cdrom-"
			ui.Sayf("Detaching ISO from CD-ROM device %s...", device)

			// Simply turn the CD-ROM device into a native cdrom instead of an iso
			vmxData[fmt.Sprintf("%s.devicetype", device)] = "cdrom-raw"
			vmxData[fmt.Sprintf("%s.filename", device)] = "auto detect"
			vmxData[fmt.Sprintf("%s.clientdevice", device)] = "TRUE"

		} else if strings.HasPrefix(device, "ethernet") && s.RemoveEthernetInterfaces {
			// We can identify an ethernet device because it begins with "ethernet"
			// Although we're supporting this, as of now it's not in use due
			// to these interfaces not ever being added to the "temporaryDevices" statebag.
			ui.Sayf("Removing %s interface...", device)

			// Delete the ethernet%d entries so the ethernet interface is removed.
			// This corresponds to the same logic defined below.
			for k := range vmxData {
				if strings.HasPrefix(k, fmt.Sprintf("%s.", device)) {
					log.Printf("[INFO] Deleting key for ethernet device: %s", k)
					delete(vmxData, k)
				}
			}
		} else {
			// Check to see if the device can be disabled.
			if _, ok := vmxData[fmt.Sprintf("%s.present", device)]; ok {
				ui.Sayf("Disabling device %s of an unknown device type...", device)
				vmxData[fmt.Sprintf("%s.present", device)] = "FALSE"
			} else {
				log.Printf("[INFO] Refusing to remove device due to being of an unsupported type: %s\n", device)
				for k := range vmxData {
					if strings.HasPrefix(k, fmt.Sprintf("%s.", device)) {
						log.Printf("[INFO] Leaving unsupported device key: %s\n", k)
					}
				}
			}
		}
	}

	// Disable the VNC server, if necessary.
	if s.VNCEnabled {
		ui.Say("Disabling VNC server...")
		vmxData["remotedisplay.vnc.enabled"] = "FALSE"
	}

	// Remove any ethernet devices, if necessary.
	if s.RemoveEthernetInterfaces {
		ui.Say("Removing Ethernet devices...")
		for k := range vmxData {
			if strings.HasPrefix(k, "ethernet") {
				log.Printf("[INFO] Deleting key for ethernet device: %s", k)
				delete(vmxData, k)
			}
		}
	}

	// Write to the VMX.
	if err := WriteVMX(vmxPath, vmxData); err != nil {
		state.Put("error", fmt.Errorf("error writing VMX: %s", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup after the VMX cleaning step completes.
func (StepCleanVMX) Cleanup(multistep.StateBag) {}
