// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"log"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
	"github.com/vmware/packer-plugin-vmware/builder/vmware/iso"
	"github.com/vmware/packer-plugin-vmware/builder/vmware/vmx"
	"github.com/vmware/packer-plugin-vmware/version"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder("iso", new(iso.Builder))
	pps.RegisterBuilder("vmx", new(vmx.Builder))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		log.Fatal(err)
	}
}
