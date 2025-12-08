// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepPrepareTools_impl(t *testing.T) {
	var _ multistep.Step = new(StepPrepareTools)
}

func TestStepPrepareTools(t *testing.T) {
	tf, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	tf.Close()
	defer os.Remove(tf.Name())

	state := testState(t)
	step := &StepPrepareTools{
		ToolsMode:         toolsModeUpload,
		ToolsUploadFlavor: "foo",
	}

	driver := state.Get("driver").(*DriverMock)

	// Mock results
	driver.ToolsIsoPathResult = tf.Name()

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if !driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should be called")
	}
	if driver.ToolsIsoPathFlavor != "foo" {
		t.Fatalf("bad: %#v", driver.ToolsIsoPathFlavor)
	}

	// Test the resulting state
	path, ok := state.GetOk("tools_upload_source")
	if !ok {
		t.Fatal("should have tools_upload_source")
	}
	if path != tf.Name() {
		t.Fatalf("bad: %#v", path)
	}
}

func TestStepPrepareTools_nonExist(t *testing.T) {
	state := testState(t)
	step := &StepPrepareTools{
		ToolsMode:         toolsModeUpload,
		ToolsUploadFlavor: "foo",
	}

	driver := state.Get("driver").(*DriverMock)

	// Mock results
	driver.ToolsIsoPathResult = "foo"

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionHalt {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); !ok {
		t.Fatal("should have error")
	}

	// Test the driver
	if !driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should be called")
	}
	if driver.ToolsIsoPathFlavor != "foo" {
		t.Fatalf("bad: %#v", driver.ToolsIsoPathFlavor)
	}

	// Test the resulting state
	if _, ok := state.GetOk("tools_upload_source"); ok {
		t.Fatal("should NOT have tools_upload_source")
	}
}

func TestStepPrepareTools_SourcePath(t *testing.T) {
	state := testState(t)
	step := &StepPrepareTools{
		ToolsMode:       toolsModeUpload,
		ToolsSourcePath: "/path/to/tool.iso",
	}

	driver := state.Get("driver").(*DriverMock)

	// Mock results
	driver.ToolsIsoPathResult = "foo"

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionHalt {
		t.Fatalf("Should have failed when stat failed %#v", action)
	}
	if _, ok := state.GetOk("error"); !ok {
		t.Fatal("should have error")
	}

	// Test the driver
	if driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should not be called when ToolsSourcePath is set")
	}

	// Test the resulting state
	if _, ok := state.GetOk("tools_upload_source"); ok {
		t.Fatal("should NOT have tools_upload_source")
	}
}

func TestStepPrepareTools_SourcePath_exists(t *testing.T) {
	state := testState(t)
	step := &StepPrepareTools{
		ToolsMode:       toolsModeUpload,
		ToolsSourcePath: "./step_prepare_tools.go",
	}

	driver := state.Get("driver").(*DriverMock)

	// Mock results
	driver.ToolsIsoPathResult = "foo"

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("Step should succeed when stat succeeds: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should not be called when ToolsSourcePath is set")
	}

	// Test the resulting state
	if _, ok := state.GetOk("tools_upload_source"); !ok {
		t.Fatal("should have tools_upload_source")
	}
}
func TestStepPrepareTools_AttachMode_SourcePath(t *testing.T) {
	state := testState(t)
	step := &StepPrepareTools{
		ToolsMode:       toolsModeAttach,
		ToolsSourcePath: "./step_prepare_tools.go",
	}

	driver := state.Get("driver").(*DriverMock)

	// Mock results
	driver.ToolsIsoPathResult = "foo"

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("Step should succeed when stat succeeds: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should not be called when ToolsSourcePath is set")
	}

	// Test the resulting state - should have tools_attach_source for attach mode
	if _, ok := state.GetOk("tools_attach_source"); !ok {
		t.Fatal("should have tools_attach_source")
	}
	if _, ok := state.GetOk("tools_upload_source"); ok {
		t.Fatal("should NOT have tools_upload_source in attach mode")
	}
}

func TestStepPrepareTools_AttachMode_Flavor(t *testing.T) {
	tf, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	tf.Close()
	defer os.Remove(tf.Name())

	state := testState(t)
	step := &StepPrepareTools{
		ToolsMode:         toolsModeAttach,
		ToolsUploadFlavor: "foo",
	}

	driver := state.Get("driver").(*DriverMock)

	// Mock results
	driver.ToolsIsoPathResult = tf.Name()

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if !driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should be called")
	}
	if driver.ToolsIsoPathFlavor != "foo" {
		t.Fatalf("bad: %#v", driver.ToolsIsoPathFlavor)
	}

	// Test the resulting state - should have tools_attach_source for attach mode
	path, ok := state.GetOk("tools_attach_source")
	if !ok {
		t.Fatal("should have tools_attach_source")
	}
	if path != tf.Name() {
		t.Fatalf("bad: %#v", path)
	}
	if _, ok := state.GetOk("tools_upload_source"); ok {
		t.Fatal("should NOT have tools_upload_source in attach mode")
	}
}

func TestStepPrepareTools_DisableMode(t *testing.T) {
	state := testState(t)
	step := &StepPrepareTools{
		ToolsMode:         toolsModeDisable,
		ToolsUploadFlavor: "foo",
		ToolsSourcePath:   "./step_prepare_tools.go",
	}

	driver := state.Get("driver").(*DriverMock)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver - should not be called in disable mode
	if driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should not be called in disable mode")
	}

	// Test the resulting state - should have no tools sources
	if _, ok := state.GetOk("tools_upload_source"); ok {
		t.Fatal("should NOT have tools_upload_source in disable mode")
	}
	if _, ok := state.GetOk("tools_attach_source"); ok {
		t.Fatal("should NOT have tools_attach_source in disable mode")
	}
}

func TestStepPrepareTools_BackwardCompatibility(t *testing.T) {
	tf, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	tf.Close()
	defer os.Remove(tf.Name())

	state := testState(t)
	step := &StepPrepareTools{
		// No ToolsMode specified - should default to upload behavior
		ToolsUploadFlavor: "foo",
	}

	driver := state.Get("driver").(*DriverMock)

	// Mock results
	driver.ToolsIsoPathResult = tf.Name()

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if !driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should be called")
	}
	if driver.ToolsIsoPathFlavor != "foo" {
		t.Fatalf("bad: %#v", driver.ToolsIsoPathFlavor)
	}

	// Test the resulting state - should default to upload mode
	path, ok := state.GetOk("tools_upload_source")
	if !ok {
		t.Fatal("should have tools_upload_source for backward compatibility")
	}
	if path != tf.Name() {
		t.Fatalf("bad: %#v", path)
	}
}

func TestStepPrepareTools_NoConfiguration(t *testing.T) {
	state := testState(t)
	step := &StepPrepareTools{
		// No configuration provided
	}

	driver := state.Get("driver").(*DriverMock)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver - should not be called
	if driver.ToolsIsoPathCalled {
		t.Fatal("tools iso path should not be called when no configuration is provided")
	}

	// Test the resulting state - should have no tools sources
	if _, ok := state.GetOk("tools_upload_source"); ok {
		t.Fatal("should NOT have tools_upload_source when no configuration is provided")
	}
	if _, ok := state.GetOk("tools_attach_source"); ok {
		t.Fatal("should NOT have tools_attach_source when no configuration is provided")
	}
}
