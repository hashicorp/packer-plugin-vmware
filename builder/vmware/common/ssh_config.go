// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"log"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type SSHConfig struct {
	Comm communicator.Config `mapstructure:",squash"`

	// TODO: Deprecated. Remove in next major release
	SSHSkipRequestPty bool `mapstructure:"ssh_skip_request_pty"`
}

func (c *SSHConfig) Prepare(ctx *interpolate.Context) []error {
	if c.SSHSkipRequestPty {
		c.Comm.SSHPty = false
		log.Printf("[WARN] 'ssh_skip_request_pty' is deprecated and will be removed in the next major release.")
	}

	return c.Comm.Prepare(ctx)
}
