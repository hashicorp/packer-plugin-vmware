# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "boot_command" {
  type    = list(string)
  default = ["<esc><wait>", "<esc><wait>", "<enter><wait>", "/install/vmlinuz<wait>", " initrd=/install/initrd.gz", " auto-install/enable=true", " debconf/priority=critical", " preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg<wait>", " -- <wait>", "<enter><wait>"]
}

variable "cdrom_adapter_type" {
  type    = string
  default = "sata"
}

variable "disk_size" {
  type    = number
  default = 65536
}

variable "disk_adapter_type" {
  type    = string
  default = "lsilogic"
}

variable "guest_os_type" {
  type    = string
  default = null
}

variable "hardware_version" {
  type    = number
  default = 19 # Refer to VMware versions https://kb.vmware.com/s/article/1003746
}

variable "http_directory" {
  type    = string
  default = "./http"
}
variable "iso_checksum" {
  type    = string
  default = null
}

variable "iso_url" {
  type    = string
  default = null
}

variable "memory" {
  type    = number
  default = null
}

variable "network_adapter_type" {
  type    = string
  default = null
}

variable "tools_upload_flavor" {
  type    = string
  default = null
}

variable "tools_upload_path" {
  type    = string
  default = null
}

variable "vm_name" {
  type    = string
  default = "debian"
}

variable "vmx_data" {
  type = map(string)
  default = {
    "cpuid.coresPerSocket" = "2"
  }
}

