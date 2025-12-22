// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepOutputDir manages the output directory configuration for a build step,
// including forceful directory overwrite.
type StepOutputDir struct {
	Force        bool
	OutputConfig *OutputConfig
	VMName       string
	success      bool
}

// SetOutputAndExportDirs configures the output and export directories based on local or remote build type.
func (s *StepOutputDir) SetOutputAndExportDirs(state multistep.StateBag) OutputDir {
	driver := state.Get("driver")

	// Output configuration is local-only now.
	var dir OutputDir
	switch d := driver.(type) {
	case OutputDir:
		// The driver fulfills the OutputDir interface.
		dir = d
	default:
		// Create the output directory locally.
		dir = new(LocalOutputDir)
	}

	// Track the local output dir for export steps.
	exportOutputPath := s.OutputConfig.OutputDir

	// Use the configured output directory as-is.
	dir.SetOutputDir(s.OutputConfig.OutputDir)

	// Stash for later steps (cleanup, artifact, export).
	state.Put("dir", dir)
	state.Put("export_output_path", exportOutputPath)
	return dir
}

// Run executes the output directory setup step, creating or cleaning the output directory as needed.
func (s *StepOutputDir) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	ui.Say("Configuring output and export directories...")

	dir := s.SetOutputAndExportDirs(state)
	exists, err := dir.DirExists()
	if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if exists {
		if s.Force {
			ui.Say("Deleting previous output directory...")
			_ = dir.RemoveAll()
		} else {
			state.Put("error", fmt.Errorf("output directory '%s' already exists", dir.String()))
			return multistep.ActionHalt
		}
	}

	if err := dir.MkdirAll(); err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}

	s.success = true
	return multistep.ActionContinue
}

// Cleanup removes the output directory if the build was cancelled or halted.
func (s *StepOutputDir) Cleanup(state multistep.StateBag) {
	if !s.success {
		return
	}

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)

	if cancelled || halted {
		dir := state.Get("dir").(OutputDir)
		ui := state.Get("ui").(packersdk.Ui)

		exists, _ := dir.DirExists()
		if exists {
			ui.Say("Deleting output directory...")
			for i := 0; i < 5; i++ {
				err := dir.RemoveAll()
				if err == nil {
					break
				}

				log.Printf("[WARN] Failed to remove output dir: %s", err)
				time.Sleep(2 * time.Second)
			}
		}
	}
}
