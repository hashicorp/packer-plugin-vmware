// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package common

import (
	"errors"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type DriverConfig struct {
	// The installation path of the VMware Fusion application. Defaults to
	// `/Applications/VMware Fusion.app`
	//
	// ~> **Note:** This is only required if you are using VMware Fusion as a
	// local desktop hypervisor and have installed it in a non-default location.
	FusionAppPath string `mapstructure:"fusion_app_path" required:"false"`
	// The type of remote hypervisor that will be used. If set, the remote
	// hypervisor will be used for the build. If not set, a local desktop
	// hypervisor (VMware Fusion or VMware Workstation) will be used.
	// Available options include `esx5` for VMware ESXi.
	RemoteType string `mapstructure:"remote_type" required:"false"`
	// The datastore where the virtual machine will be stored on the ESXi host.
	RemoteDatastore string `mapstructure:"remote_datastore" required:"false"`
	// The datastore attached to the remote hypervisor to use for the build.
	// Supporting files such as ISOs and floppies are cached in this datastore
	// during the build. Defaults to `datastore1`.
	RemoteCacheDatastore string `mapstructure:"remote_cache_datastore" required:"false"`
	// The directory path on the remote cache datastore to use for the build.
	// Supporting files such as ISOs and floppies are cached in this directory,
	// relative to the `remote_cache_datastore`, during the build. Defaults to
	// `packer_cache`.
	RemoteCacheDirectory string `mapstructure:"remote_cache_directory" required:"false"`
	// Remove items added to the remote cache after the build is complete.
	// Defaults to `false`.
	CleanUpRemoteCache bool `mapstructure:"cleanup_remote_cache" required:"false"`
	// The fully qualified domain name or IP address of the remote hypervisor
	// where the virtual machine is created.
	//
	// ~> **Note:** Required if `remote_type` is set.
	RemoteHost string `mapstructure:"remote_host" required:"false"`
	// The SSH port of the remote hypervisor. Defaults to `22`.
	RemotePort int `mapstructure:"remote_port" required:"false"`
	// The SSH username for access to the remote hypervisor. Defaults to `root`.
	RemoteUser string `mapstructure:"remote_username" required:"false"`
	// The SSH password for access to the remote hypervisor.
	RemotePassword string `mapstructure:"remote_password" required:"false"`
	// The SSH key for access to the remote hypervisor.
	RemotePrivateKey string `mapstructure:"remote_private_key_file" required:"false"`
	// Skip the validation of the credentials for access to the remote
	// hypervisor. By default, export is enabled and the plugin will validate
	// the credentials ('remote_username' and 'remote_password'), for use by
	// VMware OVF Tool, before starting the build. Defaults to `false`.
	SkipValidateCredentials bool `mapstructure:"skip_validate_credentials" required:"false"`
}

func (c *DriverConfig) Prepare(ctx *interpolate.Context) []error {
	var errs []error

	if c.FusionAppPath == "" {
		c.FusionAppPath = os.Getenv("FUSION_APP_PATH")
	}

	if c.FusionAppPath == "" {
		c.FusionAppPath = "/Applications/VMware Fusion.app"
	}

	if c.RemoteUser == "" {
		c.RemoteUser = "root"
	}

	if c.RemoteDatastore == "" {
		c.RemoteDatastore = "datastore1"
	}

	if c.RemoteCacheDatastore == "" {
		c.RemoteCacheDatastore = c.RemoteDatastore
	}

	if c.RemoteCacheDirectory == "" {
		c.RemoteCacheDirectory = "packer_cache"
	}

	if c.RemotePort == 0 {
		c.RemotePort = 22
	}

	if c.RemoteType != "" {
		if c.RemoteHost == "" {
			errs = append(errs,
				errors.New("'remote_host' must be specified when 'remote_type' is set"))
		}

		if c.RemoteType != "esx5" {
			errs = append(errs,
				errors.New("only 'esx5' value is accepted for 'remote_type'"))
		}
	}

	return errs
}

func (c *DriverConfig) Validate(SkipExport bool) error {
	if SkipExport {
		return nil
	}

	if c.RemoteType != "" && c.RemotePassword == "" {
		return errors.New(
			"'remote_password' must be provided when using 'export' with 'remote_type'",
		)
	}

	return nil
}
