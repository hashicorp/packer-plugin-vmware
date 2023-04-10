// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepCreateSnapshot_impl(t *testing.T) {
	var _ multistep.Step = new(StepCreateSnapshot)
}

func NewTestCreateSnapshotStep() *StepCreateSnapshot {
	return &StepCreateSnapshot{
		SnapshotName: strPtr("snapshot_name"),
	}
}

func TestStepCreateSnapshot(t *testing.T) {
	state := testState(t)
	step := NewTestCreateSnapshotStep()

	state.Put("vmx_path", "foo")

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	driver := state.Get("driver").(*DriverMock)
	if !driver.CreateSnapshotCalled {
		t.Fatalf("Should have called create snapshot.")
	}

	if _, ok := state.GetOk("snapshot_skiped"); ok {
		t.Fatalf("Should NOT skip snapshot creation")
	}

	if _, ok := state.GetOk("snapshot_created"); !ok {
		t.Fatalf("Should create snapshot")
	}

	// Cleanup
	step.Cleanup(state)
}

func TestStepCreateSnapshot_skip(t *testing.T) {
	state := testState(t)
	step := NewTestCreateSnapshotStep()
	step.SnapshotName = strPtr("")

	state.Put("vmx_path", "foo")

	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	driver := state.Get("driver").(*DriverMock)
	if driver.CreateSnapshotCalled {
		t.Fatalf("Should NOT have called create snapshot.")
	}

	// Cleanup
	step.Cleanup(state)
}
