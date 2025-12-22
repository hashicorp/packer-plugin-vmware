// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type OutputConfig struct {
	// This is the path on your local machine to the directory where the
	// resulting virtual machine will be created. This may be relative or
	// absolute. If relative, the path is relative to the working directory
	// when Packer is run.
	//
	// By default, this is `output-BUILDNAME` where `BUILDNAME` is the name of
	// the build.
	//
	// ~> **Note:** This directory must not exist or be empty before running the
	// build.
	OutputDir string `mapstructure:"output_directory" required:"false"`
}

// Prepare validates and sets default values for the output configuration.
func (c *OutputConfig) Prepare(ctx *interpolate.Context, pc *common.PackerConfig) []error {
	if c.OutputDir == "" {
		c.OutputDir = fmt.Sprintf("output-%s", pc.PackerBuildName)
	}

	return nil
}
