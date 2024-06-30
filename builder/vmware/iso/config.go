// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package iso

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/shutdowncommand"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	vmwcommon "github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

// Reference: https://knowledge.broadcom.com/external/article?articleNumber=315655
const minimumHardwareVersion = 13
const defaultHardwareVersion = 19

type Config struct {
	common.PackerConfig            `mapstructure:",squash"`
	commonsteps.HTTPConfig         `mapstructure:",squash"`
	commonsteps.ISOConfig          `mapstructure:",squash"`
	commonsteps.FloppyConfig       `mapstructure:",squash"`
	commonsteps.CDConfig           `mapstructure:",squash"`
	bootcommand.VNCConfig          `mapstructure:",squash"`
	vmwcommon.DriverConfig         `mapstructure:",squash"`
	vmwcommon.HWConfig             `mapstructure:",squash"`
	vmwcommon.OutputConfig         `mapstructure:",squash"`
	vmwcommon.RunConfig            `mapstructure:",squash"`
	shutdowncommand.ShutdownConfig `mapstructure:",squash"`
	vmwcommon.SSHConfig            `mapstructure:",squash"`
	vmwcommon.ToolsConfig          `mapstructure:",squash"`
	vmwcommon.VMXConfig            `mapstructure:",squash"`
	vmwcommon.ExportConfig         `mapstructure:",squash"`
	vmwcommon.DiskConfig           `mapstructure:",squash"`
	// The size of the disk in megabytes. The builder uses expandable virtual
	// hard disks. The file that backs the virtual disk will only grow as needed
	// up to this size. Default is 40000 (~40 GB).
	DiskSize uint `mapstructure:"disk_size" required:"false"`
	// The type of controller to use for the CD-ROM device.
	// Allowed values are `ide`, `sata`, and `scsi`.
	CdromAdapterType string `mapstructure:"cdrom_adapter_type" required:"false"`
	// The guest operating system identifier for the virtual machine.
	// Defaults to `other`.
	GuestOSType string `mapstructure:"guest_os_type" required:"false"`
	// The virtual machine hardware version. Refer to [KB 315655](https://knowledge.broadcom.com/external/article?articleNumber=315655)
	// for more information on supported virtual hardware versions.
	// Minimum is 13. Default is 19.
	Version int `mapstructure:"version" required:"false"`
	// The name of the virtual machine. This represents the name of the virtual
	// machine `.vmx` configuration file without the file extension.
	// Default is `packer-<BUILDNAME>`, where `<BUILDNAME>` is the name of the
	// build.
	VMName string `mapstructure:"vm_name" required:"false"`
	// The path to a [configuration template](/packer/docs/templates/legacy_json_templates/engine)
	// for defining the contents of a virtual machine `.vmx` configuration file
	// for a virtual disk. Template variables `{{ .DiskType }}`, `{{ .DiskUnit }}`,
	// `{{ .DiskName }}`, and `{{ .DiskNumber }}` are available for use within
	// the template.
	//
	// ~> **Note:** This option is intended for advanced users, as incorrect
	// configurations can lead to non-functional virtual machines.
	VMXDiskTemplatePath string `mapstructure:"vmx_disk_template_path"`
	// The path to a [configuration template](/packer/docs/templates/legacy_json_templates/engine)
	// for defining the contents of a virtual machine `.vmx` configuration file.
	//
	// ~> **Note:** This option is intended for advanced users, as incorrect
	// configurations can lead to non-functional virtual machines. For simpler
	// modifications of the virtual machine`.vmx` configuration file, consider
	// using `vmx_data` option.
	VMXTemplatePath string `mapstructure:"vmx_template_path" required:"false"`
	// The name of the virtual machine snapshot to be created.
	// If this field is left empty, no snapshot will be created.
	SnapshotName string `mapstructure:"snapshot_name" required:"false"`
	// Enable virtual hardware-assisted virtualization for the virtual machine.
	// Defaults to `false`.
	HardwareAssistedVirtualization bool `mapstructure:"vhv_enabled" required:"false"`

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

	// Accumulate any errors and warnings
	var warnings []string
	var errs *packersdk.MultiError

	runConfigWarnings, runConfigErrs := c.RunConfig.Prepare(&c.ctx, &c.DriverConfig)
	warnings = append(warnings, runConfigWarnings...)
	errs = packersdk.MultiErrorAppend(errs, runConfigErrs...)
	isoWarnings, isoErrs := c.ISOConfig.Prepare(&c.ctx)
	warnings = append(warnings, isoWarnings...)
	errs = packersdk.MultiErrorAppend(errs, isoErrs...)
	errs = packersdk.MultiErrorAppend(errs, c.HTTPConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.HWConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.OutputConfig.Prepare(&c.ctx, &c.PackerConfig)...)
	errs = packersdk.MultiErrorAppend(errs, c.DriverConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ShutdownConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.SSHConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ToolsConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.CDConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.VNCConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.VMXConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.FloppyConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.ExportConfig.Prepare(&c.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, c.DiskConfig.Prepare(&c.ctx)...)

	if c.DiskSize == 0 {
		c.DiskSize = 40000
	}

	if c.DiskTypeId == "" {
		// Default is growable virtual disk split in 2GB files.
		c.DiskTypeId = "1"

		if c.RemoteType == "esx5" {
			c.DiskTypeId = "zeroedthick"
			c.SkipCompaction = true
		}
	}

	if c.RemoteType == "esx5" {
		if c.DiskTypeId != "thin" && !c.SkipCompaction {
			errs = packersdk.MultiErrorAppend(
				errs, fmt.Errorf("skip_compaction must be 'true' for disk_type_id: %s", c.DiskTypeId))
		}
	}

	if c.GuestOSType == "" {
		c.GuestOSType = "other"
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf("packer-%s", c.PackerBuildName)
	}

	if c.Version == 0 {
		c.Version = defaultHardwareVersion
	} else if c.Version < minimumHardwareVersion {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("invalid 'version' %d, minimum hardware version: %d", c.Version, minimumHardwareVersion))
	}

	if c.VMXTemplatePath != "" {
		if err := c.validateVMXTemplatePath(); err != nil {
			errs = packersdk.MultiErrorAppend(
				errs, fmt.Errorf("vmx_template_path is invalid: %s", err))
		}
	} else {
		warn := c.checkForVMXTemplateAndVMXDataCollisions()
		if warn != "" {
			warnings = append(warnings, warn)
		}
	}

	if c.HWConfig.Network == "" {
		c.HWConfig.Network = "nat"
	}

	if c.Format == "" {
		if c.RemoteType == "" {
			c.Format = "vmx"
		} else {
			c.Format = "ovf"
		}
	}

	if c.RemoteType == "" {
		if c.Format == "vmx" {
			// if we're building locally and want a vmx, there's nothing to export.
			// Set skip export flag here to keep the export step from attempting
			// an unneded export
			c.SkipExport = true
		}
		if c.Headless && c.DisableVNC {
			warnings = append(warnings,
				"Headless mode uses VNC to retrieve output. Since VNC has been disabled,\n"+
					"you won't be able to see any output.")
		}
	}

	err = c.DriverConfig.Validate(c.SkipExport)
	if err != nil {
		errs = packersdk.MultiErrorAppend(errs, err)
	}

	if c.CdromAdapterType != "" {
		c.CdromAdapterType = strings.ToLower(c.CdromAdapterType)
		if c.CdromAdapterType != "ide" && c.CdromAdapterType != "sata" && c.CdromAdapterType != "scsi" {
			errs = packersdk.MultiErrorAppend(errs,
				fmt.Errorf("cdrom_adapter_type must be one of ide, sata, or scsi"))
		}
	}

	// Warnings
	if c.ShutdownCommand == "" {
		warnings = append(warnings,
			"A shutdown_command was not specified. Without a shutdown command, Packer\n"+
				"will forcibly halt the virtual machine, which may result in data loss.")
	}

	if errs != nil && len(errs.Errors) > 0 {
		return warnings, errs
	}

	return warnings, nil
}

func (c *Config) checkForVMXTemplateAndVMXDataCollisions() string {
	if c.VMXTemplatePath != "" {
		return ""
	}

	var overridden []string
	tplLines := strings.Split(DefaultVMXTemplate, "\n")
	tplLines = append(tplLines,
		fmt.Sprintf("%s0:0.present", strings.ToLower(c.DiskAdapterType)),
		fmt.Sprintf("%s0:0.fileName", strings.ToLower(c.DiskAdapterType)),
		fmt.Sprintf("%s0:0.deviceType", strings.ToLower(c.DiskAdapterType)),
		fmt.Sprintf("%s0:1.present", strings.ToLower(c.DiskAdapterType)),
		fmt.Sprintf("%s0:1.fileName", strings.ToLower(c.DiskAdapterType)),
		fmt.Sprintf("%s0:1.deviceType", strings.ToLower(c.DiskAdapterType)),
	)

	for _, line := range tplLines {
		if strings.Contains(line, `{{`) {
			key := line[:strings.Index(line, " =")]
			if _, ok := c.VMXData[key]; ok {
				overridden = append(overridden, key)
			}
		}
	}

	if len(overridden) > 0 {
		warnings := fmt.Sprintf("Your vmx data contains the following "+
			"variable(s), which Packer normally sets when it generates its "+
			"own default vmx template. This may cause your build to fail or "+
			"behave unpredictably: %s", strings.Join(overridden, ", "))
		return warnings
	}
	return ""
}

// Make sure custom vmx template exists and that data can be read from it
func (c *Config) validateVMXTemplatePath() error {
	f, err := os.Open(c.VMXTemplatePath)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	return interpolate.Validate(string(data), &c.ctx)
}
