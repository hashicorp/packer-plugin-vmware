# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "vm_name" {
  description = "The name of the virtual machine."
  type        = string
}

variable "headless" {
  description = "Option to run the virtual machine without the hypervisor GUI console."
  type        = bool
  default     = false
}

variable "ssh_username" {
  description = "The username for connecting to the virtual machine."
  type        = string
  default     = "packer"
}

variable "ssh_password" {
  description = "The password for connecting to the virtual machine."
  type        = string
  default     = "packer"
  sensitive   = true
}

variable "ssh_timeout" {
  description = "The maximum time to wait for SSH to become available on the guest operating system."
  type        = string
  default     = "20m"
}

variable "source_path" {
  description = "The path to the source VMX, OVF, or OVA file to clone."
  type        = string
}

variable "guest_os_type" {
  description = "The guest operating system type identifier for the virtual machine."
  type        = string
  default     = "debian-64"
}

variable "version" {
  description = "The virtual machine hardware version. Only used when cloning from OVF/OVA. Default is 21. Minimum is 19."
  type        = number
  default     = 21
}

variable "linked" {
  description = "Create a linked clone instead of a full clone."
  type        = bool
  default     = false
}

variable "vmx_data" {
  description = "The additional data to add to the virtual machine configuration file."
  type        = map(string)
  default     = {}
}

variable "skip_compaction" {
  description = "Option to skip disk compaction after provisioning."
  type        = bool
  default     = false
}

variable "output_directory" {
  description = "The output directory for the virtual machine."
  type        = string
  default     = null
}