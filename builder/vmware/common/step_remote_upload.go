// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepRemoteUpload represents a step to upload a file to a remote location.
type StepRemoteUpload struct {
	Key       string
	Message   string
	DoCleanup bool
	Checksum  string
}

func (s *StepRemoteUpload) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packersdk.Ui)

	remote, ok := driver.(RemoteDriver)
	if !ok {
		return multistep.ActionContinue
	}

	path, ok := state.Get(s.Key).(string)
	if !ok {
		return multistep.ActionContinue
	}

	if esxi, ok := remote.(*EsxiDriver); ok {
		remotePath := esxi.CachePath(path)

		if esxi.VerifyChecksum(s.Checksum, remotePath) {
			ui.Say("Remote cache verified; skipping remote upload...")
			state.Put(s.Key, remotePath)
			return multistep.ActionContinue
		}

	}

	ui.Say(s.Message)
	log.Printf("[INFO] Remote uploading: %s", path)
	newPath, err := remote.UploadISO(path, s.Checksum, ui)
	if err != nil {
		err = fmt.Errorf("error uploading file: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	state.Put(s.Key, newPath)

	return multistep.ActionContinue
}

func (s *StepRemoteUpload) Cleanup(state multistep.StateBag) {
	if !s.DoCleanup {
		return
	}

	driver := state.Get("driver").(Driver)

	remote, ok := driver.(RemoteDriver)
	if !ok {
		return
	}

	path, ok := state.Get(s.Key).(string)
	if !ok {
		return
	}

	log.Printf("[INFO] Cleaning up remote path: %s", path)
	err := remote.RemoveCache(path)
	if err != nil {
		log.Printf("[WARN] Failed to clean up: %s", err)
	}
}
