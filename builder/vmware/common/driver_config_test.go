// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

func TestDriverConfigPrepare(t *testing.T) {
	tc := []struct {
		name           string
		config         *DriverConfig
		expectedConfig *DriverConfig
		errs           []error
	}{
		{
			name:   "Set default values",
			config: new(DriverConfig),
			expectedConfig: &DriverConfig{
				FusionAppPath: "/Applications/VMware Fusion.app",
			},
			errs: nil,
		},
		{
			name: "Override default values",
			config: &DriverConfig{
				FusionAppPath: "foo",
			},
			expectedConfig: &DriverConfig{
				FusionAppPath: "foo",
			},
			errs: nil,
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			errs := c.config.Prepare(interpolate.NewContext())
			if len(errs) != len(c.errs) {
				t.Fatalf("bad: \n expected '%v' \nactual '%v'", c.errs, errs)
			}
			for i, err := range errs {
				if err.Error() != c.errs[i].Error() {
					t.Fatalf("bad: \n expected '%v' \nactual '%v'", c.errs[i], err)
				}
			}
		})
	}
}
