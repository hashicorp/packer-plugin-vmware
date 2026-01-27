// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

func TestRunConfig_Prepare(t *testing.T) {
	tc := []struct {
		name           string
		config         *RunConfig
		expectedConfig *RunConfig
		driver         *DriverConfig
		errs           []error
		warnings       []string
	}{
		{
			name:   "Default configuration.",
			config: &RunConfig{},
			expectedConfig: &RunConfig{
				VNCPortMin:     5900,
				VNCPortMax:     6000,
				VNCBindAddress: "127.0.0.1",
			},
			driver:   new(DriverConfig),
			errs:     nil,
			warnings: nil,
		},
		{
			name: "Minimum port less than maximum port.",
			config: &RunConfig{
				VNCPortMin: 5000,
				VNCPortMax: 5900,
			},
			expectedConfig: &RunConfig{
				VNCPortMin:     5000,
				VNCPortMax:     5900,
				VNCBindAddress: "127.0.0.1",
			},
			driver:   new(DriverConfig),
			errs:     nil,
			warnings: nil,
		},
		{
			name: "Minimum port greater than maximum port.",
			config: &RunConfig{
				VNCPortMin: 5900,
				VNCPortMax: 5000,
			},
			expectedConfig: nil,
			driver:         new(DriverConfig),
			errs:           []error{fmt.Errorf("'vnc_port_min' must be less than 'vnc_port_max'")},
			warnings:       nil,
		},
		{
			name: "Minimum port must be positive.",
			config: &RunConfig{
				VNCPortMin: -1,
			},
			expectedConfig: nil,
			driver:         new(DriverConfig),
			errs:           []error{fmt.Errorf("'vnc_port_min' must be positive")},
			warnings:       nil,
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			warnings, errs := c.config.Prepare(interpolate.NewContext(), c.driver)
			if len(errs) != len(c.errs) {
				t.Fatalf("bad: \n expected '%v' \nactual '%v'", c.errs, errs)
			}
			for i, err := range errs {
				if err.Error() != c.errs[i].Error() {
					t.Fatalf("bad: \n expected '%v' \nactual '%v'", c.errs[i], err)
				}
			}
			if diff := cmp.Diff(warnings, c.warnings); diff != "" {
				t.Fatalf("unexpected warnings: %s", diff)
			}
			if len(c.errs) == 0 {
				if diff := cmp.Diff(c.config, c.expectedConfig,
					cmpopts.IgnoreFields(bootcommand.VNCConfig{},
						"BootConfig",
					)); diff != "" {
					t.Fatalf("unexpected config: %s", diff)
				}
			}
		})
	}
}
