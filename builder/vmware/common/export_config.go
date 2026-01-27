// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type ExportConfig struct {
	// The output format of the exported virtual machine. Allowed values are
	// `ova`, `ovf`, or `vmx`. Defaults to `vmx`.
	//
	// ~> **Note:** Ensure VMware OVF Tool is installed. For the latest version,
	// visit [VMware OVF Tool](https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest).
	//
	// ~> **Note:** The plugin will create a `.vmx` and supporting files in the
	// output directory and will then export the virtual machine to the specified
	// format. These files are **not** automatically cleaned up after the export process.
	Format string `mapstructure:"format" required:"false"`
	// Additional command-line arguments to send to VMware OVF Tool during the
	// export process. Each string in the array represents a separate
	// command-line argument.
	//
	// ~> **Important:** The plugin does not preset any VMware OVF Tool options
	// by default.
	//
	// ~> **Note:** Ensure VMware OVF Tool is installed. For the latest version,
	// visit [VMware OVF Tool](https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest).
	OVFToolOptions []string `mapstructure:"ovftool_options" required:"false"`
	// Skips the export of the virtual machine. This is useful if the build
	// output is not the resultant image, but created inside the virtual
	// machine. This is useful for debugging purposes. Defaults to `false`.
	SkipExport bool `mapstructure:"skip_export" required:"false"`
	// At the end of the build process, the plugin defragments and compacts the
	// disks using `vmware-vdiskmanager`. In some cases, this process may result
	// in slightly larger disk sizes. If this occurs, you can opt to skip the
	// disk compaction step by using this setting. Defaults to `false`.
	SkipCompaction bool `mapstructure:"skip_compaction" required:"false"`
}

// Prepare validates and sets default values for the export configuration.
func (c *ExportConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	if (c.Format != "") && (!slices.Contains(allowedExportFormats, c.Format)) {
		errs = append(errs, fmt.Errorf("invalid 'format' type specified: %s; must be one of %s", c.Format, strings.Join(allowedExportFormats, ", ")))
	}

	return errs
}
