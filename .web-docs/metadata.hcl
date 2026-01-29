# © Broadcom. All Rights Reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
# SPDX-License-Identifier: MPL-2.0

# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "VMware"
  description = "A plugin for creating virtual machine images for VMware desktop hypervisors."
  identifier = "packer/vmware/vmware"
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
