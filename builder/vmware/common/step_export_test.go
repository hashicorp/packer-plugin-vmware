// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
)

func TestStepExport_impl(t *testing.T) {
	var _ multistep.Step = new(StepExport)
}

func stringPointer(s string) *string {
	return &s
}

func remoteExportTestState(t *testing.T) multistep.StateBag {
	state := testState(t)
	driverConfig := &DriverConfig{
		RemoteHost:     "123.45.67.8",
		RemotePassword: "password",
		RemoteUser:     "user",
		RemoteType:     "esxi",
	}
	state.Put("driverConfig", driverConfig)
	state.Put("display_name", "vm_name")
	return state
}

func TestStepExport_ReturnIfSkip(t *testing.T) {
	state := testState(t)
	driverConfig := &DriverConfig{}
	state.Put("driverConfig", driverConfig)
	step := new(StepExport)

	step.SkipExport = true

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// We told step to skip so it should not have reached the driver's Export
	// func.
	d := state.Get("driver").(*DriverMock)
	if d.ExportCalled {
		t.Fatal("Should not have called the driver export func")
	}

	// Cleanup
	step.Cleanup(state)
}

func TestStepExport_localArgs(t *testing.T) {
	// Although the remote arguments are available and not being overridden,
	// the test should ignore them because remoteType is not specified as 'esx'.
	state := testState(t)
	driverConfig := &DriverConfig{}
	state.Put("driverConfig", driverConfig)
	step := new(StepExport)

	step.SkipExport = false
	step.OutputDir = stringPointer("test_output")
	step.VMName = "test-name"
	step.Format = "ova"

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Check that step ran, and called Export with the expected args.
	d := state.Get("driver").(*DriverMock)
	if !d.ExportCalled {
		t.Fatal("Should have called the driver export func")
	}

	assert.Equal(t, d.ExportArgs,
		[]string{
			filepath.Join("test_output", "test-name.vmx"),
			filepath.Join("test_output", "test-name.ova")})

	// Cleanup
	step.Cleanup(state)
}

func TestStepExport_localArgsExportOutputPath(t *testing.T) {
	// Although the remote arguments are available and not being overridden,
	// the test should ignore them because remoteType is not specified as 'esx'.
	state := testState(t)
	driverConfig := &DriverConfig{}
	state.Put("driverConfig", driverConfig)
	state.Put("export_output_path", "local_output")
	step := new(StepExport)

	step.SkipExport = false
	step.OutputDir = stringPointer("test_output")
	step.VMName = "test-name"
	step.Format = "ova"

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Check that step ran, and called Export with the expected args.
	d := state.Get("driver").(*DriverMock)
	if !d.ExportCalled {
		t.Fatal("Should have called the driver export func")
	}

	assert.Equal(t, d.ExportArgs,
		[]string{
			filepath.Join("local_output", "test-name.vmx"),
			filepath.Join("local_output", "test-name.ova")})

	// Cleanup
	step.Cleanup(state)
}

func TestStepExport_localArgs_OvftoolOptions(t *testing.T) {
	// Although the remote arguments are available and not being overridden,
	// the test should ignore them because remoteType is not specified as 'esx'.
	state := testState(t)
	driverConfig := &DriverConfig{}
	state.Put("driverConfig", driverConfig)
	step := new(StepExport)

	step.SkipExport = false
	step.OutputDir = stringPointer("test_output")
	step.VMName = "test-name"
	step.Format = "ova"
	step.OVFToolOptions = []string{"--option=value", "--second-option=\"quoted value\""}

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Check that step ran, and called Export with the expected args.
	d := state.Get("driver").(*DriverMock)
	if !d.ExportCalled {
		t.Fatal("Should have called the driver export func")
	}

	assert.Equal(t, d.ExportArgs, []string{"--option=value",
		"--second-option=\"quoted value\"",
		filepath.Join("test_output", "test-name.vmx"),
		filepath.Join("test_output", "test-name.ova")})

	// Cleanup
	step.Cleanup(state)
}

func TestStepExport_RemoteArgs(t *testing.T) {
	// Although the remote arguments are available and not being overridden,
	// the test should ignore them because remoteType is not specified as 'esx'.
	state := remoteExportTestState(t)
	step := new(StepExport)

	step.SkipExport = false
	step.OutputDir = stringPointer("test_output")
	step.VMName = "test-name"
	step.Format = "ova"

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Check that step ran, and called Export with the expected args.
	d := state.Get("driver").(*DriverMock)
	if !d.ExportCalled {
		t.Fatal("Should have called the driver export func")
	}

	assert.Equal(t, d.ExportArgs, []string{"--noSSLVerify=true",
		"--skipManifestCheck",
		"-tt=ova",
		"vi://user:password@123.45.67.8/vm_name",
		filepath.Join("test_output", "test-name.ova")})

	// Cleanup
	step.Cleanup(state)
}

func TestStepExport_RemoteArgsWithExportOutputPath(t *testing.T) {
	// Although the remote arguments are available and not being overridden,
	// the test should ignore them because remoteType is not specified as 'esx'.
	state := remoteExportTestState(t)
	state.Put("export_output_path", "local_output")
	step := new(StepExport)

	step.SkipExport = false
	step.OutputDir = stringPointer("test_output")
	step.VMName = "test-name"
	step.Format = "ova"

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Check that step ran, and called Export with the expected args.
	d := state.Get("driver").(*DriverMock)
	if !d.ExportCalled {
		t.Fatal("Should have called the driver export func")
	}

	assert.Equal(t, d.ExportArgs, []string{"--noSSLVerify=true",
		"--skipManifestCheck",
		"-tt=ova",
		"vi://user:password@123.45.67.8/vm_name",
		filepath.Join("local_output", "test-name.ova")})

	// Cleanup
	step.Cleanup(state)
}
