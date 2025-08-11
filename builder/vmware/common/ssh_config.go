// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type SSHConfig struct {
	Comm communicator.Config `mapstructure:",squash"`
}

func (c *SSHConfig) Prepare(ctx *interpolate.Context) []error {
	return c.Comm.Prepare(ctx)
}
