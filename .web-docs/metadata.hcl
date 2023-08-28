# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "VMware"
  description = "The Packer Plugin for VMware with to create virtual machine images for use with VMware products."
  identifier = "packer/hashicorp/vmware"
  component {
    type = "builder"
    name = "VMware ISO"
    slug = "iso"
  }
  component {
    type = "builder"
    name = "VMware VMX"
    slug = "vmx"
  }
}
