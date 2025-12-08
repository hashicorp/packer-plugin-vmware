// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"log"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
	"github.com/hashicorp/packer-plugin-vmware/builder/vmware/iso"
	"github.com/hashicorp/packer-plugin-vmware/builder/vmware/vmx"
	"github.com/hashicorp/packer-plugin-vmware/version"
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
