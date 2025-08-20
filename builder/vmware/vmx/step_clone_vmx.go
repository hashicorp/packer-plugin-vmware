// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vmx

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/tmp"
	vmwcommon "github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

// StepCloneVMX clones the source virtual machine a supplied path.
type StepCloneVMX struct {
	OutputDir *string
	Path      string
	VMName    string
	Linked    bool
	Snapshot  string
	tempDir   string
}

func (s *StepCloneVMX) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	halt := func(err error) multistep.StepAction {
		state.Put("error", err)
		return multistep.ActionHalt
	}

	driver := state.Get("driver").(vmwcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)

	// Set the path for the resulting .vmx file.
	vmxPath := filepath.Join(*s.OutputDir, s.VMName+".vmx")

	lowerSrc := strings.ToLower(s.Path)

	// If the source is a .ova/.ovf file, use ovftool.
	if strings.HasSuffix(lowerSrc, ".ovf") || strings.HasSuffix(lowerSrc, ".ova") {
		// Clone the source virtual machine from the .ova/.ovf file.
		ui.Sayf("Cloning from source .ova/.ovf...")
		log.Printf("[INFO] Cloning from: %s", s.Path)
		log.Printf("[INFO] Cloning to: %s", *s.OutputDir)

		ovftool := vmwcommon.GetOvfTool()
		if ovftool == "" {
			return halt(fmt.Errorf("ovftool not found in PATH"))
		}

		// Ensure that the output directory exists.
		if err := os.MkdirAll(*s.OutputDir, 0o755); err != nil {
			return halt(fmt.Errorf("failed to create output directory: %w", err))
		}

		args := []string{
			"--lax",
			fmt.Sprintf("--name=%s", s.VMName),
			s.Path,
			*s.OutputDir,
		}

		cmd := exec.CommandContext(ctx, ovftool, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return halt(fmt.Errorf("failed to clone from .ovf/.ova: %w", err))
		}

		ui.Say("Successfully cloned from .ovf/.ova.")
	} else {
		// Clone the source virtual machine from the .vmx configuration file.
		ui.Say("Cloning from source .vmx...")
		log.Printf("[INFO] Cloning from: %s", s.Path)
		log.Printf("[INFO] Cloning to: %s", vmxPath)

		if err := driver.Clone(vmxPath, s.Path, s.Linked, s.Snapshot); err != nil {
			return halt(fmt.Errorf("failed to clone from .vmx: %s", err))
		}

		ui.Say("Successfully cloned from .vmx.")
	}

	// Read in the machine configuration from the resulting .vmx file.
	if remoteDriver, ok := driver.(vmwcommon.RemoteDriver); ok {
		remoteVmxPath := vmxPath
		tempDir, err := tmp.Dir("packer-vmx")
		if err != nil {
			return halt(err)
		}
		s.tempDir = tempDir
		vmxPath = filepath.Join(tempDir, s.VMName+".vmx")
		if err = remoteDriver.Download(remoteVmxPath, vmxPath); err != nil {
			return halt(err)
		}
	}

	vmxData, err := vmwcommon.ReadVMX(vmxPath)
	if err != nil {
		return halt(err)
	}

	var diskFilenames []string
	diskPathKeyRe := regexp.MustCompile(`(?i)^(scsi|sata|ide|nvme)[[:digit:]]:[[:digit:]]{1,2}\.fileName`)
	for k, v := range vmxData {
		match := diskPathKeyRe.FindString(k)
		if match != "" && filepath.Ext(v) == ".vmdk" {
			diskFilenames = append(diskFilenames, v)
		}
	}

	var diskFullPaths []string
	for _, diskFilename := range diskFilenames {
		log.Printf("[INFO] Found attached disk with filename: %s", diskFilename)
		diskFullPaths = append(diskFullPaths, filepath.Join(*s.OutputDir, diskFilename))
	}

	if len(diskFullPaths) == 0 {
		return halt(fmt.Errorf("unable to enumerate disk info from the vmx file"))
	}

	var networkType string
	if _, ok := vmxData["ethernet0.connectiontype"]; ok {
		networkType = vmxData["ethernet0.connectiontype"]
		log.Printf("[INFO] Discovered the network type: %s", networkType)
	}
	if networkType == "" {
		networkType = vmwcommon.DefaultNetworkType
		log.Printf("[INFO] Defaulting to network type: %s", networkType)
	}

	state.Put("vmx_path", vmxPath)
	state.Put("disk_full_paths", diskFullPaths)
	state.Put("vmnetwork", networkType)

	return multistep.ActionContinue
}

func (s *StepCloneVMX) Cleanup(state multistep.StateBag) {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}
