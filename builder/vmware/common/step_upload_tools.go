// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// toolsUploadPathTemplate contains template data for VMware Tools upload path interpolation.
type toolsUploadPathTemplate struct {
	Flavor string
}

// StepUploadTools represents a step for uploading VMware Tools to the virtual machine.
type StepUploadTools struct {
	ToolsUploadFlavor string
	ToolsUploadPath   string
	Ctx               interpolate.Context
}

// Run executes the VMware Tools upload step, transferring the VMware Tools ISO to the virtual machine.
func (c *StepUploadTools) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if c.ToolsUploadFlavor == "" {
		return multistep.ActionContinue
	}

	comm := state.Get("communicator").(packersdk.Communicator)
	toolsSource := state.Get("tools_upload_source").(string)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Sayf("Uploading VMware Tools (%s)...", c.ToolsUploadFlavor)
	f, err := os.Open(toolsSource)
	if err != nil {
		state.Put("error", fmt.Errorf("error opening VMware Tools ISO: %s", err))
		return multistep.ActionHalt
	}
	defer f.Close()

	c.Ctx.Data = &toolsUploadPathTemplate{
		Flavor: c.ToolsUploadFlavor,
	}
	c.ToolsUploadPath, err = interpolate.Render(c.ToolsUploadPath, &c.Ctx)
	if err != nil {
		err = fmt.Errorf("error preparing upload path: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if err := comm.Upload(c.ToolsUploadPath, f, nil); err != nil {
		err = fmt.Errorf("error uploading vmware tools: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup after the VMware Tools upload step completes.
func (c *StepUploadTools) Cleanup(multistep.StateBag) {}
