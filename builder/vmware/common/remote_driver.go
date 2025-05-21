// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type RemoteDriver interface {
	Driver

	// UploadISO uploads an ISO file to a remote destination.
	UploadISO(path string, checksum string, ui packersdk.Ui) (string, error)

	// RemoveCache removes a cached file or resource from the specified local path.
	RemoveCache(localPath string) error

	// Register adds a virtual machine to the inventory using the provided path to the VMX file.
	Register(path string) error

	// Unregister removes a virtual machine from the inventory using the provided path to the VMX file.
	Unregister(path string) error

	// Destroy removes the virtual machine from the remote inventory and deletes its associated resources.
	Destroy() error

	// IsDestroyed checks if the virtual machine has been successfully destroyed.
	IsDestroyed() (bool, error)

	// upload transfers a local file to a remote destination.
	upload(dst, src string, ui packersdk.Ui) error

	// Download transfers a file from a remote source location to a local destination path.
	Download(src, dst string) error

	// ReloadVM reloads the virtual machine configuration on the remote hypervisor and applies any necessary updates.
	ReloadVM() error
}
