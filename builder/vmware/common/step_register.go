// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

const destroyTimeout = 30 * time.Minute

type StepRegister struct {
	registeredPath string
	Format         string
	KeepRegistered bool
	SkipExport     bool
}

func (s *StepRegister) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)

	vmxPath := state.Get("vmx_path").(string)

	if remoteDriver, ok := driver.(RemoteDriver); ok {
		ui.Say("Registering virtual machine on remote hypervisor...")
		if err := remoteDriver.Register(vmxPath); err != nil {
			err := fmt.Errorf("error registering virtual machine on remote hypervisor: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		s.registeredPath = vmxPath
	}

	return multistep.ActionContinue
}

func (s *StepRegister) Cleanup(state multistep.StateBag) {
	if s.registeredPath == "" {
		return
	}

	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if (s.KeepRegistered) && (!cancelled && !halted) {
		log.Printf("Virtual machine will remain registered; `keep_registered` set to `true`.")
		return
	}

	if remoteDriver, ok := driver.(RemoteDriver); ok {
		if s.SkipExport && !cancelled && !halted {
			ui.Say("Removing virtual machine from inventory...")
			if err := remoteDriver.Unregister(s.registeredPath); err != nil {
				ui.Errorf("error removing virtual machine: %s", err)
			}

			s.registeredPath = ""
		} else {
			ui.Say("Deleting virtual machine...")
			if err := remoteDriver.Destroy(); err != nil {
				ui.Errorf("error deleting virtual machine: %s", err)
			}

			// Wait for the virtual machine to be deleted.
			start := time.Now()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				destroyed, err := remoteDriver.IsDestroyed()
				if destroyed {
					break
				}
				if err != nil {
					log.Printf("error deleting virtual machine: %s", err)
				}
				if time.Since(start) >= destroyTimeout {
					ui.Error("error removing virtual machine from inventory; manual cleanup may be required")
					break
				}
			}
		}
	}
}
