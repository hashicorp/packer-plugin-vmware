// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func testLocalOutputDir(t *testing.T) *LocalOutputDir {
	td, err := os.MkdirTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	os.RemoveAll(td)

	result := new(LocalOutputDir)
	result.SetOutputDir(td)
	return result
}

func testStepShutdownState(t *testing.T) multistep.StateBag {
	dir := testLocalOutputDir(t)
	if err := dir.MkdirAll(); err != nil {
		t.Fatalf("err: %s", err)
	}

	state := testState(t)
	state.Put("communicator", new(packersdk.MockCommunicator))
	state.Put("dir", dir)
	state.Put("vmx_path", "foo")
	return state
}

func TestStepShutdown_impl(t *testing.T) {
	var _ multistep.Step = new(StepShutdown)
}

func TestStepShutdown_command(t *testing.T) {
	state := testStepShutdownState(t)
	step := new(StepShutdown)
	step.Command = "foo"
	step.Timeout = 10 * time.Second
	step.Testing = true

	comm := state.Get("communicator").(*packersdk.MockCommunicator)
	driver := state.Get("driver").(*DriverMock)
	driver.IsRunningResult = true

	// Set not running after some time
	go func() {
		time.Sleep(100 * time.Millisecond)
		driver.Lock()
		defer driver.Unlock()
		driver.IsRunningResult = false
	}()

	resultCh := make(chan multistep.StepAction, 1)
	go func() {
		resultCh <- step.Run(context.Background(), state)
	}()

	select {
	case <-resultCh:
		t.Fatal("should not have returned so quickly")
	case <-time.After(50 * time.Millisecond):
	}

	var action multistep.StepAction
	select {
	case action = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("should've returned by now")
	}

	// Test the run
	if action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if driver.StopCalled {
		t.Fatal("stop should not be called")
	}

	if !comm.StartCalled {
		t.Fatal("start should be called")
	}
	if comm.StartCmd.Command != "foo" {
		t.Fatalf("bad: %#v", comm.StartCmd.Command)
	}

	// Clean up the created test output directory
	dir := state.Get("dir").(*LocalOutputDir)
	if err := dir.RemoveAll(); err != nil {
		t.Fatalf("Error cleaning up directory: %s", err)
	}
}

func TestStepShutdown_noCommand(t *testing.T) {
	state := testStepShutdownState(t)
	step := new(StepShutdown)

	comm := state.Get("communicator").(*packersdk.MockCommunicator)
	driver := state.Get("driver").(*DriverMock)

	// Test the run
	if action := step.Run(context.Background(), state); action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if !driver.StopCalled {
		t.Fatal("stop should be called")
	}
	if driver.StopPath != "foo" {
		t.Fatal("should call with right path")
	}

	if comm.StartCalled {
		t.Fatal("start should not be called")
	}

	// Clean up the created test output directory
	dir := state.Get("dir").(*LocalOutputDir)
	if err := dir.RemoveAll(); err != nil {
		t.Fatalf("Error cleaning up directory: %s", err)
	}
}

func TestStepShutdown_locks(t *testing.T) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("This test is only run with PACKER_ACC=1 due to the requirement of access to the VMware binaries.")
	}

	state := testStepShutdownState(t)
	step := new(StepShutdown)
	step.Testing = true

	dir := state.Get("dir").(*LocalOutputDir)
	comm := state.Get("communicator").(*packersdk.MockCommunicator)
	driver := state.Get("driver").(*DriverMock)

	// Create some lock files
	lockPath := filepath.Join(dir.dir, "nope.lck")
	err := os.WriteFile(lockPath, []byte("foo"), 0644) //nolint:gosec
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Remove the lock file after a certain time
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.Remove(lockPath)
	}()

	resultCh := make(chan multistep.StepAction, 1)
	go func() {
		resultCh <- step.Run(context.Background(), state)
	}()

	select {
	case <-resultCh:
		t.Fatal("should not have returned so quickly")
	case <-time.After(50 * time.Millisecond):
	}

	var action multistep.StepAction
	select {
	case action = <-resultCh:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("should've returned by now")
	}

	// Test the run
	if action != multistep.ActionContinue {
		t.Fatalf("bad action: %#v", action)
	}
	if _, ok := state.GetOk("error"); ok {
		t.Fatal("should NOT have error")
	}

	// Test the driver
	if !driver.StopCalled {
		t.Fatal("stop should be called")
	}
	if driver.StopPath != "foo" {
		t.Fatal("should call with right path")
	}

	if comm.StartCalled {
		t.Fatal("start should not be called")
	}
}
