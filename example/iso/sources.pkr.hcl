# © Broadcom. All Rights Reserved.
# The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
# SPDX-License-Identifier: MPL-2.0

locals {
  data_directory = var.data_directory == null ? "data" : var.data_directory
  output_dir = var.output_directory == null ? "builds/${var.vm_name}" : var.output_directory
}

source "vmware-iso" "debian" {
  vm_name              = var.vm_name
  headless             = var.headless
  version              = var.version
  guest_os_type        = var.guest_os_type
  cpus                 = var.cpus
  memory               = var.memory
  network_adapter_type = var.network_adapter_type
  cdrom_adapter_type   = var.cdrom_adapter_type
  disk_adapter_type    = var.disk_adapter_type
  vmx_data             = var.vmx_data
  http_content = {
    "/preseed.cfg" = templatefile("${abspath(path.root)}/${local.data_directory}/preseed.pkrtpl.hcl", {
      build_username       = var.build_username
      build_password       = var.build_password
      vm_guest_os_language = var.vm_guest_os_language
      vm_guest_os_keyboard = var.vm_guest_os_keyboard
      vm_guest_os_timezone = var.vm_guest_os_timezone
    })
  }
  iso_checksum     = var.iso_checksum
  iso_url          = var.iso_url
  boot_wait        = var.boot_wait
  boot_command     = var.boot_command
  ssh_username     = var.build_password
  ssh_password     = var.build_username
  ssh_timeout      = var.ssh_timeout
  output_directory = local.output_dir
  shutdown_command = "echo '${var.build_password}' | sudo -S -E shutdown -P now"
}
