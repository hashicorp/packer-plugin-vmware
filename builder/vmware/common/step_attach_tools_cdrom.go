// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepAttachToolsCDROM represents a step to attach VMware Tools ISO as a CD-ROM device.
type StepAttachToolsCDROM struct {
	ToolsMode         string
	ToolsSourcePath   string
	ToolsUploadFlavor string
	CDROMAdapterType  string
}

// Run executes the tools CD-ROM attachment step, attaching the VMware Tools ISO as a CD-ROM device.
func (s *StepAttachToolsCDROM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	// Skip if tools mode is not 'attach'.
	if s.ToolsMode != toolsModeAttach {
		return multistep.ActionContinue
	}

	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(Driver)

	// Determine the tools source path.
	var toolsPath string

	// Check if path is already in state.
	if toolsSourcePath, ok := state.GetOk("tools_attach_source"); ok && toolsSourcePath != "" {
		toolsPath = toolsSourcePath.(string)
	} else {
		// Resolve path from configuration.
		toolsPath = s.ToolsSourcePath
		if toolsPath == "" && s.ToolsUploadFlavor != "" {
			toolsPath = driver.ToolsIsoPath(s.ToolsUploadFlavor)
		}
	}

	// Skip if no tools path is available.
	if toolsPath == "" {
		log.Println("[INFO] No tools source path available, skipping tools CD-ROM attachment")
		return multistep.ActionContinue
	}

	// Validate that the file exists.
	if _, err := os.Stat(toolsPath); err != nil {
		if os.IsNotExist(err) && s.ToolsUploadFlavor != "" {
			err = fmt.Errorf("unable to locate a VMware Tools ISO for %q in the default path: %s", s.ToolsUploadFlavor, toolsPath)
		} else {
			err = fmt.Errorf("error accessing VMware Tools ISO: %s", err)
		}
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	vmxPath := state.Get("vmx_path").(string)
	log.Printf("[INFO] Attaching VMware Tools ISO as CD-ROM: %s", toolsPath)

	vmxData, err := ReadVMX(vmxPath)
	if err != nil {
		err = fmt.Errorf("error reading VMX file: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Determine the CD-ROM adapter type.
	adapterType := s.CDROMAdapterType
	if adapterType == "" {
		adapterType = "ide"
	}

	// Find the next available slot.
	devicePath, err := FindNextAvailableCDROMSlot(vmxData, adapterType)
	if err != nil {
		err = fmt.Errorf("error finding available CD-ROM slot: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	log.Printf("[INFO] Using CD-ROM device slot: %s", devicePath)

	// Attach the CD-ROM device.
	err = AttachCDROMDevice(vmxData, devicePath, toolsPath, adapterType)
	if err != nil {
		err = fmt.Errorf("error attaching tools CD-ROM device: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Write the updated .vmx configuration file.
	err = WriteVMX(vmxPath, vmxData)
	if err != nil {
		err = fmt.Errorf("error writing VMX file: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Store the attached device information in state bag for cleanup.
	state.Put("tools_cdrom_device", devicePath)
	state.Put("tools_cdrom_adapter", adapterType)

	// Add to temporary devices list for cleanup.
	tmpBuildDevices, ok := state.GetOk("temporaryDevices")
	if !ok {
		tmpBuildDevices = []string{}
	}
	devices := tmpBuildDevices.([]string)
	devices = append(devices, devicePath)
	state.Put("temporaryDevices", devices)

	ui.Say(fmt.Sprintf("Attached VMware Tools ISO as CD-ROM device: %s", devicePath))

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup after the tools CD-ROM attachment step completes.
func (s *StepAttachToolsCDROM) Cleanup(state multistep.StateBag) {
	// No-op
}
