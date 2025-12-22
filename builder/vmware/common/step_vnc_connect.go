// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/tenthirtyam/go-vnc"
)

// StepVNCConnect represents a step for establishing VNC connection to the virtual machine.
type StepVNCConnect struct {
	VNCEnabled   bool
	DriverConfig *DriverConfig
}

// Run executes the VNC connection step, establishing either websocket or direct VNC connection.
func (s *StepVNCConnect) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if !s.VNCEnabled {
		return multistep.ActionContinue
	}
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Connecting to VNC...")
	c, err := s.ConnectVNC(ctx, state)
	if err != nil {
		err = fmt.Errorf("error connecting to VNC: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("vnc_conn", c)
	return multistep.ActionContinue
}

// ConnectVNC establishes a direct VNC connection to the virtual machine.
func (s *StepVNCConnect) ConnectVNC(ctx context.Context, state multistep.StateBag) (*vnc.ClientConn, error) {
	vncIp := state.Get("vnc_ip").(string)
	vncPort := state.Get("vnc_port").(int)
	vncPassword := state.Get("vnc_password")

	nc, err := net.Dial("tcp", fmt.Sprintf("%s:%d", vncIp, vncPort))
	if err != nil {
		err := fmt.Errorf("error connecting to VNC: %s", err)
		state.Put("error", err)
		return nil, err
	}

	auth := []vnc.ClientAuth{new(vnc.ClientAuthNone)}
	if vncPassword != nil && len(vncPassword.(string)) > 0 {
		auth = []vnc.ClientAuth{&vnc.PasswordAuth{Password: vncPassword.(string)}}
	}

	c, err := vnc.ClientWithContext(ctx, nc, &vnc.ClientConfig{Auth: auth, Exclusive: true})
	if err != nil {
		err := fmt.Errorf("error handshaking with VNC: %s", err)
		state.Put("error", err)
		return nil, err
	}
	return c, nil
}

// Cleanup performs any necessary cleanup after the VNC connection step completes.
func (s *StepVNCConnect) Cleanup(multistep.StateBag) {
	// No cleanup needed.
}
