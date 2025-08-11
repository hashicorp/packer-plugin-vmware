// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepShutdown shuts down the machine. It first attempts to do so gracefully,
// but ultimately forcefully shuts it down if that fails.
type StepShutdown struct {
	Command string
	Timeout time.Duration

	// Used for testing.
	Testing bool
}

func (s *StepShutdown) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	comm := state.Get("communicator").(packersdk.Communicator)
	dir := state.Get("dir").(OutputDir)
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmxPath := state.Get("vmx_path").(string)

	if s.Command != "" {
		ui.Say("Gracefully halting virtual machine...")
		log.Printf("[INFO] Running shutdown command: %s", s.Command)

		var stdout, stderr bytes.Buffer
		cmd := &packersdk.RemoteCmd{
			Command: s.Command,
			Stdout:  &stdout,
			Stderr:  &stderr,
		}
		if err := comm.Start(ctx, cmd); err != nil {
			err = fmt.Errorf("error sending shutdown command: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Wait for the virtual machine to shut down.
		log.Printf("[INFO] Waiting up to %s for shutdown to complete", s.Timeout)
		shutdownTimer := time.After(s.Timeout)
		for {
			running, _ := driver.IsRunning(vmxPath)
			if !running {
				break
			}

			select {
			case <-shutdownTimer:
				log.Printf("Shutdown stdout: %s", stdout.String())
				log.Printf("Shutdown stderr: %s", stderr.String())
				err := errors.New("timeout waiting for virtual machine to shut down")
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			default:
				time.Sleep(shutdownPollInterval)
			}
		}
	} else {
		ui.Say("Forcibly halting virtual machine...")
		if err := driver.Stop(vmxPath); err != nil {
			err := fmt.Errorf("error stopping virtual machine: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	ui.Say("Waiting for clean up...")
	lockRegex := regexp.MustCompile(`(?i)\.lck$`)
	timer := time.After(shutdownLockTimeout)
LockWaitLoop:
	for {
		files, err := dir.ListFiles()
		if err != nil {
			log.Printf("error listing files in output directory: %s", err)
		} else {
			var locks []string
			for _, file := range files {
				if lockRegex.MatchString(file) {
					locks = append(locks, file)
				}
			}

			if len(locks) == 0 {
				log.Println("[INFO] No more lock files found. Assuming the virtual machine is clean.")
				break
			}

			if len(locks) == 1 && strings.HasSuffix(locks[0], ".vmx.lck") {
				log.Println("[INFO] Only waiting on the '.vmx.lck' file. Assuming the virtual machine is clean.")
				break
			}

			log.Printf("[INFO] Waiting on lock files: %#v", locks)
		}

		select {
		case <-timer:
			log.Println("[INFO] Reached timeout on waiting for lock files to be cleaned up. Assuming the virtual machine is clean.")
			break LockWaitLoop
		case <-time.After(shutdownLockPollInterval):
		}
	}

	if !s.Testing {
		// Wait for OS file cleanup and the hypervisor to release the lock.
		// TODO: Replace with an event that is signaled when the OS has cleaned up.
		time.Sleep(shutdownCleanupDelay)
	}

	log.Println("[INFO] Shutdown of virtual machine has completed.")
	return multistep.ActionContinue
}

func (s *StepShutdown) Cleanup(state multistep.StateBag) {}
