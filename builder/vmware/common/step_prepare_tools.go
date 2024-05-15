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
	RemoteType        string
	ToolsUploadFlavor string
	ToolsSourcePath   string
}

func (c *StepPrepareTools) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)

	if c.RemoteType == "esx5" {
		return multistep.ActionContinue
	}

	if c.ToolsUploadFlavor == "" && c.ToolsSourcePath == "" {
		return multistep.ActionContinue
	}

	path := c.ToolsSourcePath
	if path == "" {
		path = driver.ToolsIsoPath(c.ToolsUploadFlavor)
	}

	if _, err := os.Stat(path); err != nil {
		state.Put("error", fmt.Errorf("error finding vmware tools for %q: %s", c.ToolsUploadFlavor, err))
		return multistep.ActionHalt
	}

	state.Put("tools_upload_source", path)
	return multistep.ActionContinue
}

func (c *StepPrepareTools) Cleanup(multistep.StateBag) {}
