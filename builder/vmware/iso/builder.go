// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	common "github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

type Builder struct {
	config Config
	runner multistep.Runner
}

type StepCheckUploadExists struct {
	TargetPath   string
	RemoteType   string
	SkipDownload func() bool
}

type conditionalStepDownload struct {
	condition func() bool
	step      multistep.Step
}

func (s *StepCheckUploadExists) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if s.SkipDownload() {
		return multistep.ActionContinue
	}
	return multistep.ActionContinue
}

func (s *StepCheckUploadExists) Cleanup(state multistep.StateBag) {
	// Nothing to clean up.
}

func (s *conditionalStepDownload) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if s.condition() {
		return s.step.Run(ctx, state)
	}
	return multistep.ActionContinue
}

func (c *conditionalStepDownload) Cleanup(state multistep.StateBag) {
	c.step.Cleanup(state)
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	warnings, errs := b.config.Prepare(raws...)
	if errs != nil {
		return nil, warnings, errs
	}

	return nil, warnings, nil
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	driver, err := common.NewDriver(&b.config.DriverConfig, &b.config.SSHConfig, b.config.VMName)
	if err != nil {
		return nil, fmt.Errorf("failed creating driver : %s", err)
	}

	// If Open Virtualization Format (OVF) Tool ('ovftool') is required for the
	// build, verify that the it is present and that credentials are valid,
	if err := driver.VerifyOvfTool(b.config.SkipExport, b.config.SkipValidateCredentials); err != nil {
		return nil, err
	}

	// Setup the state bag.
	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("debug", b.config.PackerDebug)
	state.Put("driver", driver)
	state.Put("hook", hook)
	state.Put("ui", ui)
	state.Put("sshConfig", &b.config.SSHConfig)
	state.Put("driverConfig", &b.config.DriverConfig)
	// Devices added to the .vmx by Packer during the build.
	state.Put("temporaryDevices", []string{})

	// Checks if a matching upload exists in the remote cache.
	// If so, the download step is skipped using the conditionalStepDownload.
	stepCheckUploadExists := &StepCheckUploadExists{
		TargetPath: b.config.TargetPath,
		RemoteType: b.config.RemoteType,
		SkipDownload: func() bool {
			driver, ok := state.Get("driver").(common.Driver)
			if !ok {
				return false
			}

			ui, ok := state.Get("ui").(packersdk.Ui)
			if !ok {
				return false
			}

			remote, ok := driver.(common.RemoteDriver)
			if !ok {
				return false
			}

			if esx5, ok := remote.(*common.ESX5Driver); ok {
				remotePath := esx5.CachePath(b.config.TargetPath)
				ui.Say("Verifying availability of the ISO file in the remote cache...")

				if esx5.VerifyChecksum(b.config.ISOChecksum, remotePath) {
					ui.Say("ISO file is available in the remote cache; download will be skipped.")
					state.Put("iso_path", remotePath)
					return true
				} else {
					ui.Say("ISO file is not available in the remote cache; download will proceed.")
				}
			}

			return false
		},
	}

	steps := []multistep.Step{
		&common.StepPrepareTools{
			RemoteType:        b.config.RemoteType,
			ToolsUploadFlavor: b.config.ToolsUploadFlavor,
		},
		stepCheckUploadExists,
		&conditionalStepDownload{
			condition: func() bool {
				_, ok := state.Get("iso_path").(string)
				return !ok
			},
			step: &commonsteps.StepDownload{
				Checksum:    b.config.ISOChecksum,
				Description: "ISO",
				Extension:   b.config.TargetExtension,
				ResultKey:   "iso_path",
				TargetPath:  b.config.TargetPath,
				Url:         b.config.ISOUrls,
			},
		},
		&common.StepOutputDir{
			Force:        b.config.PackerForce,
			OutputConfig: &b.config.OutputConfig,
			RemoteType:   b.config.RemoteType,
			VMName:       b.config.VMName,
		},
		&commonsteps.StepCreateFloppy{
			Files:       b.config.FloppyConfig.FloppyFiles,
			Directories: b.config.FloppyConfig.FloppyDirectories,
			Content:     b.config.FloppyConfig.FloppyContent,
			Label:       b.config.FloppyConfig.FloppyLabel,
		},
		&commonsteps.StepCreateCD{
			Files:   b.config.CDConfig.CDFiles,
			Content: b.config.CDConfig.CDContent,
			Label:   b.config.CDConfig.CDLabel,
		},
		&common.StepRemoteUpload{
			Key:       "floppy_path",
			Message:   "Uploading floppy content to remote cache...",
			DoCleanup: true,
			Checksum:  "none",
		},
		&common.StepRemoteUpload{
			Key:       "cd_path",
			Message:   "Uploading CD content to remote cache...",
			DoCleanup: true,
			Checksum:  "none",
		},
		&common.StepRemoteUpload{
			Key:       "iso_path",
			Message:   "Uploading ISO file to remote cache...",
			DoCleanup: b.config.DriverConfig.CleanUpRemoteCache,
			Checksum:  b.config.ISOChecksum,
		},
		&common.StepCreateDisks{
			OutputDir:          &b.config.OutputDir,
			CreateMainDisk:     true,
			DiskName:           b.config.DiskName,
			MainDiskSize:       b.config.DiskSize,
			AdditionalDiskSize: b.config.AdditionalDiskSize,
			DiskAdapterType:    b.config.DiskAdapterType,
			DiskTypeId:         b.config.DiskTypeId,
		},
		&stepCreateVMX{},
		&common.StepConfigureVMX{
			CustomData:       b.config.VMXData,
			VMName:           b.config.VMName,
			DisplayName:      b.config.VMXDisplayName,
			DiskAdapterType:  b.config.DiskAdapterType,
			CDROMAdapterType: b.config.CdromAdapterType,
		},
		&common.StepSuppressMessages{},
		&common.StepHTTPIPDiscover{},
		commonsteps.HTTPServerFromHTTPConfig(&b.config.HTTPConfig),
		multistep.If(b.config.Comm.Type == "ssh", &communicator.StepSSHKeyGen{
			CommConf:            &b.config.Comm,
			SSHTemporaryKeyPair: b.config.Comm.SSHTemporaryKeyPair,
		}),
		&common.StepConfigureVNC{
			Enabled:            !b.config.DisableVNC && !b.config.VNCOverWebsocket,
			VNCBindAddress:     b.config.VNCBindAddress,
			VNCPortMin:         b.config.VNCPortMin,
			VNCPortMax:         b.config.VNCPortMax,
			VNCDisablePassword: b.config.VNCDisablePassword,
		},
		&common.StepRegister{
			Format:         b.config.Format,
			KeepRegistered: b.config.KeepRegistered,
			SkipExport:     b.config.SkipExport,
		},
		&common.StepRun{
			DurationBeforeStop: 5 * time.Second,
			Headless:           b.config.Headless,
		},
		&common.StepVNCConnect{
			VNCEnabled:         !b.config.DisableVNC,
			VNCOverWebsocket:   b.config.VNCOverWebsocket,
			InsecureConnection: b.config.InsecureConnection,
			DriverConfig:       &b.config.DriverConfig,
		},
		&common.StepVNCBootCommand{
			Config: b.config.VNCConfig,
			VMName: b.config.VMName,
			Ctx:    b.config.ctx,
			Comm:   &b.config.Comm,
		},
		&communicator.StepConnect{
			Config:    &b.config.SSHConfig.Comm,
			Host:      driver.CommHost,
			SSHConfig: b.config.SSHConfig.Comm.SSHConfigFunc(),
		},
		&common.StepUploadTools{
			RemoteType:        b.config.RemoteType,
			ToolsUploadFlavor: b.config.ToolsUploadFlavor,
			ToolsUploadPath:   b.config.ToolsUploadPath,
			Ctx:               b.config.ctx,
		},
		&commonsteps.StepProvision{},
		&commonsteps.StepCleanupTempKeys{
			Comm: &b.config.SSHConfig.Comm,
		},
		&common.StepShutdown{
			Command: b.config.ShutdownCommand,
			Timeout: b.config.ShutdownTimeout,
		},
		&common.StepCleanFiles{},
		&common.StepCompactDisk{
			Skip: b.config.SkipCompaction,
		},
		&common.StepConfigureVMX{
			CustomData:  b.config.VMXDataPost,
			SkipDevices: true,
			VMName:      b.config.VMName,
			DisplayName: b.config.VMXDisplayName,
		},
		&common.StepCleanVMX{
			RemoveEthernetInterfaces: b.config.VMXConfig.VMXRemoveEthernet,
			VNCEnabled:               !b.config.DisableVNC,
		},
		&common.StepCreateSnapshot{
			SnapshotName: &b.config.SnapshotName,
		},
		&common.StepUploadVMX{
			RemoteType: b.config.RemoteType,
		},
		&common.StepExport{
			Format:         b.config.Format,
			SkipExport:     b.config.SkipExport,
			VMName:         b.config.VMName,
			OVFToolOptions: b.config.OVFToolOptions,
			OutputDir:      &b.config.OutputConfig.OutputDir,
		},
	}

	// Create the runner and run the steps.
	b.runner = commonsteps.NewRunnerWithPauseFn(steps, b.config.PackerConfig, ui, state)
	b.runner.Run(ctx, state)

	// If there was an error, return the error and stop.
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If cancelled, return an error and stop.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("build was cancelled")
	}

	// If halted, return an error and stop.
	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("build was halted")
	}

	// Compile the artifact list.
	exportOutputPath := state.Get("export_output_path").(string) // set in StepOutputDir
	return common.NewArtifact(b.config.RemoteType, b.config.Format, exportOutputPath,
		b.config.VMName, b.config.SkipExport, b.config.KeepRegistered, state)
}
