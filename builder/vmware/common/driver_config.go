// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"os"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type DriverConfig struct {
	// The installation path of the VMware Fusion application. Defaults to
	// `/Applications/VMware Fusion.app`
	//
	// ~> **Note:** This is only required if you are using VMware Fusion as a
	// desktop hypervisor and have installed it in a non-default location.
	FusionAppPath string `mapstructure:"fusion_app_path" required:"false"`
}

func (c *DriverConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	if c.FusionAppPath == "" {
		c.FusionAppPath = os.Getenv("FUSION_APP_PATH")
	}

	if c.FusionAppPath == "" {
		c.FusionAppPath = "/Applications/VMware Fusion.app"
	}

	return errs
}

func (c *DriverConfig) Validate(SkipExport bool) error {
	if SkipExport {
		return nil
	}

	return nil
}
