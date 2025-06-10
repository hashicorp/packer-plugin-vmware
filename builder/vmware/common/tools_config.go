// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// Set the allowed values for the `ToolsUploadFlavor`.
const (
	ToolsFlavorMacOS   = osMacOS
	ToolsFlavorLinux   = osLinux
	ToolsFlavorWindows = osWindows
)

// allowedToolsFlavorValues is a list of allowed values for the
// `ToolsUploadFlavor`.
var allowedToolsFlavorValues = []string{ToolsFlavorMacOS, ToolsFlavorLinux, ToolsFlavorWindows}

type ToolsConfig struct {
	// The flavor of VMware Tools to upload into the virtual machine based on
	// the guest operating system. Allowed values are `darwin` (macOS), `linux`,
	// and `windows`. Default is empty and no version will be uploaded.
	ToolsUploadFlavor string `mapstructure:"tools_upload_flavor" required:"false"`
	// The path in the VM to upload the VMware tools. This only takes effect if
	// `tools_upload_flavor` is non-empty. This is a [configuration
	// template](/packer/docs/templates/legacy_json_templates/engine) that has a
	// single valid variable: `Flavor`, which will be the value of
	// `tools_upload_flavor`. By default, the upload path is set to
	// `{{.Flavor}}.iso`.
	//
	// ~> **Note:** This setting is not used when `remote_type` is `esxi`.
	ToolsUploadPath string `mapstructure:"tools_upload_path" required:"false"`
	// The local path on your machine to the VMware Tools ISO file.
	//
	// ~> **Note:** If not set, but the `tools_upload_flavor` is set, the plugin
	// will load the VMware Tools from the product installation directory.
	ToolsSourcePath string `mapstructure:"tools_source_path" required:"false"`
}

func (c *ToolsConfig) Prepare(ctx *interpolate.Context) []error {
	if c.ToolsUploadPath != "" {
		return nil
	}

	var errs []error

	if c.ToolsSourcePath != "" && c.ToolsUploadFlavor == "" {
		errs = append(errs, errors.New("provide either 'tools_upload_flavor' or 'tools_upload_path' with 'tools_source_path'"))
	} else if c.ToolsUploadFlavor != "" && !slices.Contains(allowedToolsFlavorValues, c.ToolsUploadFlavor) {
		errs = append(errs, fmt.Errorf("invalid 'tools_upload_flavor' specified: %s; must be one of %s", c.ToolsUploadFlavor, strings.Join(allowedToolsFlavorValues, ", ")))
	} else {
		c.ToolsUploadPath = fmt.Sprintf("%s.iso", c.ToolsUploadFlavor)
	}

	if c.ToolsSourcePath == "" {
		c.ToolsUploadPath = "{{ .Flavor }}.iso"
	}

	return errs
}
