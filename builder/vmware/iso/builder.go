// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package iso provides a Packer builder for creating VMware virtual machines from ISO images.
// This builder creates virtual machines by booting from an ISO file and installing an operating system.
package iso

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	vmwcommon "github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

// Builder is responsible for constructing the virtual machine based on
// provided settings and steps.
type Builder struct {
	config Config
	runner multistep.Runner
}

// ConfigSpec returns the HCL2 object specification for the builder's configuration.
func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

// Prepare validates the raw configuration and updates the builder's settings.
func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	warnings, errs := b.config.Prepare(raws...)
	if errs != nil {
		return nil, warnings, errs
	}

	return nil, warnings, nil
}

// Run executes the builder's steps to create a virtual machine from an ISO image.
func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	driver, err := vmwcommon.NewDriver(&b.config.DriverConfig, &b.config.SSHConfig, b.config.VMName)

	if err != nil {
		return nil, fmt.Errorf("failed creating driver : %s", err)
	}

	// Verify that ovftool is installed if exporting the virtual machine.
	if err := driver.VerifyOvfTool(b.config.SkipExport, false); err != nil {
		return nil, err
	}

	// Set up the state.
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)
	state.Put("sshConfig", &b.config.SSHConfig)
	state.Put("driverConfig", &b.config.DriverConfig)
	state.Put("temporaryDevices", []string{}) // Devices (in .vmx) created during the build.

	// Build the steps.
	steps := []multistep.Step{
		&vmwcommon.StepPrepareTools{
			ToolsUploadFlavor: b.config.ToolsUploadFlavor,
		},
		&commonsteps.StepDownload{
			Checksum:    b.config.ISOChecksum,
			Description: "ISO",
			Extension:   b.config.TargetExtension,
			ResultKey:   "iso_path",
			TargetPath:  b.config.TargetPath,
			Url:         b.config.ISOUrls,
		},
		&vmwcommon.StepOutputDir{
			Force:        b.config.PackerForce,
			OutputConfig: &b.config.OutputConfig,
			VMName:       b.config.VMName,
		},
		multistep.If(b.config.Comm.Type == "ssh", &communicator.StepSSHKeyGen{
			CommConf:            &b.config.Comm,
			SSHTemporaryKeyPair: b.config.Comm.SSHTemporaryKeyPair,
		}),
		&commonsteps.StepCreateFloppy{
			Files:       b.config.FloppyFiles,
			Directories: b.config.FloppyDirectories,
			Content:     b.config.FloppyContent,
			Label:       b.config.FloppyLabel,
		},
		&commonsteps.StepCreateCD{
			Files:   b.config.CDFiles,
			Content: b.config.CDContent,
			Label:   b.config.CDLabel,
		},
		&vmwcommon.StepCreateDisks{
			OutputDir:          &b.config.OutputDir,
			CreateMainDisk:     true,
			DiskName:           b.config.DiskName,
			MainDiskSize:       b.config.DiskSize,
			AdditionalDiskSize: b.config.AdditionalDiskSize,
			DiskAdapterType:    b.config.DiskAdapterType,
			DiskTypeId:         b.config.DiskTypeId,
		},
		&stepCreateVMX{},
		&vmwcommon.StepConfigureVMX{
			CustomData:       b.config.VMXData,
			VMName:           b.config.VMName,
			DisplayName:      b.config.VMXDisplayName,
			DiskAdapterType:  b.config.DiskAdapterType,
			CDROMAdapterType: b.config.CdromAdapterType,
		},
		&vmwcommon.StepSuppressMessages{},
		&vmwcommon.StepHTTPIPDiscover{},
		commonsteps.HTTPServerFromHTTPConfig(&b.config.HTTPConfig),
		&vmwcommon.StepConfigureVNC{
			Enabled:            !b.config.DisableVNC,
			VNCBindAddress:     b.config.VNCBindAddress,
			VNCPortMin:         b.config.VNCPortMin,
			VNCPortMax:         b.config.VNCPortMax,
			VNCDisablePassword: b.config.VNCDisablePassword,
		},
		&vmwcommon.StepRun{
			DurationBeforeStop: 5 * time.Second,
			Headless:           b.config.Headless,
		},
		&vmwcommon.StepVNCConnect{
			VNCEnabled:   !b.config.DisableVNC,
			DriverConfig: &b.config.DriverConfig,
		},
		&vmwcommon.StepVNCBootCommand{
			Config: b.config.VNCConfig,
			VMName: b.config.VMName,
			Ctx:    b.config.ctx,
			Comm:   &b.config.Comm,
		},
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      driver.CommHost,
			SSHConfig: b.config.Comm.SSHConfigFunc(),
		},
		&vmwcommon.StepUploadTools{
			ToolsUploadFlavor: b.config.ToolsUploadFlavor,
			ToolsUploadPath:   b.config.ToolsUploadPath,
			Ctx:               b.config.ctx,
		},
		&commonsteps.StepProvision{},
		&commonsteps.StepCleanupTempKeys{
			Comm: &b.config.Comm,
		},
		&vmwcommon.StepShutdown{
			Command: b.config.ShutdownCommand,
			Timeout: b.config.ShutdownTimeout,
		},
		&vmwcommon.StepCleanFiles{},
		&vmwcommon.StepCompactDisk{
			Skip: b.config.SkipCompaction,
		},
		&vmwcommon.StepConfigureVMX{
			CustomData:  b.config.VMXDataPost,
			SkipDevices: true,
			VMName:      b.config.VMName,
			DisplayName: b.config.VMXDisplayName,
		},
		&vmwcommon.StepCleanVMX{
			RemoveEthernetInterfaces: b.config.VMXRemoveEthernet,
			VNCEnabled:               !b.config.DisableVNC,
		},
		&vmwcommon.StepCreateSnapshot{
			SnapshotName: &b.config.SnapshotName,
		},
		&vmwcommon.StepExport{
			Format:         b.config.Format,
			SkipExport:     b.config.SkipExport,
			VMName:         b.config.VMName,
			OVFToolOptions: b.config.OVFToolOptions,
			OutputDir:      &b.config.OutputDir,
		},
	}

	// Run the steps.
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

	// Report any errors.
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If interrupted or cancelled, then return.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("build was cancelled")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("build was halted")
	}

	// Generate the artifact.
	return vmwcommon.NewArtifact(b.config.Format, b.config.VMName, b.config.SkipExport, state)
}
