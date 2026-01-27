// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepCleanVMX_impl(t *testing.T) {
	var _ multistep.Step = new(StepCleanVMX)
}

func TestStepCleanVMX(t *testing.T) {
	state := testState(t)
	step := new(StepCleanVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	state.Put("vmx_path", vmxPath)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}
}

func TestStepCleanVMX_floppyPath(t *testing.T) {
	state := testState(t)
	step := new(StepCleanVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	if err := os.WriteFile(vmxPath, []byte(testVMXFloppyPath), 0644); err != nil { //nolint:gosec
		t.Fatalf("err: %s", err)
	}

	// Set the path to the temporary vmx
	state.Put("vmx_path", vmxPath)

	// Add the floppy device to the list of temporary build devices
	state.Put("temporaryDevices", []string{"floppy0"})

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the resulting data
	vmxContents, err := os.ReadFile(vmxPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	vmxData := ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		{"floppy0.present", "FALSE"},
		{"floppy0.filetype", ""},
		{"floppy0.filename", ""},
	}

	for _, tc := range cases {
		if tc.Value == "" {
			if _, ok := vmxData[tc.Key]; ok {
				t.Fatalf("should not have key: %s", tc.Key)
			}
		} else {
			if vmxData[tc.Key] != tc.Value {
				t.Fatalf("bad: %s %#v", tc.Key, vmxData[tc.Key])
			}
		}
	}
}

func TestStepCleanVMX_isoPath(t *testing.T) {
	state := testState(t)
	step := new(StepCleanVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	if err := os.WriteFile(vmxPath, []byte(testVMXISOPath), 0644); err != nil { //nolint:gosec
		t.Fatalf("err: %s", err)
	}

	// Set the path to the temporary vmx
	state.Put("vmx_path", vmxPath)

	// Add the cdrom device to the list of temporary build devices
	state.Put("temporaryDevices", []string{"ide0:0"})

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the resulting data
	vmxContents, err := os.ReadFile(vmxPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	vmxData := ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		{"ide0:0.filename", "auto detect"},
		{"ide0:0.devicetype", "cdrom-raw"},
		{"ide0:1.filename", "bar"},
		{"foo", "bar"},
	}

	for _, tc := range cases {
		if tc.Value == "" {
			if _, ok := vmxData[tc.Key]; ok {
				t.Fatalf("should not have key: %s", tc.Key)
			}
		} else {
			if vmxData[tc.Key] != tc.Value {
				t.Fatalf("bad: %s %#v", tc.Key, vmxData[tc.Key])
			}
		}
	}
}

func TestStepCleanVMX_ethernet(t *testing.T) {
	state := testState(t)
	step := &StepCleanVMX{
		RemoveEthernetInterfaces: true,
	}

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	if err := os.WriteFile(vmxPath, []byte(testVMXEthernet), 0644); err != nil { //nolint:gosec
		t.Fatalf("err: %s", err)
	}

	// Set the path to the temporary vmx
	state.Put("vmx_path", vmxPath)

	// TODO: Add the ethernet devices to the list of temporary build devices
	// state.Put("temporaryDevices", []string{"ethernet0", "ethernet1"})

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the resulting data
	vmxContents, err := os.ReadFile(vmxPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	vmxData := ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		{"ethernet0.addresstype", ""},
		{"ethernet0.bsdname", ""},
		{"ethernet0.connectiontype", ""},
		{"ethernet1.addresstype", ""},
		{"ethernet1.bsdname", ""},
		{"ethernet1.connectiontype", ""},
		{"foo", "bar"},
	}

	for _, tc := range cases {
		if tc.Value == "" {
			if _, ok := vmxData[tc.Key]; ok {
				t.Fatalf("should not have key: %s", tc.Key)
			}
		} else {
			if vmxData[tc.Key] != tc.Value {
				t.Fatalf("bad: %s %#v", tc.Key, vmxData[tc.Key])
			}
		}
	}
}

const testVMXFloppyPath = `
floppy0.present = "TRUE"
floppy0.filetype = "file"
`

const testVMXISOPath = `
ide0:0.devicetype = "cdrom-image"
ide0:0.filename = "foo"
ide0:1.filename = "bar"
foo = "bar"
`

const testVMXEthernet = `
ethernet0.addresstype = "generated"
ethernet0.bsdname = "en0"
ethernet0.connectiontype = "nat"
ethernet1.addresstype = "generated"
ethernet1.bsdname = "en1"
ethernet1.connectiontype = "nat"
foo = "bar"
`

func TestStepCleanVMX_toolsCDROM(t *testing.T) {
	state := testState(t)
	step := new(StepCleanVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	if err := os.WriteFile(vmxPath, []byte(testVMXToolsCDROM), 0644); err != nil { //nolint:gosec
		t.Fatalf("err: %s", err)
	}

	// Set the path to the temporary vmx
	state.Put("vmx_path", vmxPath)

	// Set the tools CD-ROM device in state bag (simulating StepAttachToolsCDROM)
	state.Put("tools_cdrom_device", "ide0:1")
	state.Put("tools_cdrom_adapter", "ide")

	// Add both installation ISO and tools CD-ROM to temporary devices
	state.Put("temporaryDevices", []string{"ide0:0", "ide0:1"})

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the resulting data
	vmxContents, err := os.ReadFile(vmxPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	vmxData := ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		// Installation ISO should be converted to cdrom-raw (existing behavior)
		{"ide0:0.filename", "auto detect"},
		{"ide0:0.devicetype", "cdrom-raw"},
		{"ide0:0.clientdevice", "TRUE"},
		// Tools CD-ROM should be completely removed
		{"ide0:1.present", "FALSE"},
		{"ide0:1.filename", ""},
		{"ide0:1.devicetype", ""},
		{"ide0:1.clientdevice", ""},
		// Other VMX entries should be preserved
		{"foo", "bar"},
		{"ide0.present", "TRUE"},
	}

	for _, tc := range cases {
		if tc.Value == "" {
			if _, ok := vmxData[tc.Key]; ok {
				t.Fatalf("should not have key: %s", tc.Key)
			}
		} else {
			if vmxData[tc.Key] != tc.Value {
				t.Fatalf("bad: %s expected %#v, got %#v", tc.Key, tc.Value, vmxData[tc.Key])
			}
		}
	}
}

func TestStepCleanVMX_toolsCDROMOnly(t *testing.T) {
	state := testState(t)
	step := new(StepCleanVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	if err := os.WriteFile(vmxPath, []byte(testVMXToolsCDROMOnly), 0644); err != nil { //nolint:gosec
		t.Fatalf("err: %s", err)
	}

	// Set the path to the temporary vmx
	state.Put("vmx_path", vmxPath)

	// Set the tools CD-ROM device in state bag (simulating StepAttachToolsCDROM)
	state.Put("tools_cdrom_device", "sata0:0")
	state.Put("tools_cdrom_adapter", "sata")

	// Add only tools CD-ROM to temporary devices
	state.Put("temporaryDevices", []string{"sata0:0"})

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the resulting data
	vmxContents, err := os.ReadFile(vmxPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	vmxData := ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		// Tools CD-ROM should be completely removed
		{"sata0:0.present", "FALSE"},
		{"sata0:0.filename", ""},
		{"sata0:0.devicetype", ""},
		// Other VMX entries should be preserved
		{"foo", "bar"},
	}

	for _, tc := range cases {
		if tc.Value == "" {
			if _, ok := vmxData[tc.Key]; ok {
				t.Fatalf("should not have key: %s", tc.Key)
			}
		} else {
			if vmxData[tc.Key] != tc.Value {
				t.Fatalf("bad: %s expected %#v, got %#v", tc.Key, tc.Value, vmxData[tc.Key])
			}
		}
	}
}

func TestStepCleanVMX_noToolsCDROM(t *testing.T) {
	state := testState(t)
	step := new(StepCleanVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	if err := os.WriteFile(vmxPath, []byte(testVMXISOPath), 0644); err != nil { //nolint:gosec
		t.Fatalf("err: %s", err)
	}

	// Set the path to the temporary vmx
	state.Put("vmx_path", vmxPath)

	// No tools CD-ROM device in state bag
	// Add only installation ISO to temporary devices
	state.Put("temporaryDevices", []string{"ide0:0"})

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the resulting data
	vmxContents, err := os.ReadFile(vmxPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	vmxData := ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		// Installation ISO should be converted to cdrom-raw (existing behavior)
		{"ide0:0.filename", "auto detect"},
		{"ide0:0.devicetype", "cdrom-raw"},
		// Other entries should be preserved
		{"ide0:1.filename", "bar"},
		{"foo", "bar"},
	}

	for _, tc := range cases {
		if vmxData[tc.Key] != tc.Value {
			t.Fatalf("bad: %s expected %#v, got %#v", tc.Key, tc.Value, vmxData[tc.Key])
		}
	}
}

func TestStepCleanVMX_preserveUserCDROM(t *testing.T) {
	state := testState(t)
	step := new(StepCleanVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)
	if err := os.WriteFile(vmxPath, []byte(testVMXMultipleCDROM), 0644); err != nil { //nolint:gosec
		t.Fatalf("err: %s", err)
	}

	// Set the path to the temporary vmx
	state.Put("vmx_path", vmxPath)

	// Set the tools CD-ROM device in state bag (simulating StepAttachToolsCDROM)
	state.Put("tools_cdrom_device", "ide0:1")
	state.Put("tools_cdrom_adapter", "ide")

	// Add installation ISO and tools CD-ROM to temporary devices
	// Note: ide1:0 is a user-defined CD-ROM that should be preserved
	state.Put("temporaryDevices", []string{"ide0:0", "ide0:1"})

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the resulting data
	vmxContents, err := os.ReadFile(vmxPath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	vmxData := ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		// Installation ISO should be converted to cdrom-raw (existing behavior)
		{"ide0:0.filename", "auto detect"},
		{"ide0:0.devicetype", "cdrom-raw"},
		{"ide0:0.clientdevice", "TRUE"},
		// Tools CD-ROM should be completely removed
		{"ide0:1.present", "FALSE"},
		{"ide0:1.filename", ""},
		{"ide0:1.devicetype", ""},
		// User-defined CD-ROM should be preserved unchanged
		{"ide1:0.filename", "user-defined.iso"},
		{"ide1:0.devicetype", "cdrom-image"},
		{"ide1:0.present", "TRUE"},
		// Other VMX entries should be preserved
		{"foo", "bar"},
	}

	for _, tc := range cases {
		if tc.Value == "" {
			if _, ok := vmxData[tc.Key]; ok {
				t.Fatalf("should not have key: %s", tc.Key)
			}
		} else {
			if vmxData[tc.Key] != tc.Value {
				t.Fatalf("bad: %s expected %#v, got %#v", tc.Key, tc.Value, vmxData[tc.Key])
			}
		}
	}
}

const testVMXToolsCDROM = `
ide0.present = "TRUE"
ide0:0.devicetype = "cdrom-image"
ide0:0.filename = "install.iso"
ide0:0.present = "TRUE"
ide0:1.devicetype = "cdrom-image"
ide0:1.filename = "/path/to/vmware-tools.iso"
ide0:1.present = "TRUE"
foo = "bar"
`

const testVMXToolsCDROMOnly = `
sata0.present = "TRUE"
sata0:0.devicetype = "cdrom-image"
sata0:0.filename = "/path/to/vmware-tools.iso"
sata0:0.present = "TRUE"
foo = "bar"
`

const testVMXMultipleCDROM = `
ide0.present = "TRUE"
ide0:0.devicetype = "cdrom-image"
ide0:0.filename = "install.iso"
ide0:0.present = "TRUE"
ide0:1.devicetype = "cdrom-image"
ide0:1.filename = "/path/to/vmware-tools.iso"
ide0:1.present = "TRUE"
ide1.present = "TRUE"
ide1:0.devicetype = "cdrom-image"
ide1:0.filename = "user-defined.iso"
ide1:0.present = "TRUE"
foo = "bar"
`
