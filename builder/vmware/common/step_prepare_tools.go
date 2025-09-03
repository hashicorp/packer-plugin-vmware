// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type StepPrepareTools struct {
	ToolsMode         string
	ToolsUploadFlavor string
	ToolsSourcePath   string
}

// Run executes the tools preparation step, locating and validating VMware Tools for both upload and attach modes.
func (c *StepPrepareTools) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)

	// Skip if tools is disabled.
	if c.ToolsMode == toolsModeDisable {
		return multistep.ActionContinue
	}

	// Skip if no tools configuration is provided.
	if c.ToolsUploadFlavor == "" && c.ToolsSourcePath == "" && c.ToolsMode == "" {
		return multistep.ActionContinue
	}

	// Determine the tools source path.
	path := c.ToolsSourcePath
	if path == "" && c.ToolsUploadFlavor != "" {
		path = driver.ToolsIsoPath(c.ToolsUploadFlavor)
	}

	// Validate that the ISO file exists.
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			state.Put("error", fmt.Errorf("error finding vmware tools for %q: %s", c.ToolsUploadFlavor, err))
			return multistep.ActionHalt
		}
	}

	// Store the tools source path in state for both upload and attach modes.
	if path != "" {
		if c.ToolsMode == toolsModeAttach {
			state.Put("tools_attach_source", path)
		} else {
			// Default to upload mode for backward compatibility.
			state.Put("tools_upload_source", path)
		}
	}

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup after the tools preparation step completes.
func (c *StepPrepareTools) Cleanup(multistep.StateBag) {
	// No-op.
}
