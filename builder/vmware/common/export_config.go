// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type ExportConfig struct {
	// The output format of the exported virtual machine. Allowed values are
	// `ova`, `ovf`, or `vmx`. The default value is `ovf` for remote hypervisor
	// on, and `vmx` for local desktop hypervisors.
	//
	// For builds on a remote hypervisor, `remote_password` must be set when
	// exporting the virtual machine
	//
	// For builds on a local desktop hypervisor, the plugin will create a `.vmx`
	// and export the virtual machine as an `.ovf` or `.ova` file. THe plugin
	// will not delete the `.vmx` and `.vmdk` files. You must manually delete
	// these files if they are no longer needed.
	//
	// ~> **Note:** Ensure VMware OVF Tool version 4.6.0 or later is installed.
	// For the latest version, visit [VMware OVF Tool](https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest).
	Format string `mapstructure:"format" required:"false"`
	// Additional command-line arguments to send to VMware OVF Tool during the
	// export process. Each string in the array represents a separate
	// command-line argument.
	//
	// ~> **Important:** The options `--noSSLVerify`, `--skipManifestCheck`, and
	// `--targetType` are automatically applied by the plugin for remote exports
	// and should not be included in the options. For local OVF/OVA exports,
	// the plugin does not preset any VMware OVF Tool options by default.
	//
	// ~> **Note:** Ensure VMware OVF Tool version 4.6.0 or later is installed.
	// For the latest version, visit [VMware OVF Tool](https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest).
	OVFToolOptions []string `mapstructure:"ovftool_options" required:"false"`
	// Skips the export of the virtual machine. This is useful if the build
	// output is not the resultant image, but created inside the virtual
	// machine. This is useful for debugging purposes. Defaults to `false`.
	// Defaults to `false`.
	SkipExport bool `mapstructure:"skip_export" required:"false"`
	// Determines whether a virtual machine built on a remote hypervisor should
	// remain registered after the build process. Setting this to `true` can be
	// useful if the virtual machine does not need to be exported. Defaults to
	// `false`.
	KeepRegistered bool `mapstructure:"keep_registered" required:"false"`
	// At the end of the build process, the plugin defragments and compacts the
	// disks using `vmware-vdiskmanager` or `vmkfstools` for ESXi environments.
	// In some cases, this process may result in slightly larger disk sizes.
	// If this occurs, you can opt to skip the disk compaction step by using
	// this setting. Defaults to `false`. Defaults to `true` for ESXi when
	// `disk_type_id` is not explicitly defined and `false` otherwise.
	SkipCompaction bool `mapstructure:"skip_compaction" required:"false"`
}

func (c *ExportConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error
	if c.Format != "" {
		if !(c.Format == "ova" || c.Format == "ovf" || c.Format == "vmx") {
			errs = append(
				errs, fmt.Errorf("format must be one of ova, ovf, or vmx"))
		}
	}

	return errs
}
