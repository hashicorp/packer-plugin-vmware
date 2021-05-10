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
	// TODO: proper testing

	// Cleanup
	step.Cleanup(state)
}
