// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type DiskConfig struct {
	// The size(s) of additional virtual hard disks in MB. If not specified,
	// the virtual machine will contain only a primary hard disk.
	AdditionalDiskSize []uint `mapstructure:"disk_additional_size" required:"false"`
	// The adapter type for additional virtual disk(s). Available options
	//  are `ide`, `sata`, `nvme`, or `scsi`.
	//
	// ~> **Note:** When specifying `scsi` as the adapter type, the default
	// adapter type is set to `lsilogic`. If another option is specified, the
	// plugin will assume it is a `scsi` interface of that specified type.
	//
	// ~> **Note:** This option is intended for advanced users.
	DiskAdapterType string `mapstructure:"disk_adapter_type" required:"false"`
	// The filename for the virtual disk to create _without_ the `.vmdk`
	// extension. Defaults to `disk`.
	DiskName string `mapstructure:"vmdk_name" required:"false"`
	// The type of virtual disk to create. The available options include:
	//
	//   | Type ID | Description                                                             |
	//   |---------|-------------------------------------------------------------------------|
	//   | `0`     | Growable virtual disk contained in a single file (monolithic sparse).   |
	//   | `1`     | Growable virtual disk split into 2GB files (split sparse).              |
	//   | `2`     | Preallocated virtual disk contained in a single file (monolithic flat). |
	//   | `3`     | Preallocated virtual disk split into 2GB files (split flat).            |
	//   | `4`     | Preallocated virtual disk compatible with ESXi (VMFS flat).             |
	//   | `5`     | Compressed disk optimized for streaming.                                |
	//
	//   Defaults to `1`
	//
	//   ~> **Note:** Set `skip_compaction` to `true` when using `zeroedthick`
	//   or `eagerzeroedthick` due to default disk compaction behavior.
	//
	// ~> **Note:** This option is intended for advanced users.
	DiskTypeId string `mapstructure:"disk_type_id" required:"false"`
}

func (c *DiskConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	if c.DiskName == "" {
		c.DiskName = defaultDiskName
	}

	if c.DiskAdapterType == "" {
		c.DiskAdapterType = defaultDiskAdapterType
	}

	return errs
}
