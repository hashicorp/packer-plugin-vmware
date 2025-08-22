// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepCompactDisk represents a step for compacting attached virtual disks.
type StepCompactDisk struct {
	Skip bool
}

// Run executes the disk compaction step, compacting all attached virtual disks to reclaim space.
func (s StepCompactDisk) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	diskFullPaths := state.Get("disk_full_paths").([]string)

	if s.Skip {
		ui.Say("Skipping disk compaction...")
		return multistep.ActionContinue
	}

	ui.Say("Compacting all attached virtual disks...")
	for i, diskFullPath := range diskFullPaths {
		ui.Sayf("Compacting virtual disk %d", i+1)
		if err := driver.CompactDisk(diskFullPath); err != nil {
			state.Put("error", fmt.Errorf("error compacting disk: %s", err))
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

// Cleanup performs any necessary cleanup after the disk compaction step completes.
func (StepCompactDisk) Cleanup(multistep.StateBag) {}
