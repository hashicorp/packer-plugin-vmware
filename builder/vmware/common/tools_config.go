// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type ToolsConfig struct {
	// The mode for providing VMware Tools to the virtual machine. Must be
	// explicitly specified when using any tools configuration. Allowed values are:
	// - `upload`: Uploads VMware Tools ISO to the virtual machine during the build.
	//   Requires either `tools_upload_flavor` or `tools_source_path` to be specified.
	// - `attach`: Attaches the VMware Tools ISO to the virtual machine as a CD-ROM
	//   device during the build and removes the device upon build completion.
	//   Requires `tools_source_path` to be specified.
	// - `disable`: No VMware Tools ISO is provided to the virtual machine.
	//   Any other tools configuration fields are ignored.
	ToolsMode string `mapstructure:"tools_mode" required:"false"`
	// The absolute local path on your machine to the VMware Tools ISO file.
	// Can be used with `tools_mode` set to `attach` or `upload`. When used with
	// `upload` mode, cannot be used together with `tools_upload_flavor`.
	//
	// Must be a path accessible during the build (e.g., "/path/to/vmware-tools.iso".)
	ToolsSourcePath string `mapstructure:"tools_source_path" required:"false"`
	// The flavor of VMware Tools to upload into the virtual machine based on the
	// guest operating system. Can only be used when `tools_mode` is set to
	// `upload`. Cannot be used together with `tools_source_path`. Allowed
	// values include: `darwin` (macOS), `linux`, and `windows`.
	//
	// The plugin will load the VMware Tools ISO from the desktop hypervisor's
	// default installation directory based on the specified flavor, if available.
	ToolsUploadFlavor string `mapstructure:"tools_upload_flavor" required:"false"`
	// The absolute path in the virtual machine guest operating system where the
	// VMware Tools ISO will be uploaded. Only used when `tools_mode` is set to
	// `upload`. This is a [configuration template](/packer/docs/templates/legacy_json_templates/engine)
	// that has a single valid variable: `Flavor`, which will be the value of
	// `tools_upload_flavor`. Defaults to `{{.Flavor}}.iso` when
	// `tools_upload_flavor` is specified.
	//
	// Must be an absolute path in the guest operating system (e.g., "/tmp/vmware-tools.iso").
	ToolsUploadPath string `mapstructure:"tools_upload_path" required:"false"`
}

// Prepare validates the VMware Tools configuration and sets default values.
// Requires explicit tools_mode specification for any tools configuration.
// Validates mutual exclusivity between tools_upload_flavor and tools_source_path.
// Sets default tools_upload_path when using upload mode with tools_upload_flavor.
// When tools_mode is "disable", all other tools configuration fields are ignored with warnings.
func (c *ToolsConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	if c.ToolsMode != "" && !slices.Contains(allowedToolsModeValues, c.ToolsMode) {
		errs = append(errs, fmt.Errorf("invalid 'tools_mode' specified: %s; must be one of %s", c.ToolsMode, strings.Join(allowedToolsModeValues, ", ")))
	}

	if c.ToolsMode == toolsModeDisable {
		if c.ToolsUploadPath != "" {
			log.Printf("[WARN] 'tools_upload_path' is ignored when 'tools_mode=\"disable\"'")
		}
		if c.ToolsUploadFlavor != "" {
			log.Printf("[WARN] 'tools_upload_flavor' is ignored when 'tools_mode=\"disable\"'")
		}
		if c.ToolsSourcePath != "" {
			log.Printf("[WARN] 'tools_source_path' is ignored when 'tools_mode=\"disable\"'")
		}
		return errs
	}

	if c.ToolsMode == toolsModeAttach {
		if c.ToolsUploadPath != "" {
			errs = append(errs, errors.New("'tools_upload_path' can only be used with 'tools_mode=\"upload\"', not 'tools_mode=\"attach\"'"))
		}

		if c.ToolsUploadFlavor != "" {
			errs = append(errs, errors.New("'tools_upload_flavor' can only be used with 'tools_mode=\"upload\"', not 'tools_mode=\"attach\"'"))
		}

		if c.ToolsSourcePath == "" {
			errs = append(errs, errors.New("'tools_source_path' is required when 'tools_mode=\"attach\"'"))
		}

		if err := c.validateToolsSourcePath(); err != nil {
			errs = append(errs, err)
		}

		return errs
	}

	if c.ToolsMode == toolsModeUpload {
		if c.ToolsUploadFlavor != "" && c.ToolsSourcePath != "" {
			errs = append(errs, errors.New("'tools_upload_flavor' and 'tools_source_path' cannot both be specified - use one or the other"))
		}

		if c.ToolsUploadFlavor == "" && c.ToolsSourcePath == "" {
			errs = append(errs, errors.New("'tools_mode=\"upload\"' requires either 'tools_upload_flavor' or 'tools_source_path'"))
		}

		if c.ToolsUploadFlavor != "" && !slices.Contains(allowedToolsFlavorValues, c.ToolsUploadFlavor) {
			errs = append(errs, fmt.Errorf("invalid 'tools_upload_flavor' specified: %s; must be one of %s", c.ToolsUploadFlavor, strings.Join(allowedToolsFlavorValues, ", ")))
		}

		if err := c.validateToolsSourcePath(); err != nil {
			errs = append(errs, err)
		}

		if c.ToolsUploadPath == "" && c.ToolsUploadFlavor != "" {
			c.ToolsUploadPath = "{{ .Flavor }}.iso"
		}

		return errs
	}

	if (c.ToolsUploadFlavor != "" || c.ToolsSourcePath != "" || c.ToolsUploadPath != "") && c.ToolsMode == "" {
		errs = append(errs, errors.New("'tools_mode' must be explicitly specified when using any tools configuration"))
	}

	return errs
}

// validateToolsSourcePath performs comprehensive validation of the tools source path
func (c *ToolsConfig) validateToolsSourcePath() error {
	if c.ToolsSourcePath == "" {
		return nil
	}

	// Check file existence and accessibility (works for both absolute and relative paths)
	stat, err := os.Stat(c.ToolsSourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("'tools_source_path' does not exist: %s", c.ToolsSourcePath)
		}
		return fmt.Errorf("tools source path is not accessible: %s", err)
	}

	// Check if it's a regular file (not directory)
	if stat.IsDir() {
		return fmt.Errorf("tools source path must be a file, not a directory: %s", c.ToolsSourcePath)
	}

	// Warn if not .iso extension (but don't fail)
	if !strings.HasSuffix(strings.ToLower(c.ToolsSourcePath), ".iso") {
		log.Printf("[WARN] 'tools_source_path' does not have .iso extension, this may cause issues: %s", c.ToolsSourcePath)
	}

	return nil
}
