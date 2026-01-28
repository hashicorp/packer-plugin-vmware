// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
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
	vmwcommon "github.com/vmware/packer-plugin-vmware/builder/vmware/common"
)

// StepCloneVMX clones the source virtual machine from a supplied path.
type StepCloneVMX struct {
	OutputDir   *string
	Path        string
	VMName      string
	Linked      bool
	Snapshot    string
	Version     int
	GuestOSType string
	tempDir     string
}

// Run executes the VMX cloning step, creating a copy of the source virtual machine.
func (s *StepCloneVMX) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	halt := func(err error) multistep.StepAction {
		state.Put("error", err)
		return multistep.ActionHalt
	}

	driver := state.Get("driver").(vmwcommon.Driver)
	ui := state.Get("ui").(packersdk.Ui)

	lowerSrc := strings.ToLower(s.Path)
	var vmxPath string

	// If the source is a .ovf/.ova file, use ovftool.
	if strings.HasSuffix(lowerSrc, ".ovf") || strings.HasSuffix(lowerSrc, ".ova") {
		// Clone the source virtual machine from the .ovf/.ova file.
		ui.Sayf("Cloning from source .ovf/.ova...")
		log.Printf("[INFO] Cloning from: %s", s.Path)
		log.Printf("[INFO] Cloning to: %s", *s.OutputDir)

		// ovftool always creates a subdirectory with the virtual machine name.
		// Pass the output directory to ovftool, then move the contents up one level.
		ovftoolTargetDir := *s.OutputDir

		// Ensure that the output directory exists.
		if err := os.MkdirAll(ovftoolTargetDir, 0o755); err != nil {
			return halt(fmt.Errorf("failed to create output directory: %w", err))
		}

		// Set up the ovftool command.
		ovftool := vmwcommon.GetOvfTool()

		// Pass the virtual machine name, virtual hardware version, and output directory to ovftool.
		args := []string{
			"--lax",
			fmt.Sprintf("--maxVirtualHardwareVersion=%d", s.Version),
			fmt.Sprintf("--name=%s", s.VMName),
			s.Path,
			ovftoolTargetDir,
		}

		cmd := exec.CommandContext(ctx, ovftool, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return halt(fmt.Errorf("failed to clone from .ovf/.ova: %w", err))
		}

		// Determine where ovftool actually created the output within the target directory.
		// ovftool creates either <target>/<vmname> or <target>/<vmname>.vmwarevm depending on the platform.
		ovftoolCreatedPath := filepath.Join(ovftoolTargetDir, s.VMName)
		if _, err := os.Stat(ovftoolCreatedPath); os.IsNotExist(err) {
			// Check if ovftool created a .vmwarevm bundle instead (VMware Fusion on macOS).
			vmwarevmPath := ovftoolCreatedPath + ".vmwarevm"
			if _, err := os.Stat(vmwarevmPath); err == nil {
				ovftoolCreatedPath = vmwarevmPath
			} else {
				return halt(fmt.Errorf("ovftool output not found at %s or %s", ovftoolCreatedPath, vmwarevmPath))
			}
		}

		// Move the ovftool output contents to the root of the output directory.
		// Use a temporary directory outside the output directory to avoid conflicts.
		log.Printf("[INFO] Moving output from %s to %s", ovftoolCreatedPath, *s.OutputDir)
		tempDir := strings.TrimRight(*s.OutputDir, string(filepath.Separator)) + ".tmp"
		s.tempDir = tempDir
		if err := os.Rename(ovftoolCreatedPath, tempDir); err != nil {
			return halt(fmt.Errorf("failed to rename ovftool output: %w", err))
		}

		// Remove the output directory.
		if err := os.RemoveAll(*s.OutputDir); err != nil && !os.IsNotExist(err) {
			if restoreErr := os.Rename(tempDir, ovftoolCreatedPath); restoreErr != nil {
				log.Printf("[WARN] Failed to restore ovftool output after error: %s", restoreErr)
			} else {
				log.Printf("[INFO] Restored ovftool output to original location after error")
			}
			return halt(fmt.Errorf("failed to remove output directory: %w", err))
		}

		// Ensure parent directories exist before the final move.
		// Use the cleaned output directory path to get the correct parent.
		cleanedOutputDir := strings.TrimRight(*s.OutputDir, string(filepath.Separator))
		if err := os.MkdirAll(filepath.Dir(cleanedOutputDir), 0o755); err != nil {
			return halt(fmt.Errorf("failed to create parent directories: %w", err))
		}

		// Move the temporary directory to the final output location.
		if err := os.Rename(tempDir, *s.OutputDir); err != nil {
			return halt(fmt.Errorf("failed to move ovftool results to output directory: %w", err))
		}
		s.tempDir = ""

		// Find the .vmx file in the output directory.
		vmxPath = filepath.Join(*s.OutputDir, s.VMName+".vmx")
		if _, err := os.Stat(vmxPath); os.IsNotExist(err) {
			// VMware Fusion: Check for .vmwarevm bundle from ovftool.
			vmxPath = filepath.Join(*s.OutputDir, s.VMName+".vmwarevm", s.VMName+".vmx")
			if _, err := os.Stat(vmxPath); os.IsNotExist(err) {
				// Search for any .vmx file in the output directory.
				var found bool
				err := filepath.Walk(*s.OutputDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".vmx") {
						vmxPath = path
						found = true
						return filepath.SkipAll
					}
					return nil
				})
				if err != nil || !found {
					return halt(fmt.Errorf("unable to find .vmx file after ovftool conversion"))
				}
			}
		}

		// Override guest operating system identifier, if specified.
		if s.GuestOSType != "" {
			log.Printf("[INFO] Overriding guest operating system identifier set by ovftool: %s", s.GuestOSType)
			vmxData, err := vmwcommon.ReadVMX(vmxPath)
			if err != nil {
				return halt(fmt.Errorf("failed to read vmx: %w", err))
			}

			vmxData["guestos"] = s.GuestOSType

			if err := vmwcommon.WriteVMX(vmxPath, vmxData); err != nil {
				return halt(fmt.Errorf("failed to write vmx: %w", err))
			}
		}
	} else {
		// Clone the source virtual machine from the .vmx configuration file.
		ui.Say("Cloning from source .vmx...")
		vmxPath = filepath.Join(*s.OutputDir, s.VMName+".vmx")
		log.Printf("[INFO] Cloning from: %s", s.Path)
		log.Printf("[INFO] Cloning to: %s", vmxPath)

		if err := driver.Clone(vmxPath, s.Path, s.Linked, s.Snapshot); err != nil {
			return halt(fmt.Errorf("failed to clone from .vmx: %s", err))
		}
	}

	ui.Say("Successfully cloned the source virtual machine.")

	// Read in the virtual machine configuration from the cloned .vmx file.
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
	vmxDir := filepath.Dir(vmxPath)
	for _, diskFilename := range diskFilenames {
		log.Printf("[INFO] Found attached disk with filename: %s", diskFilename)
		// Disk paths are relative to the .vmx file location, not OutputDir.
		diskFullPaths = append(diskFullPaths, filepath.Join(vmxDir, diskFilename))
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

// Cleanup removes any temporary directories created during the cloning process.
func (s *StepCloneVMX) Cleanup(state multistep.StateBag) {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}
