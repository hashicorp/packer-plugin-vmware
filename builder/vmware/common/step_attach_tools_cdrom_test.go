// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepAttachToolsCDROM_impl(t *testing.T) {
	var _ multistep.Step = new(StepAttachToolsCDROM)
}

func TestStepAttachToolsCDROM_SkipWhenNotAttachMode(t *testing.T) {
	state := testState(t)
	step := &StepAttachToolsCDROM{
		ToolsMode: toolsModeUpload,
	}

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}
}

func TestStepAttachToolsCDROM_SkipWhenNoToolsSource(t *testing.T) {
	state := testState(t)
	step := &StepAttachToolsCDROM{
		ToolsMode: toolsModeAttach,
	}

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}
}

func TestStepAttachToolsCDROM_Success(t *testing.T) {
	tmpDir := t.TempDir()
	vmxPath := filepath.Join(tmpDir, "test.vmx")

	vmxContent := `config.version = "8"
virtualHW.version = "10"
displayName = "test"
`
	err := os.WriteFile(vmxPath, []byte(vmxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test VMX file: %v", err)
	}

	toolsIsoPath := filepath.Join(tmpDir, "tools.iso")
	err = os.WriteFile(toolsIsoPath, []byte("fake iso content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tools ISO file: %v", err)
	}

	state := testState(t)
	state.Put("vmx_path", vmxPath)
	state.Put("tools_attach_source", toolsIsoPath)

	step := &StepAttachToolsCDROM{
		ToolsMode:        toolsModeAttach,
		CDROMAdapterType: "ide",
	}

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatalf("should NOT have error: %v", state.Get("error"))
	}

	devicePath, ok := state.GetOk("tools_cdrom_device")
	if !ok {
		t.Fatal("should have tools_cdrom_device in state")
	}
	if devicePath.(string) != "ide0:0" {
		t.Fatalf("expected device path 'ide0:0', got '%s'", devicePath.(string))
	}

	adapterType, ok := state.GetOk("tools_cdrom_adapter")
	if !ok {
		t.Fatal("should have tools_cdrom_adapter in state")
	}
	if adapterType.(string) != "ide" {
		t.Fatalf("expected adapter type 'ide', got '%s'", adapterType.(string))
	}

	tmpDevices := state.Get("temporaryDevices").([]string)
	found := false
	for _, device := range tmpDevices {
		if device == "ide0:0" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("tools CD-ROM device should be added to temporaryDevices")
	}

	vmxData, err := ReadVMX(vmxPath)
	if err != nil {
		t.Fatalf("Failed to read VMX file: %v", err)
	}

	if vmxData["ide0:0.present"] != "TRUE" {
		t.Fatal("CD-ROM device should be present")
	}
	if vmxData["ide0:0.filename"] != toolsIsoPath {
		t.Fatalf("expected filename '%s', got '%s'", toolsIsoPath, vmxData["ide0:0.filename"])
	}
	if vmxData["ide0:0.devicetype"] != "cdrom-image" {
		t.Fatalf("expected devicetype 'cdrom-image', got '%s'", vmxData["ide0:0.devicetype"])
	}
	if vmxData["ide0.present"] != "TRUE" {
		t.Fatal("IDE adapter should be present")
	}
}

func TestStepAttachToolsCDROM_ConflictResolution(t *testing.T) {
	tmpDir := t.TempDir()
	vmxPath := filepath.Join(tmpDir, "test.vmx")

	vmxContent := `config.version = "8"
virtualHW.version = "10"
displayName = "test"
ide0.present = "TRUE"
ide0:0.present = "TRUE"
ide0:0.filename = "/path/to/install.iso"
ide0:0.devicetype = "cdrom-image"
`
	err := os.WriteFile(vmxPath, []byte(vmxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test VMX file: %v", err)
	}

	// Create a temporary tools ISO file
	toolsIsoPath := filepath.Join(tmpDir, "tools.iso")
	err = os.WriteFile(toolsIsoPath, []byte("fake iso content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tools ISO file: %v", err)
	}

	state := testState(t)
	state.Put("vmx_path", vmxPath)
	state.Put("tools_attach_source", toolsIsoPath)

	step := &StepAttachToolsCDROM{
		ToolsMode:        toolsModeAttach,
		CDROMAdapterType: "ide",
	}

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatalf("should NOT have error: %v", state.Get("error"))
	}

	devicePath, ok := state.GetOk("tools_cdrom_device")
	if !ok {
		t.Fatal("should have tools_cdrom_device in state")
	}
	if devicePath.(string) != "ide0:1" {
		t.Fatalf("expected device path 'ide0:1', got '%s'", devicePath.(string))
	}

	vmxData, err := ReadVMX(vmxPath)
	if err != nil {
		t.Fatalf("Failed to read VMX file: %v", err)
	}

	if vmxData["ide0:1.present"] != "TRUE" {
		t.Fatal("Tools CD-ROM device should be present at ide0:1")
	}

	if vmxData["ide0:1.filename"] != toolsIsoPath {
		t.Fatalf("expected filename '%s', got '%s'", toolsIsoPath, vmxData["ide0:1.filename"])
	}

	if vmxData["ide0:0.present"] != "TRUE" {
		t.Fatal("Original CD-ROM device should still be present")
	}

	if vmxData["ide0:0.filename"] != "/path/to/install.iso" {
		t.Fatal("Original CD-ROM device filename should be preserved")
	}
}

func TestStepAttachToolsCDROM_DefaultAdapterType(t *testing.T) {
	tmpDir := t.TempDir()
	vmxPath := filepath.Join(tmpDir, "test.vmx")

	vmxContent := `config.version = "8"
virtualHW.version = "10"
displayName = "test"
`
	err := os.WriteFile(vmxPath, []byte(vmxContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test VMX file: %v", err)
	}

	// Create a temporary tools ISO file
	toolsIsoPath := filepath.Join(tmpDir, "tools.iso")
	err = os.WriteFile(toolsIsoPath, []byte("fake iso content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test tools ISO file: %v", err)
	}

	state := testState(t)
	state.Put("vmx_path", vmxPath)
	state.Put("tools_attach_source", toolsIsoPath)

	step := &StepAttachToolsCDROM{
		ToolsMode: toolsModeAttach,
	}

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}

	if _, ok := state.GetOk("error"); ok {
		t.Fatalf("should NOT have error: %v", state.Get("error"))
	}

	adapterType, ok := state.GetOk("tools_cdrom_adapter")
	if !ok {
		t.Fatal("should have tools_cdrom_adapter in state")
	}

	if adapterType.(string) != "ide" {
		t.Fatalf("expected default adapter type 'ide', got '%s'", adapterType.(string))
	}
}

func TestStepAttachToolsCDROM_VMXReadError(t *testing.T) {
	state := testState(t)
	state.Put("vmx_path", "/nonexistent/path/test.vmx")
	state.Put("tools_attach_source", "/path/to/tools.iso")

	step := &StepAttachToolsCDROM{
		ToolsMode:        toolsModeAttach,
		CDROMAdapterType: "ide",
	}

	if action := step.Run(context.Background(), state); action != multistep.ActionHalt {
		t.Fatalf("expected ActionHalt, got: %#v", action)
	}

	if _, ok := state.GetOk("error"); !ok {
		t.Fatal("should have error")
	}
}

func TestStepAttachToolsCDROM_Cleanup(t *testing.T) {
	state := testState(t)
	step := &StepAttachToolsCDROM{}

	// no-op
	step.Cleanup(state)
}
