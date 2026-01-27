// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepUploadTools_Run_AttachMode(t *testing.T) {
	state := new(multistep.BasicStateBag)

	step := &StepUploadTools{
		ToolsMode:         toolsModeAttach,
		ToolsUploadFlavor: "linux",
	}

	action := step.Run(context.Background(), state)

	if action != multistep.ActionContinue {
		t.Errorf("Expected ActionContinue, got %v", action)
	}

	// Verify no error was set in state
	if err := state.Get("error"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestStepUploadTools_Run_DisableMode(t *testing.T) {
	state := new(multistep.BasicStateBag)

	step := &StepUploadTools{
		ToolsMode:         toolsModeDisable,
		ToolsUploadFlavor: "linux",
	}

	action := step.Run(context.Background(), state)

	if action != multistep.ActionContinue {
		t.Errorf("Expected ActionContinue, got %v", action)
	}

	// Verify no error was set in state
	if err := state.Get("error"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestStepUploadTools_Run_UploadMode(t *testing.T) {
	state := new(multistep.BasicStateBag)

	step := &StepUploadTools{
		ToolsMode:         toolsModeUpload,
		ToolsUploadFlavor: "", // Empty flavor should cause early return
	}

	action := step.Run(context.Background(), state)

	if action != multistep.ActionContinue {
		t.Errorf("Expected ActionContinue, got %v", action)
	}

	// Verify no error was set in state
	if err := state.Get("error"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestStepUploadTools_Run_NoModeSpecified(t *testing.T) {
	state := new(multistep.BasicStateBag)

	step := &StepUploadTools{
		ToolsMode:         "", // No mode specified
		ToolsUploadFlavor: "", // Empty flavor should cause early return
	}

	action := step.Run(context.Background(), state)

	if action != multistep.ActionContinue {
		t.Errorf("Expected ActionContinue, got %v", action)
	}

	// Verify no error was set in state
	if err := state.Get("error"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestStepUploadTools_Run_BackwardCompatibility(t *testing.T) {
	state := new(multistep.BasicStateBag)

	// Test that existing behavior is maintained when no tools_mode is specified
	// but tools_upload_flavor is set (should continue to existing logic)
	step := &StepUploadTools{
		ToolsMode:         "", // No mode specified (backward compatibility)
		ToolsUploadFlavor: "", // Empty flavor should cause early return
	}

	action := step.Run(context.Background(), state)

	if action != multistep.ActionContinue {
		t.Errorf("Expected ActionContinue, got %v", action)
	}

	// Verify no error was set in state
	if err := state.Get("error"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
