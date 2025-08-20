# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

packer {
  required_version = ">= 1.7.0"
  required_plugins {
    vmware = {
      version = ">= 2.0.0"
      source  = "github.com/hashicorp/vmware"
    }
  }
}

build {
  sources = ["source.vmware-iso.debian"]
}
