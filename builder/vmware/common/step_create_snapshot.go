// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepCreateSnapshot creates a named snapshot of the virtual machine after build completion.
type StepCreateSnapshot struct {
	SnapshotName *string
}

// Run executes the snapshot creation step, creating a named snapshot if configured.
func (s *StepCreateSnapshot) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	// If snapshot name is not set, skip snapshot creation.
	if *s.SnapshotName == "" {
		state.Put("snapshot_skipped", true)
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Creating snapshot of virtual machine...")
	vmFullPath := state.Get("vmx_path").(string)
	if err := driver.CreateSnapshot(vmFullPath, *s.SnapshotName); err != nil {
		err := fmt.Errorf("error creating snapshot: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	state.Put("snapshot_created", true)
	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup after the snapshot creation step completes.
func (s *StepCreateSnapshot) Cleanup(multistep.StateBag) {}
