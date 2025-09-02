// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type DriverConfig struct {
	// The installation path of the VMware Fusion application.
	//
	// ~> **Note:** This is only required if you are using VMware Fusion as a
	// desktop hypervisor and have installed it in a non-default location.
	FusionAppPath string `mapstructure:"fusion_app_path" required:"false"`
	// No longer supported.
	//
	// ~> **Important:** VMware ESX is not supported by the plugin as of v2.0.0.
	// Please use the [Packer plugin for VMware vSphere](https://developer.hashicorp.com/packer/integrations/hashicorp/vsphere).
	RemoteType string `mapstructure:"remote_type" required:"false"`
}

// Prepare validates and sets default values for the driver configuration.
func (c *DriverConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	// If the Fusion app path is not set, try to get it from the environment.
	if c.FusionAppPath == "" {
		c.FusionAppPath = os.Getenv(fusionAppPathVariable)
	}

	// If the Fusion app path is still not set, set it to the default.
	if c.FusionAppPath == "" {
		c.FusionAppPath = fusionAppPath
	}

	if c.RemoteType != "" {
		// The use of VMware ESX is no longer supported in the plugin.
		// If a user attempts to use the legacy option, an error is returned with instructions.
		errs = append(errs, fmt.Errorf("remote_type: VMware ESX is not supported by the plugin as of v2.0.0. Please use the Packer plugin for VMware vSphere"))
	}

	return errs
}

// Validate checks the driver configuration for export-specific requirements.
func (c *DriverConfig) Validate(SkipExport bool) error {
	if SkipExport {
		return nil
	}

	return nil
}
