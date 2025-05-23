// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepRun runs the created virtual machine.
type StepRun struct {
	DurationBeforeStop time.Duration
	Headless           bool

	bootTime time.Time
	vmxPath  string
}

func (s *StepRun) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)
	vmxPath := state.Get("vmx_path").(string)

	// Set the VMX path so that we know we started the machine
	s.bootTime = time.Now()
	s.vmxPath = vmxPath

	ui.Say("Powering on virtual machine...")
	if s.Headless {
		vncIpRaw, vncIpOk := state.GetOk("vnc_ip")
		vncPortRaw, vncPortOk := state.GetOk("vnc_port")
		vncPasswordRaw, vncPasswordOk := state.GetOk("vnc_password")

		if vncIpOk && vncPortOk && vncPasswordOk {
			vncIp := vncIpRaw.(string)
			vncPort := vncPortRaw.(int)
			vncPassword := vncPasswordRaw.(string)

			ui.Sayf(
				"The virtual machine will be run headless, without a GUI.\n"+
					"To view the virtual machine console, connect using VNC:\n\n"+
					"Endpoint: \"vnc://%s:%d\"\n"+
					"Password: \"%s\"", vncIp, vncPort, vncPassword)
		} else {
			ui.Say(
				"The virtual machine will be run headless, without a GUI.\n" +
					"If the build is not succeeding, enable the GUI to\n" +
					"inspect the progress of the build.")
		}
	}

	if err := driver.Start(vmxPath, s.Headless); err != nil {
		err = fmt.Errorf("error starting virtual machine: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Set the instance ID so that we can use it in the provisioners.
	state.Put("instance_id", vmxPath)

	return multistep.ActionContinue
}

func (s *StepRun) Cleanup(state multistep.StateBag) {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)

	// If the virtual machine was started, stop it.
	if s.vmxPath != "" {
		// If started, wait for the duration before stopping.
		sinceBootTime := time.Since(s.bootTime)
		waitBootTime := s.DurationBeforeStop
		if sinceBootTime < waitBootTime {
			sleepTime := waitBootTime - sinceBootTime
			ui.Sayf("Waiting %s for clean up...", sleepTime.String())
			time.Sleep(sleepTime)
		}

		// If the virtual machine is running, stop it.
		running, _ := driver.IsRunning(s.vmxPath)
		if running {
			ui.Say("Stopping virtual machine...")
			if err := driver.Stop(s.vmxPath); err != nil {
				ui.Errorf("error stopping the virtual machine: %s", err)
				ui.Say("Please perform the necessary manual operations to stop the virtual machine.")
				return
			}
		}
	}
}
