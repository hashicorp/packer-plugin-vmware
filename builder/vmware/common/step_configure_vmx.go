// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepConfigureVMX represents the configuration settings for a VMX configuration file.
type StepConfigureVMX struct {
	CustomData       map[string]string
	DisplayName      string
	SkipDevices      bool
	VMName           string
	DiskAdapterType  string
	CDROMAdapterType string
}

func (s *StepConfigureVMX) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	log.Printf("[INFO] Configuring VMX...\n")

	var err error
	ui := state.Get("ui").(packersdk.Ui)

	vmxPath := state.Get("vmx_path").(string)
	vmxData, err := ReadVMX(vmxPath)
	if err != nil {
		err = fmt.Errorf("error reading VMX file: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Set this so that no dialogs ever appear from Packer.
	vmxData["msg.autoanswer"] = "true"

	// Create a new UUID for this VM, since it is a new VM
	vmxData["uuid.action"] = "create"

	// Delete any generated addresses since we want to regenerate
	// them. Conflicting MAC addresses is a bad time.
	addrRegex := regexp.MustCompile(`(?i)^ethernet\d+\.generatedAddress`)
	for k := range vmxData {
		if addrRegex.MatchString(k) {
			delete(vmxData, k)
		}
	}

	// Set custom data
	for k, v := range s.CustomData {
		log.Printf("[INFO] Setting VMX: '%s' = '%s'", k, v)
		k = strings.ToLower(k)
		vmxData[k] = v
	}

	// StepConfigureVMX runs both before and after provisioning (for VmxDataPost),
	// the latter time shouldn't create temporary devices
	if !s.SkipDevices {
		// Grab list of temporary builder devices so we can append to it
		tmpBuildDevices := state.Get("temporaryDevices").([]string)

		// Set a floppy disk if we have one
		if floppyPathRaw, ok := state.GetOk("floppy_path"); ok {
			log.Println("Floppy path present, setting in VMX")
			vmxData["floppy0.present"] = "TRUE"
			vmxData["floppy0.filetype"] = "file"
			vmxData["floppy0.filename"] = floppyPathRaw.(string)

			// Add it to our list of build devices to later remove
			tmpBuildDevices = append(tmpBuildDevices, "floppy0")
		}

		// Add our custom CD, if it exists
		if cdPath, ok := state.GetOk("cd_path"); ok {
			if cdPath != "" {
				diskAndCDConfigData := DefaultDiskAndCDROMTypes(s.DiskAdapterType, s.CDROMAdapterType)
				cdromPrefix := diskAndCDConfigData.CdromType + "1:" + diskAndCDConfigData.CdromTypePrimarySecondary

				// Ensure the CD-ROM adapter is present.
				adapterKey := diskAndCDConfigData.CdromType + "1.present"
				vmxData[adapterKey] = "TRUE"

				// Configure the CD-ROM device.
				vmxData[cdromPrefix+".present"] = "TRUE"
				vmxData[cdromPrefix+".filename"] = cdPath.(string)
				vmxData[cdromPrefix+".devicetype"] = "cdrom-image"

				// Add both the adapter and device to our list of build devices to later remove.
				tmpBuildDevices = append(tmpBuildDevices, adapterKey, cdromPrefix)
			}
		}

		// Build the list back in our statebag
		state.Put("temporaryDevices", tmpBuildDevices)
	}

	if s.DisplayName != "" {
		vmxData["displayname"] = s.DisplayName
		state.Put("display_name", s.DisplayName)
	} else {
		displayName, ok := vmxData["displayname"]
		if !ok { // Packer converts key names to lowercase!
			err := errors.New("error returning value of displayName from VMX data")
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		} else {
			state.Put("display_name", displayName)
		}
	}

	// Set the extendedConfigFile setting for the .vmxf filename to the VMName
	// if displayName is not set. This is needed so that when VMware creates
	// the .vmxf file it matches the displayName if it is set. When just using
	// the displayName if it was empty VMware would make a file named ".vmxf".
	// The ".vmxf" file would not get deleted when the VM got deleted.
	if s.DisplayName != "" {
		vmxData["extendedconfigfile"] = fmt.Sprintf("%s.vmxf", s.DisplayName)
	} else {
		vmxData["extendedconfigfile"] = fmt.Sprintf("%s.vmxf", s.VMName)
	}

	err = WriteVMX(vmxPath, vmxData)

	if err != nil {
		err = fmt.Errorf("error writing VMX file: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

type DiskAndCDConfigData struct {
	ScsiPresent         string
	ScsiDiskAdapterType string
	SataPresent         string
	NvmePresent         string

	DiskType                  string
	CdromType                 string
	CdromTypePrimarySecondary string
	CdromPath                 string
}

// DefaultDiskAndCDROMTypes takes the disk adapter type and cdrom adapter type from the config and converts them
// into template interpolation data for creating or configuring a vmx.
func DefaultDiskAndCDROMTypes(diskAdapterType string, cdromAdapterType string) DiskAndCDConfigData {
	diskData := DiskAndCDConfigData{
		ScsiPresent:         "FALSE",
		ScsiDiskAdapterType: "lsilogic",
		SataPresent:         "FALSE",
		NvmePresent:         "FALSE",

		DiskType:                  "scsi",
		CdromType:                 "ide",
		CdromTypePrimarySecondary: "0",
	}
	diskAdapterType = strings.ToLower(diskAdapterType)
	switch diskAdapterType {
	case "ide":
		diskData.DiskType = "ide"
		diskData.CdromType = "ide"
		diskData.CdromTypePrimarySecondary = "1"
	case "sata":
		diskData.SataPresent = "TRUE"
		diskData.DiskType = "sata"
		diskData.CdromType = "sata"
		diskData.CdromTypePrimarySecondary = "1"
	case "nvme":
		diskData.NvmePresent = "TRUE"
		diskData.DiskType = "nvme"
		diskData.SataPresent = "TRUE"
		diskData.CdromType = "sata"
		diskData.CdromTypePrimarySecondary = "0"
	case "scsi":
		diskAdapterType = "lsilogic"
		fallthrough
	default:
		diskData.ScsiPresent = "TRUE"
		diskData.ScsiDiskAdapterType = diskAdapterType // defaults to lsilogic
		diskData.DiskType = "scsi"
		diskData.CdromType = "ide"
		diskData.CdromTypePrimarySecondary = "0"
	}

	// Handle the cdrom adapter type. If the disk adapter type and the
	//  cdrom adapter type are the same, then ensure that the cdrom is the
	//  secondary device on whatever bus the disk adapter is on.
	switch cdromAdapterType {
	case "":
		cdromAdapterType = diskData.CdromType
	case diskAdapterType:
		diskData.CdromTypePrimarySecondary = "1"
	default:
		diskData.CdromTypePrimarySecondary = "0"
	}

	switch cdromAdapterType {
	case "ide":
		diskData.CdromType = "ide"
	case "sata":
		diskData.SataPresent = "TRUE"
		diskData.CdromType = "sata"
	case "scsi":
		diskData.ScsiPresent = "TRUE"
		diskData.CdromType = "scsi"
	}
	return diskData
}

func (s *StepConfigureVMX) Cleanup(state multistep.StateBag) {
}
