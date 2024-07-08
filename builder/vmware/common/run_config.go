// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const (
	defaultVNCPortMin     = 5900
	defaultVNCPortMax     = 6000
	defaultVNCBindAddress = "127.0.0.1"
)

type RunConfig struct {
	// The plugin defaults to building virtual machines by launching the
	// desktop hypervisor's graphical user interface (GUI) to display the
	// console of the virtual machine being built. When this value is set to
	// `true`, the virtual machine will start without a console; however, the
	// plugin will output VNC connection information in case you need to connect
	// to the console to debug the build process. Defaults to `false`.
	//
	// ~> **Note:** Some users have experienced issues where Packer cannot
	// properly connect to a virtual machine when using `headless`. This is
	// often attributed to the use of an evaluation license for VMware desktop
	// hypervisors. It is recommended to launch the product and accept the
	// evaluation license to resolve this if you encounter an issue with this
	// option.
	Headless bool `mapstructure:"headless" required:"false"`
	// The IP address to use for VNC access to the virtual machine. Defaults to
	// `127.0.0.1`.
	//
	// ~> **Note:** To bind to all interfaces use `0.0.0.0`.
	VNCBindAddress string `mapstructure:"vnc_bind_address" required:"false"`
	// The minimum port number to use for VNC access to the virtual machine.
	// The plugin uses VNC to type the `boot_command`. Defaults to `5900`.
	VNCPortMin int `mapstructure:"vnc_port_min" required:"false"`
	// The maximum port number to use for VNC access to the virtual machine.
	// The plugin uses VNC to type the `boot_command`. Defaults to `6000`.
	//
	// ~> **Note:** The plugin randomly selects port within the inclusive range
	// specified by `vnc_port_min` and `vnc_port_max`.
	VNCPortMax int `mapstructure:"vnc_port_max"`
	// Disables the auto-generation of a VNC password that is used to secure the
	// VNC communication with the virtual machine. Defaults to `false`.
	//
	// ~> **Important:** Must be set to `true` for remote hypervisor builds with
	// VNC enabled.
	VNCDisablePassword bool `mapstructure:"vnc_disable_password" required:"false"`
	// Connect to VNC over a websocket connection. Defaults to `false`.
	//
	// ~> **Note:** When set to `true`, any other VNC configuration options will
	// be ignored.
	//
	// ~> **Important:** Must be set to `true` for remote hypervisor builds with
	// VNC enabled.
	VNCOverWebsocket bool `mapstructure:"vnc_over_websocket" required:"false"`
	// Do not validate TLS certificate when connecting to VNC over a websocket
	// connection. Defaults to `false`.
	InsecureConnection bool `mapstructure:"insecure_connection" required:"false"`
}

func (c *RunConfig) Prepare(_ *interpolate.Context, driverConfig *DriverConfig) (warnings []string, errs []error) {
	if c.VNCOverWebsocket {
		if driverConfig.RemoteType == "" {
			errs = append(errs, fmt.Errorf("'vnc_over_websocket' can only be used with remote hypervisor builds"))
			return
		}
		if c.VNCPortMin != 0 || c.VNCPortMax != 0 || c.VNCBindAddress != "" || c.VNCDisablePassword {
			warnings = append(warnings, "[WARN] 'vnc_over_websocket' enabled, any other VNC configuration will be ignored.")
		}
		return
	}

	if c.VNCPortMin == 0 {
		c.VNCPortMin = defaultVNCPortMin
	}

	if c.VNCPortMax == 0 {
		c.VNCPortMax = defaultVNCPortMax
	}

	if c.VNCBindAddress == "" {
		c.VNCBindAddress = defaultVNCBindAddress
	}

	if c.VNCPortMin > c.VNCPortMax {
		errs = append(errs, fmt.Errorf("'vnc_port_min' must be less than 'vnc_port_max'"))
	}

	if c.VNCPortMin < 0 {
		errs = append(errs, fmt.Errorf("'vnc_port_min' must be positive"))
	}

	return
}
