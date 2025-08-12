// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepCleanFiles represents a step for cleaning up unnecessary files from a directory.
type StepCleanFiles struct{}

func (StepCleanFiles) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	dir := state.Get("dir").(OutputDir)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Deleting unnecessary files...")
	files, err := dir.ListFiles()
	if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}

	for _, path := range files {
		// If the file isn't critical to the function of the
		// virtual machine, we get rid of it.
		keep := false
		ext := filepath.Ext(path)
		for _, goodExt := range fileExtensions {
			if goodExt == ext {
				keep = true
				break
			}
		}

		if !keep {
			ui.Sayf("Deleting: %s", path)
			if err = dir.Remove(path); err != nil {
				// Only report the error if the file still exists. We do this
				// because sometimes the files naturally get removed on their
				// own as VMware does its own cleanup.
				if _, serr := os.Stat(path); serr == nil || !os.IsNotExist(serr) {
					state.Put("error", err)
					return multistep.ActionHalt
				}
			}
		}
	}

	return multistep.ActionContinue
}

func (StepCleanFiles) Cleanup(multistep.StateBag) {}
