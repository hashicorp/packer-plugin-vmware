# © Broadcom. All Rights Reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
# SPDX-License-Identifier: MPL-2.0

locals {
  output_dir = var.output_directory == null ? "builds/${var.vm_name}" : var.output_directory
}

source "vmware-vmx" "debian" {
  source_path      = var.source_path
  vm_name          = var.vm_name
  guest_os_type    = var.guest_os_type
  version          = var.version
  linked           = var.linked
  headless         = var.headless
  skip_compaction  = var.skip_compaction
  ssh_username     = var.ssh_username
  ssh_password     = var.ssh_password
  ssh_timeout      = var.ssh_timeout
  output_directory = local.output_dir
  shutdown_command = "echo '${var.ssh_password}' | sudo -S shutdown -P now"
  vmx_data         = var.vmx_data
}
