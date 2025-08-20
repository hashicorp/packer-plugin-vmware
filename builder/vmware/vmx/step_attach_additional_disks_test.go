// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vmx

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	vmwcommon "github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

func TestGetNextAvailableUnit(t *testing.T) {
	step := &StepAttachAdditionalDisks{}

	vmxData := map[string]string{
		// Existing SCSI units
		"scsi0:0.present": "TRUE",
		"scsi0:2.present": "TRUE",
		// Different adapter should be ignored
		"sata0:1.present": "TRUE",
		// Non-device key should be ignored
		"displayName": "vm",
	}

	next := step.getNextAvailableUnit(vmxData, "scsi")
	if next != 3 {
		t.Fatalf("expected next scsi unit 3, got %d", next)
	}

	nextSata := step.getNextAvailableUnit(vmxData, "sata")
	if nextSata != 2 {
		t.Fatalf("expected next sata unit 2, got %d", nextSata)
	}
}

func TestRun_AttachAdditionalDisks_SkipsReservedUnit7(t *testing.T) {
	tmpDir := t.TempDir()
	vmxPath := filepath.Join(tmpDir, "test.vmx")

	initial := map[string]string{
		"config.version":    "8",
		"virtualHW.version": "19",
		// Existing SCSI units such that next is 7 (reserved)
		"scsi0:0.present": "TRUE",
		"scsi0:6.present": "TRUE",
	}
	if err := vmwcommon.WriteVMX(vmxPath, initial); err != nil {
		t.Fatalf("failed to write initial vmx: %v", err)
	}

	// State and UI
	state := new(multistep.BasicStateBag)
	state.Put("ui", packersdk.TestUi(t))
	state.Put("vmx_path", vmxPath)

	// Config
	cfg := &Config{}
	cfg.VMName = "testvm"
	cfg.DiskAdapterType = ""
	cfg.DiskTypeId = ""
	cfg.AdditionalDiskSize = []uint{1024, 2048}

	// Manually prepare just the DiskConfig to set defaults (DiskName will be set to "disk")
	errs := cfg.DiskConfig.Prepare(nil)
	if len(errs) > 0 {
		t.Fatalf("failed to prepare disk config: %v", errs)
	}

	state.Put("config", cfg)

	step := &StepAttachAdditionalDisks{}
	res := step.Run(context.Background(), state)
	if res != multistep.ActionContinue {
		t.Fatalf("expected ActionContinue, got %v", res)
	}

	updated, err := vmwcommon.ReadVMX(vmxPath)
	if err != nil {
		t.Fatalf("failed to read updated vmx: %v", err)
	}

	// Existing entries remain
	if updated["scsi0:6.present"] != "TRUE" {
		t.Fatalf("expected existing scsi0:6.present to remain TRUE")
	}

	// Ensure reserved unit 7 is not used
	if _, ok := updated["scsi0:7.present"]; ok {
		t.Fatalf("reserved unit 7 should not be present")
	}

	// First additional disk at 8
	if got := updated["scsi0:8.present"]; got != "TRUE" {
		t.Fatalf("expected scsi0:8.present TRUE, got %q", got)
	}
	if got := updated["scsi0:8.filename"]; got != "disk-1.vmdk" {
		t.Fatalf("expected scsi0:8.filename disk-1.vmdk, got %q", got)
	}

	// Second additional disk at 9
	if got := updated["scsi0:9.present"]; got != "TRUE" {
		t.Fatalf("expected scsi0:9.present TRUE, got %q", got)
	}
	if got := updated["scsi0:9.filename"]; got != "disk-2.vmdk" {
		t.Fatalf("expected scsi0:9.filename disk-2.vmdk, got %q", got)
	}

	// If DiskTypeId is empty, writethrough should not be set
	if _, ok := updated["scsi0:8.writethrough"]; ok {
		t.Fatalf("did not expect writethrough for scsi0:8 when DiskTypeId is empty")
	}
	if _, ok := updated["scsi0:9.writethrough"]; ok {
		t.Fatalf("did not expect writethrough for scsi0:9 when DiskTypeId is empty")
	}
}

func TestRun_NoAdditionalDisks_NoOp(t *testing.T) {
	tmpDir := t.TempDir()
	vmxPath := filepath.Join(tmpDir, "noop.vmx")

	initial := map[string]string{
		"config.version": "8",
	}
	if err := vmwcommon.WriteVMX(vmxPath, initial); err != nil {
		t.Fatalf("failed to write vmx: %v", err)
	}

	state := new(multistep.BasicStateBag)
	state.Put("ui", packersdk.TestUi(t))
	state.Put("vmx_path", vmxPath)

	cfg := &Config{}
	cfg.VMName = "noopvm"
	cfg.AdditionalDiskSize = nil // no additional disks
	state.Put("config", cfg)

	step := &StepAttachAdditionalDisks{}
	res := step.Run(context.Background(), state)
	if res != multistep.ActionContinue {
		t.Fatalf("expected ActionContinue, got %v", res)
	}

	after, err := vmwcommon.ReadVMX(vmxPath)
	if err != nil {
		t.Fatalf("failed to read vmx: %v", err)
	}

	if !reflect.DeepEqual(after, initial) {
		t.Fatalf("expected VMX unchanged; diff found:\ninitial=%v\nafter=%v", initial, after)
	}
}
