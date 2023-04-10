// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepCreateSnapshot step creates the intial snapshot for the VM after clean-up.
type StepCreateSnapshot struct {
	SnapshotName *string
}

func (s *StepCreateSnapshot) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if *s.SnapshotName != "" { // if snapshot name is set create one, if not don't
		driver := state.Get("driver").(Driver)
		ui := state.Get("ui").(packersdk.Ui)

		ui.Say("Creating inital snapshot")
		vmFullPath := state.Get("vmx_path").(string)
		if err := driver.CreateSnapshot(vmFullPath, *s.SnapshotName); err != nil {
			err := fmt.Errorf("Error creating snapshot: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		state.Put("snapshot_created", true)
	} else {
		state.Put("snapshot_skipped", true)
	}
	return multistep.ActionContinue
}

func (s *StepCreateSnapshot) Cleanup(multistep.StateBag) {}
