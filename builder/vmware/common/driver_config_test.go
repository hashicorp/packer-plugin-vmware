// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"errors"
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
				FusionAppPath:        "/Applications/VMware Fusion.app",
				RemoteDatastore:      "datastore1",
				RemoteCacheDatastore: "datastore1",
				RemoteCacheDirectory: "packer_cache",
				RemotePort:           22,
				RemoteUser:           "root",
			},
			errs: nil,
		},
		{
			name: "Override default values",
			config: &DriverConfig{
				FusionAppPath:        "foo",
				RemoteDatastore:      "set-datastore1",
				RemoteCacheDatastore: "set-datastore1",
				RemoteCacheDirectory: "set_packer_cache",
				RemotePort:           443,
				RemoteUser:           "admin",
			},
			expectedConfig: &DriverConfig{
				FusionAppPath:        "foo",
				RemoteDatastore:      "set-datastore1",
				RemoteCacheDatastore: "set-datastore1",
				RemoteCacheDirectory: "set_packer_cache",
				RemotePort:           443,
				RemoteUser:           "admin",
			},
			errs: nil,
		},
		{
			name: "Invalid remote type",
			config: &DriverConfig{
				RemoteType: "invalid",
				RemoteHost: "host",
			},
			expectedConfig: nil,
			errs:           []error{errors.New("only 'esx5' value is accepted for 'remote_type'")},
		},
		{
			name: "Remote host not set",
			config: &DriverConfig{
				RemoteType: "esx5",
			},
			expectedConfig: nil,
			errs:           []error{errors.New("'remote_host' must be specified when 'remote_type' is set")},
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
