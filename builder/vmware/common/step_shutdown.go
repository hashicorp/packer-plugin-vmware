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

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

//go:generate packer-sdc struct-markdown

const (
	defaultShutdownTimeout = 5 * time.Minute
	pollInterval           = 150 * time.Millisecond
	postShutdownTimeout    = 120 * time.Second
	fileReleaseDelay       = 5 * time.Second
)

type StepShutdown struct {
	Command string
	Timeout time.Duration
	Testing bool
	Disable bool
}

type ShutdownDisableConfig struct {
	// Disables the default shutdown process. This is useful for debugging.
	// Normally, Packer halts a virtual machine after all provisioners have
	// run when no `shutdown_command` is defined. If set to `true`, the virtual
	// machine will not be halted, but the plugin but will assume that you will
	// send the stop signal (_e.g._, a script or the final provisioner).
	//
	// ~> **Note:** Takes precedence over `shutdown_command`.
	//
	// ~> **Note:** The default five (5) minute timeout will be observed unless
	// the `shutdown_timeout` option is set.
	ShutdownDisable bool `mapstructure:"shutdown_disabled"`
}

func (s *StepShutdown) Prepare(comm communicator.Config) (warnings []string, errs []error) {

	if s.Timeout == 0 {
		s.Timeout = defaultShutdownTimeout
	}

	if comm.Type == "none" && s.Command != "" {
		warnings = append(warnings, "The parameter 'shutdown_command' is ignored as it requires a 'communicator'.")
	}

	return
}

func (s *StepShutdown) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	comm := state.Get("communicator").(packersdk.Communicator)
	dir := state.Get("dir").(OutputDir)
	driver := state.Get("driver").(Driver)
	vmxPath := state.Get("vmx_path").(string)

	// Virtual machine shutdown process:
	// 1. If shutdown is disabled, wait for virtual machine to shut down until
	//    the timeout expired. Halt build when timeout is reached.
	// 2. If a shutdown command is defined, execute it and wait until timeout.
	//    Halt build when timeout is reached.
	// 3. Otherwise, stop the virtual machine.
	if s.Disable {
		ui.Say("Shutdown disabled; waiting for another process to stop virtual machine...")
		if action := waitForShutdown(driver, vmxPath, s.Timeout, ui, state); action == multistep.ActionHalt {
			return multistep.ActionHalt
		}
	} else if s.Command != "" {
		ui.Say("Shutting down virtual machine...")
		log.Printf("Running shutdown command on virtual machine: %s", s.Command)

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
		if action := waitForShutdown(driver, vmxPath, s.Timeout, ui, state); action == multistep.ActionHalt {
			log.Printf("Shutdown stdout: %s", stdout.String())
			log.Printf("Shutdown stderr: %s", stderr.String())
			return multistep.ActionHalt
		}
	} else {
		ui.Say("Stopping the virtual machine...")
		if err := driver.Stop(vmxPath); err != nil {
			err = fmt.Errorf("error stopping virtual machine: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Notify the user that the virtual machine has been shut down.
	ui.Say("Virtual machine has been shut down.")
	log.Printf("Virtual machine has been shut down.")

	// Wait for the virtual machine to finish cleaning up after shutdown.
	ui.Say("Waiting for virtual machine post-shutdown clean up to complete...")
	log.Printf("Waiting for virtual machine post-shutdown clean up to complete.")
	lockRegex := regexp.MustCompile(`(?i)\.lck$`)
	timer := time.After(postShutdownTimeout)
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
				log.Println("No more lock files found are present.")
				break
			}

			if len(locks) == 1 && strings.HasSuffix(locks[0], ".vmx.lck") {
				log.Println("The .vmx.lck file is the only lock file present. Assuming clean.")
				break
			}

			log.Printf("Waiting on virtual machine lock files: %#v", locks)
		}

		select {
		case <-timer:
			log.Println("Post-shutdown virtual machine clean up timeout expired. Assuming clean.")
			break LockWaitLoop
		case <-time.After(pollInterval):
		}
	}

	if !s.Testing {
		// Windows takes a while to yield control of the files when the
		// process is exiting. Ubuntu and macOS will yield control of the files;
		// however, the hypervisor may overwrite the cleanup steps that run
		// after this so this sleep is necessary.
		//
		// TODO: Investigate if there is a better way to handle this.
		time.Sleep(fileReleaseDelay)
	}

	// Notify the user that the virtual machine has been cleaned up.
	// This is the last step in the shutdown process.
	ui.Say("Post-shutdown virtual machine clean up complete.")
	log.Println("Post-shutdown virtual machine clean up complete.")
	return multistep.ActionContinue
}

func (s *StepShutdown) Cleanup(state multistep.StateBag) {}

// waitForShutdown waits for the virtual machine to shut down until the
// timeout expires. If the virtual machine is still running after the timeout,
// an error is returned.
func waitForShutdown(driver Driver, vmxPath string, timeout time.Duration, ui packer.Ui, state multistep.StateBag) multistep.StepAction {
	ui.Say(fmt.Sprintf("Waiting a maximum of %s for virtual machine to shut down...", timeout))
	log.Printf("Waiting a maximum of %s for virtual machine to shut down.", timeout)
	shutdownTimer := time.After(timeout)
	for {
		running, _ := driver.IsRunning(vmxPath)
		if !running {
			break
		}

		select {
		case <-shutdownTimer:
			err := errors.New("timeout expired waiting for virtual machine to shut down")
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		default:
			time.Sleep(pollInterval)
		}
	}
	return multistep.ActionContinue
}
