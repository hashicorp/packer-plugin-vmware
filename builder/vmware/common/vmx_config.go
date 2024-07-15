// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type VMXConfig struct {
	// Key-value pairs that will be inserted into the virtual machine `.vmx`
	// file **before** the virtual machine is started. This is useful for
	// setting advanced properties that are not supported by the plugin.
	//
	// ~> **Note**: This option is intended for advanced users who understand
	// the ramifications of making changes to the `.vmx` file. This option is
	// not necessary for most users.
	VMXData map[string]string `mapstructure:"vmx_data" required:"false"`
	// Key-value pairs that will be inserted into the virtual machine `.vmx`
	// file **after** the virtual machine is started. This is useful for setting
	// advanced properties that are not supported by the plugin.
	//
	// ~> **Note**: This option is intended for advanced users who understand
	// the ramifications of making changes to the `.vmx` file. This option is
	// not necessary for most users.
	VMXDataPost map[string]string `mapstructure:"vmx_data_post" required:"false"`
	// Remove all network adapters from virtual machine `.vmx` file after the
	// virtual machine build is complete. Defaults to `false`.
	//
	// ~> **Note**: This option is useful when building Vagrant boxes since
	// Vagrant will create interfaces when provisioning a box.
	VMXRemoveEthernet bool `mapstructure:"vmx_remove_ethernet_interfaces" required:"false"`
	// The inventory display name for the virtual machine. If set, the value
	// provided will override any value set in the `vmx_data` option or in the
	// `.vmx` file. This option is useful if you are chaining builds and want to
	// ensure that the display name of each step in the chain is unique.
	VMXDisplayName string `mapstructure:"display_name" required:"false"`
}

func (c *VMXConfig) Prepare(ctx *interpolate.Context) []error {
	return nil
}
