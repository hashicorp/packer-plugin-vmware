// Copyright (c) HashiCorp, Inc.
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
		// No additional disks to attach
		return multistep.ActionContinue
	}

	ui.Sayf("Attaching %s additional disk(s)...", len(config.AdditionalDiskSize))

	// Read the existing .vmx configuration file.
	vmxData, err := vmwcommon.ReadVMX(vmxPath)
	if err != nil {
		err = fmt.Errorf("error reading VMX file for additional disk attachment: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Determine adapter type. Default to 'scsi' if not specified.
	adapterType := config.DiskAdapterType
	if adapterType == "" {
		adapterType = "scsi"
	}

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
		diskFilename := fmt.Sprintf("%s-%d.vmdk", config.VMName, diskNumber)

		// Add disk entries to the .vmx configuration file.
		vmxData[diskPrefix+".filename"] = diskFilename
		vmxData[diskPrefix+".present"] = "TRUE"
		vmxData[diskPrefix+".redo"] = "TRUE"

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
