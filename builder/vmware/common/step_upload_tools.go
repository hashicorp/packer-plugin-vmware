// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type toolsUploadPathTemplate struct {
	Flavor string
}

type StepUploadTools struct {
	RemoteType        string
	ToolsUploadFlavor string
	ToolsUploadPath   string
	Ctx               interpolate.Context
}

func (c *StepUploadTools) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)

	if c.ToolsUploadFlavor == "" {
		return multistep.ActionContinue
	}

	if c.RemoteType == "esxi" {
		if err := driver.ToolsInstall(); err != nil {
			state.Put("error", errors.New("unable to mount VMware Tools ISO, check the 'guest_os_type'"))
		}
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

func (c *StepUploadTools) Cleanup(multistep.StateBag) {}
