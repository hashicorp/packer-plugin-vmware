// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package iso

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"

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
	// Defaults to `other-64` on amd64 and `arm-other-64` on arm64.
	GuestOSType string `mapstructure:"guest_os_type" required:"false"`
	// The virtual machine hardware version. Refer to [KB 315655](https://knowledge.broadcom.com/external/article?articleNumber=315655)
	// for more information on supported virtual hardware versions.
	// Default is 21. Minimum is 19.
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

// Prepare validates and sets default values for the ISO builder configuration.
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
		c.DiskSize = vmwcommon.DefaultDiskSize
	}

	if c.DiskTypeId == "" {
		c.DiskTypeId = vmwcommon.DefaultDiskType
	}

	if c.GuestOSType == "" {
		switch runtime.GOARCH {
		case "arm64":
			c.GuestOSType = vmwcommon.DefaultGuestOsTypeArm64
		case "amd64":
			c.GuestOSType = vmwcommon.DefaultGuestOsTypeAmd64
		default:
			c.GuestOSType = vmwcommon.FallbackGuestOsType
			warnings = append(warnings,
				fmt.Sprintf("[WARN] Failed to recognize the runtime architecture %q. Defaulting to %q.",
					runtime.GOARCH, vmwcommon.FallbackGuestOsType))
		}
	}

	if c.VMName == "" {
		c.VMName = fmt.Sprintf("%s-%s", vmwcommon.DefaultNamePrefix, c.PackerBuildName)
	}

	if c.Version == 0 {
		c.Version = vmwcommon.DefaultHardwareVersion
	} else if c.Version < vmwcommon.MinimumHardwareVersion {
		errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("invalid 'version' %d, minimum hardware version: %d", c.Version, vmwcommon.MinimumHardwareVersion))
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

	if c.Network == "" {
		c.Network = vmwcommon.DefaultNetworkType
	}

	if c.Format == "" {
		c.Format = vmwcommon.ExportFormatVmx
	}

	if c.Format == vmwcommon.ExportFormatVmx {
		// Set skip an export flag to avoid an unneeded export.
		c.SkipExport = true
	}
	if c.Headless && c.DisableVNC {
		warnings = append(warnings,
			"Headless mode uses VNC to retrieve output. Since VNC has been disabled,\n"+
				"you won't be able to see any output.")
	}

	err = c.Validate(c.SkipExport)
	if err != nil {
		errs = packersdk.MultiErrorAppend(errs, err)
	}

	if c.CdromAdapterType != "" {
		c.CdromAdapterType = strings.ToLower(c.CdromAdapterType)
		if !slices.Contains(vmwcommon.AllowedCdromAdapterTypes, c.CdromAdapterType) {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("invalid 'cdrom_adapter_type' specified: %s; must be one of %s", c.CdromAdapterType, strings.Join(vmwcommon.AllowedCdromAdapterTypes, ", ")))
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

// checkForVMXTemplateAndVMXDataCollisions detects conflicts between VMX template and custom VMX data.
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

// validateVMXTemplatePath ensures the custom VMX template file exists and is readable.
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
