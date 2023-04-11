# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

locals {
  http_directory = var.http_directory == null ? "${path.root}/http" : var.http_directory
  memory         = var.memory == null ? 2048 : var.memory
}
source "vmware-iso" "vm" {
  boot_command         = var.boot_command
  boot_wait            = "10s"
  cpus                 = 2
  cdrom_adapter_type   = var.cdrom_adapter_type
  disk_adapter_type    = var.disk_adapter_type
  guest_os_type        = var.guest_os_type
  headless             = true
  http_directory       = local.http_directory
  iso_checksum         = var.iso_checksum
  iso_url              = var.iso_url
  memory               = local.memory
  network_adapter_type = var.network_adapter_type
  output_directory     = "builds/${var.vm_name}"
  shutdown_command     = "echo 'vagrant' | sudo -S shutdown -P now"
  ssh_password         = "vagrant"
  ssh_timeout          = "10000s"
  ssh_username         = "vagrant"
  tools_upload_flavor  = var.tools_upload_flavor
  tools_upload_path    = var.tools_upload_path
  version              = var.hardware_version
  vmx_data             = var.vmx_data
}

