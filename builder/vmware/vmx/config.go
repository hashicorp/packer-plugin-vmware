// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package vmx

import (
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/shutdowncommand"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	vmwcommon "github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

type Config struct {
	common.PackerConfig            `mapstructure:",squash"`
	commonsteps.HTTPConfig         `mapstructure:",squash"`
	commonsteps.FloppyConfig       `mapstructure:",squash"`
	bootcommand.VNCConfig          `mapstructure:",squash"`
	commonsteps.CDConfig           `mapstructure:",squash"`
	vmwcommon.DriverConfig         `mapstructure:",squash"`
	vmwcommon.OutputConfig         `mapstructure:",squash"`
	vmwcommon.RunConfig            `mapstructure:",squash"`
	shutdowncommand.ShutdownConfig `mapstructure:",squash"`
	vmwcommon.SSHConfig            `mapstructure:",squash"`
	vmwcommon.ToolsConfig          `mapstructure:",squash"`
	vmwcommon.VMXConfig            `mapstructure:",squash"`
	vmwcommon.ExportConfig         `mapstructure:",squash"`
	vmwcommon.DiskConfig           `mapstructure:",squash"`
	// By default, the plugin creates a 'full' clone of the virtual machine
	// specified in `source_path`. The resultant virtual machine is fully
	// independent of the parent it was cloned from.
	//
	// Setting linked to true instead causes the plugin to create the virtual
	// machine as a linked clone. Linked clones use and require ongoing
	// access to the disks of the parent virtual machine. The benefit of a
	// linked clone is that the clones virtual disk is typically very much
	// smaller than would be the case for a full clone. Additionally, the
	// cloned virtual machine can also be created much faster. Creating a
	// linked clone will typically only be of benefit in some advanced build
	// scenarios. Most users will wish to create a full clone instead.
	// Defaults to `false`.
	Linked bool `mapstructure:"linked" required:"false"`
	// The name of an existing snapshot to which the builder shall attach the
	// virtual machine before powering on. If no snapshot is specified the
	// virtual machine is started from its current state.  Default to
	// `null/empty`.
	AttachSnapshot string `mapstructure:"attach_snapshot" required:"false"`
	// Path to the source `.vmx` file to clone. If `remote_type` is enabled
	// then this specifies a path on the `remote_host`.
	SourcePath string `mapstructure:"source_path" required:"true"`
	// This is the name of the `.vmx` file for the virtual machine, without
	// the file extension. By default, this is `packer-BUILDNAME`, where
	// `BUILDNAME` is the name of the build.
	VMName string `mapstructure:"vm_name" required:"false"`
	// This is the name of the initial snapshot created after provisioning and
	// cleanup. If blank, no snapshot is created.
	SnapshotName string `mapstructure:"snapshot_name" required:"false"`

	ctx interpolate.Context
}

func (c *Config) Prepare(raws ...interface{}) ([]string, error) {
	err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
				"tools_upload_path",
			},
		},
	}, raws...)
	if err != nil {
		return nil, err
	}

	// Defaults
	if c.VMName == "" {
		c.VMName = fmt.Sprintf(
			"packer-%s-%d", c.PackerBuildName, interpolate.InitTime.Unix())
	}

	// Accumulate any errors and warnings
	var warnings []string
	var errs *packersdk.MultiError

	runConfigWarnings, runConfigErrs := c.RunConfig.Prepare(&c.ctx, &c.DriverConfig)
	warnings = append(warnings, runConfigWarnings...)
	errs = packersdk.MultiErrorAppend(errs, runConfigErrs...)
	errs = packersdk.MultiErrorAppend(errs, c.DriverConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.HTTPConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.OutputConfig.Prepare(&c.ctx, &c.PackerConfig)...)
	errs = packersdk.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.SSHConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ToolsConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.FloppyConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.CDConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.VNCConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.VNCConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.DiskConfig.Prepare(&c.ctx)...)

	if c.RemoteType == "" {
		if c.SourcePath == "" {
			errs = packersdk.MultiErrorAppend(errs, errors.New("'source_path' is blank, but is required"))
		} else {
			if _, err := os.Stat(c.SourcePath); err != nil {
				errs = packersdk.MultiErrorAppend(errs,
					fmt.Errorf("source_path is invalid: %s", err))
			}
		}
		if c.Headless && c.DisableVNC {
			warnings = append(warnings,
				"Headless mode uses VNC to retrieve output. Since VNC has been disabled,\n"+
					"you won't be able to see any output.")
		}
	}

	if c.DiskTypeId == "" {
		// Default is growable virtual disk split in 2GB files.
		c.DiskTypeId = "1"

		if c.RemoteType == "esxi" {
			c.DiskTypeId = "zeroedthick"
		}
	}

	if c.Format == "" {
		if c.RemoteType == "" {
			c.Format = "vmx"
		} else {
			c.Format = "ovf"
		}
	}

	if c.RemoteType == "" && c.Format == "vmx" {
		// if we're building locally and want a vmx, there's nothing to export.
		// Set skip export flag here to keep the export step from attempting
		// an unneeded export
		c.SkipExport = true
	}

	err = c.Validate(c.SkipExport)
	if err != nil {
		errs = packersdk.MultiErrorAppend(errs, err)
	}

	if c.ShutdownCommand == "" {
		warnings = append(warnings,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	// Check for any errors.
	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}
