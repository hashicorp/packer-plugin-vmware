// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
)

func testVMXFile(t *testing.T) string {
	tf, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// displayName must always be set
	err = WriteVMX(tf.Name(), map[string]string{"displayName": "PackerBuild"})
	if err != nil {
		t.Fatalf("error writing .vmx file: %s", err)
	}
	tf.Close()

	return tf.Name()
}

func TestStepConfigureVMX_impl(t *testing.T) {
	var _ multistep.Step = new(StepConfigureVMX)
}

func TestStepConfigureVMX(t *testing.T) {
	state := testState(t)
	step := new(StepConfigureVMX)
	step.CustomData = map[string]string{
		"foo": "bar",
	}

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
		// Stuff we set
		{"msg.autoanswer", "true"},
		{"uuid.action", "create"},

		// Custom data
		{"foo", "bar"},

		// Stuff that should NOT exist
		{"floppy0.present", ""},
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

func TestStepConfigureVMX_floppyPath(t *testing.T) {
	state := testState(t)
	step := new(StepConfigureVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)

	state.Put("floppy_path", "foo")
	state.Put("vmx_path", vmxPath)

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
		{"floppy0.present", "TRUE"},
		{"floppy0.filetype", "file"},
		{"floppy0.filename", "foo"},
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

func TestStepConfigureVMX_generatedAddresses(t *testing.T) {
	state := testState(t)
	step := new(StepConfigureVMX)

	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)

	additionalTestVmxData := []struct {
		Key   string
		Value string
	}{
		{"foo", "bar"},
		{"ethernet0.generatedaddress", "foo"},
		{"ethernet1.generatedaddress", "foo"},
		{"ethernet1.generatedaddressoffset", "foo"},
	}

	// Get any existing VMX data from the VMX file
	vmxData, err := ReadVMX(vmxPath)
	if err != nil {
		t.Fatalf("err %s", err)
	}

	// Add the additional key/value pairs we need for this test to the existing VMX data
	for _, data := range additionalTestVmxData {
		vmxData[data.Key] = data.Value
	}

	// Recreate the VMX file so it includes all the data needed for this test
	err = WriteVMX(vmxPath, vmxData)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state.Put("vmx_path", vmxPath)

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
	vmxData = ParseVMX(string(vmxContents))

	cases := []struct {
		Key   string
		Value string
	}{
		{"foo", "bar"},
		{"ethernet0.generatedaddress", ""},
		{"ethernet1.generatedaddress", ""},
		{"ethernet1.generatedaddressoffset", ""},
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

// Should fail if the displayName key is not found in the VMX
func TestStepConfigureVMX_displayNameMissing(t *testing.T) {
	state := testState(t)
	step := new(StepConfigureVMX)

	// testVMXFile adds displayName key/value pair to the VMX
	vmxPath := testVMXFile(t)
	defer os.Remove(vmxPath)

	// Bad: Delete displayName from the VMX/Create an empty VMX file
	err := WriteVMX(vmxPath, map[string]string{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state.Put("vmx_path", vmxPath)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionHalt {
		t.Fatalf("bad action: %#v. Should halt when displayName key is missing from VMX", action)
	}
	if _, ok := state.GetOk("error"); !ok {
		t.Fatal("should store error in state when displayName key is missing from VMX")
	}
}

// Should store the value of displayName in the statebag
func TestStepConfigureVMX_displayNameStore(t *testing.T) {
	state := testState(t)
	step := new(StepConfigureVMX)

	// testVMXFile adds displayName key/value pair to the VMX
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

	// The value of displayName must be stored in the statebag
	if _, ok := state.GetOk("display_name"); !ok {
		t.Fatalf("displayName should be stored in the statebag as 'display_name'")
	}
}

func TestStepConfigureVMX_vmxPathBad(t *testing.T) {
	state := testState(t)
	step := new(StepConfigureVMX)

	state.Put("vmx_path", "some_bad_path")

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionHalt {
		t.Fatalf("bad action: %#v. Should halt when vmxPath is bad", action)
	}
	if _, ok := state.GetOk("error"); !ok {
		t.Fatal("should store error in state when vmxPath is bad")
	}

}

func TestStepConfigureVMX_DefaultDiskAndCDROMTypes(t *testing.T) {
	type testCase struct {
		inDiskAdapter  string
		inCDromAdapter string
		expectedOut    DiskAndCDConfigData
		reason         string
	}
	testcases := []testCase{
		{
			inDiskAdapter:  "",
			inCDromAdapter: "",
			expectedOut: DiskAndCDConfigData{
				ScsiPresent:         "TRUE",
				ScsiDiskAdapterType: "",
				SataPresent:         "FALSE",
				NvmePresent:         "FALSE",

				DiskType:                  "scsi",
				CdromType:                 "ide",
				CdromTypePrimarySecondary: "0",
			},
			reason: "Test that default creases scsi disk with ide cd",
		},
		{
			inDiskAdapter:  "ide",
			inCDromAdapter: "",
			expectedOut: DiskAndCDConfigData{
				ScsiPresent:         "FALSE",
				ScsiDiskAdapterType: "lsilogic",
				SataPresent:         "FALSE",
				NvmePresent:         "FALSE",

				DiskType:                  "ide",
				CdromType:                 "ide",
				CdromTypePrimarySecondary: "1",
			},
			reason: "ide disk adapter should pass through and not get defaulted to scsi",
		},
		{
			inDiskAdapter:  "sata",
			inCDromAdapter: "",
			expectedOut: DiskAndCDConfigData{
				ScsiPresent:         "FALSE",
				ScsiDiskAdapterType: "lsilogic",
				SataPresent:         "TRUE",
				NvmePresent:         "FALSE",

				DiskType:                  "sata",
				CdromType:                 "sata",
				CdromTypePrimarySecondary: "1",
			},
			reason: "when disk is set to sata, cdromtype should also default to sata",
		},
		{
			inDiskAdapter:  "nvme",
			inCDromAdapter: "",
			expectedOut: DiskAndCDConfigData{
				ScsiPresent:         "FALSE",
				ScsiDiskAdapterType: "lsilogic",
				SataPresent:         "TRUE",
				NvmePresent:         "TRUE",

				DiskType:                  "nvme",
				CdromType:                 "sata",
				CdromTypePrimarySecondary: "0",
			},
			reason: "when disk is set to nvme, cdromtype should default to sata",
		},
		{
			inDiskAdapter:  "scsi",
			inCDromAdapter: "",
			expectedOut: DiskAndCDConfigData{
				ScsiPresent:         "TRUE",
				ScsiDiskAdapterType: "lsilogic",
				SataPresent:         "FALSE",
				NvmePresent:         "FALSE",

				DiskType:                  "scsi",
				CdromType:                 "ide",
				CdromTypePrimarySecondary: "0",
			},
			reason: "when disk is set to scsi, adapter should default back to lsilogic",
		},
		{
			inDiskAdapter:  "scsi",
			inCDromAdapter: "scsi",
			expectedOut: DiskAndCDConfigData{
				ScsiPresent:         "TRUE",
				ScsiDiskAdapterType: "lsilogic",
				SataPresent:         "FALSE",
				NvmePresent:         "FALSE",

				DiskType:                  "scsi",
				CdromType:                 "scsi",
				CdromTypePrimarySecondary: "0",
			},
			reason: "when cdrom adapter is set, it should override the default",
		},
	}
	for _, tc := range testcases {
		diskConfigData := DefaultDiskAndCDROMTypes(tc.inDiskAdapter, tc.inCDromAdapter)
		assert.Equal(t, diskConfigData, tc.expectedOut, tc.reason)
	}
}
