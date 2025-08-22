// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepSuppressMessages represents a step to handle the suppression of messages in a VMX file.
type StepSuppressMessages struct{}

// Run executes the message suppression step, configuring the VM to suppress dialog messages.
func (s *StepSuppressMessages) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmxPath := state.Get("vmx_path").(string)

	log.Println("[INFO] Suppressing messages in VMX")
	if err := driver.SuppressMessages(vmxPath); err != nil {
		err = fmt.Errorf("error suppressing messages: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup after the message suppression step completes.
func (s *StepSuppressMessages) Cleanup(state multistep.StateBag) {}
