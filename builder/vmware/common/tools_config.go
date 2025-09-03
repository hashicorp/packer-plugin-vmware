// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type ToolsConfig struct {
	// The flavor of VMware Tools to upload into the virtual machine based on
	// the guest operating system. Allowed values are `darwin` (macOS), `linux`,
	// and `windows`. Default is empty and no version will be uploaded.
	//
	// ~> **Note:** When `tools_upload_flavor` is specified without `tools_mode`,
	// the plugin automatically defaults to `tools_mode="upload"` for v1
	// backward compatibility.
	ToolsUploadFlavor string `mapstructure:"tools_upload_flavor" required:"false"`
	// The path in the virtual machine to upload the VMware Tools ISO. This only
	// takes effect if `tools_upload_flavor` is non-empty. This is a [configuration
	// template](/packer/docs/templates/legacy_json_templates/engine) that has a
	// single valid variable: `Flavor`, which will be the value of
	// `tools_upload_flavor`. By default, the upload path is set to
	// `{{.Flavor}}.iso`.
	ToolsUploadPath string `mapstructure:"tools_upload_path" required:"false"`
	// The local path on your machine to the VMware Tools ISO file.
	//
	// ~> **Note:** If not set, but the `tools_upload_flavor` is set, the plugin
	// will load the VMware Tools ISO from the product installation defaults.
	ToolsSourcePath string `mapstructure:"tools_source_path" required:"false"`
	// The mode for providing VMware Tools to the virtual machine. Allowed
	// values are:
	// - `upload`: uploads VMware Tools ISO to the virtual machine during the
	// build
	// - `attach`: attaches the VMware Tools ISO to the virtual machine as
	// CD-ROM device during the build and removes the device upon build
	// completion.
	// - `disable`: no VMware Tools ISO is provided to the virtual machine.
	//
	// ~> **Note:** Automatically defaults to `upload` when `tools_upload_flavor`
	// is specified.
	ToolsMode string `mapstructure:"tools_mode" required:"false"`
}

// Prepare validates and sets default values for the VMware Tools configuration.
func (c *ToolsConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	// Handle backward compatibility: if tools_upload_flavor is set but
	// tools_mode is not, default to upload mode.
	if c.ToolsMode == "" && c.ToolsUploadFlavor != "" {
		c.ToolsMode = toolsModeUpload
	}

	// Validate tools_mode if specified.
	if c.ToolsMode != "" && !slices.Contains(allowedToolsModeValues, c.ToolsMode) {
		errs = append(errs, fmt.Errorf("invalid 'tools_mode' specified: %s; must be one of %s", c.ToolsMode, strings.Join(allowedToolsModeValues, ", ")))
	}

	// Validate conflicting configurations.
	if c.ToolsUploadFlavor != "" && c.ToolsSourcePath != "" {
		errs = append(errs, errors.New("'tools_upload_flavor' and 'tools_source_path' cannot both be specified"))
		return errs
	}

	// Skip validation if tools are disabled.
	if c.ToolsMode == toolsModeDisable {
		return errs
	}

	// Handle attach mode validation.
	if c.ToolsMode == toolsModeAttach {
		if c.ToolsSourcePath == "" && c.ToolsUploadFlavor == "" {
			errs = append(errs, errors.New("'tools_source_path' or 'tools_upload_flavor' must be specified when 'tools_mode=\"attach\"'"))
		}

		// Validate tools source path exists when specified for attach mode.
		if c.ToolsSourcePath != "" {
			if _, err := os.Stat(c.ToolsSourcePath); err != nil {
				if os.IsNotExist(err) {
					errs = append(errs, fmt.Errorf("tools source path does not exist: %s", c.ToolsSourcePath))
				} else {
					errs = append(errs, fmt.Errorf("tools source path is not accessible: %s", err))
				}
			}
		}

		// Validate tools_upload_flavor if specified for attach mode.
		if c.ToolsUploadFlavor != "" && !slices.Contains(allowedToolsFlavorValues, c.ToolsUploadFlavor) {
			errs = append(errs, fmt.Errorf("invalid 'tools_upload_flavor' specified: %s; must be one of %s", c.ToolsUploadFlavor, strings.Join(allowedToolsFlavorValues, ", ")))
		}

		return errs
	}

	// Handle upload mode validation.
	if c.ToolsUploadPath != "" {
		return errs
	}

	if c.ToolsSourcePath != "" && c.ToolsUploadFlavor == "" {
		errs = append(errs, errors.New("provide either 'tools_upload_flavor' or 'tools_upload_path' with 'tools_source_path'"))
	} else if c.ToolsUploadFlavor != "" && !slices.Contains(allowedToolsFlavorValues, c.ToolsUploadFlavor) {
		errs = append(errs, fmt.Errorf("invalid 'tools_upload_flavor' specified: %s; must be one of %s", c.ToolsUploadFlavor, strings.Join(allowedToolsFlavorValues, ", ")))
	}

	// Set default upload path based on configuration.
	if c.ToolsUploadPath == "" {
		if c.ToolsUploadFlavor != "" {
			// When flavor is specified, use template for dynamic path resolution.
			c.ToolsUploadPath = "{{ .Flavor }}.iso"
		} else if c.ToolsMode == "" {
			// Default behavior when no tools configuration is provided.
			c.ToolsUploadPath = "{{ .Flavor }}.iso"
		}
	}

	return errs
}
