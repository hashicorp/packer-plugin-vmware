// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package vmx

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	vmwcommon "github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

// StepAttachAdditionalDisks attaches additional disks to the virtual machine.
type StepAttachAdditionalDisks struct{}

// Run attaches additional disks to the virtual machine.
func (s *StepAttachAdditionalDisks) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)
	vmxPath := state.Get("vmx_path").(string)

	if len(config.AdditionalDiskSize) == 0 {
		// No additional disks to attach.
		return multistep.ActionContinue
	}

	ui.Sayf("Attaching %d additional disk(s)...", len(config.AdditionalDiskSize))

	// Read the existing .vmx configuration file.
	vmxData, err := vmwcommon.ReadVMX(vmxPath)
	if err != nil {
		err = fmt.Errorf("error reading .vmx file for additional disk attachment: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Detect adapter type from existing VMX file
	adapterType := s.detectAdapterType(vmxData)
	if adapterType == "" {
		err = fmt.Errorf("error reading .vmx file for the disk adapter type")
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Sayf("Detected existing disk adapter type: %s", adapterType)

	// Find the next available unit number for the adapter type.
	nextUnit := s.getNextAvailableUnit(vmxData, adapterType)

	// Attach additional disks to the virtual machine.
	incrementer := 1
	for i := range config.AdditionalDiskSize {
		// SCSI slot 7 is reserved; increment the unit number for the next disk.
		if i+incrementer == 7 {
			incrementer = 2
			nextUnit++
		}

		diskNumber := i + incrementer
		diskUnit := nextUnit + i
		if diskUnit >= 7 && adapterType == "scsi" {
			diskUnit++ // Skip reserved slot 7.
		}

		diskPrefix := fmt.Sprintf("%s0:%d", adapterType, diskUnit)
		diskFilename := fmt.Sprintf("%s-%d.vmdk", config.DiskName, diskNumber)

		// Add disk entries to the .vmx configuration file.
		vmxData[diskPrefix+".filename"] = diskFilename
		vmxData[diskPrefix+".present"] = "TRUE"
		vmxData[diskPrefix+".redo"] = ""

		ui.Sayf("Attached additional disk: %s at %s", diskFilename, diskPrefix)
	}

	// Write updated .vmx configuration file.
	if err := vmwcommon.WriteVMX(vmxPath, vmxData); err != nil {
		err = fmt.Errorf("error updating VMX file with additional disks: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// detectAdapterType detects the disk adapter type from existing VMX configuration.
func (s *StepAttachAdditionalDisks) detectAdapterType(vmxData map[string]string) string {
	adapterTypes := []string{"nvme", "sata", "scsi", "ide"}

	for _, adapterType := range adapterTypes {
		prefix := adapterType + "0:"
		for key := range vmxData {
			if strings.HasPrefix(key, prefix) && strings.HasSuffix(key, ".present") {
				if vmxData[key] == "TRUE" {
					return adapterType
				}
			}
		}
	}

	return "" // No disk adapter type detected.
}

// getNextAvailableUnit returns the next available unit number for the given adapter type.
func (s *StepAttachAdditionalDisks) getNextAvailableUnit(vmxData map[string]string, adapterType string) int {
	maxUnit := -1
	prefix := adapterType + "0:"

	for key := range vmxData {
		if strings.HasPrefix(key, prefix) {
			// Extract the unit number from the key.
			remaining := key[len(prefix):]
			dotIndex := strings.Index(remaining, ".")
			if dotIndex > 0 {
				unitStr := remaining[:dotIndex]
				if unit, err := strconv.Atoi(unitStr); err == nil {
					if unit > maxUnit {
						maxUnit = unit
					}
				}
			}
		}
	}

	return maxUnit + 1
}

// Cleanup performs any necessary cleanup operations after attaching additional disks.
func (s *StepAttachAdditionalDisks) Cleanup(state multistep.StateBag) {}
