# Copyright IBM Corp. 2013, 2025
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

variable "build_username" {
  description = "The username for the user account for the build process."
  type        = string
  default     = "packer"
}

variable "build_password" {
  description = "The password for the user account for the build process."
  type        = string
  default     = "packer"
  sensitive   = true
}

variable "boot_wait" {
  description = "The time to wait after booting the initial virtual machine before sending the boot command."
  type        = string
  default     = "10s"
}

variable "ssh_timeout" {
  description = "The maximum time to wait for SSH to become available on the guest operating system."
  type        = string
  default     = "10000s"
}

variable "boot_command" {
  description = "A list of boot commands to send to the virtual machine console during boot sequence."
  type        = list(string)
  default     = ["<esc><wait>", "<esc><wait>", "<enter><wait>", "/install/vmlinuz<wait>", " initrd=/install/initrd.gz", " auto-install/enable=true", " debconf/priority=critical", " preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg<wait>", " -- <wait>", "<enter><wait>"]
}

variable "data_directory" {
  description = "The directory path for the kickstart files."
  type        = string
  default     = "data/debian"
}

variable "disk_size" {
  description = "The size of the virtual disk in megabytes."
  type        = number
  default     = 20480
}

variable "cdrom_adapter_type" {
  description = "The adapter type for the CD-ROM drive. Must be 'sata' for ARM64 builds."
  type        = string
  default     = "sata"
}

variable "disk_adapter_type" {
  description = "The adapter type for the virtual disk. Must be 'sata' for ARM64 builds."
  type        = string
  default     = "sata"
}

variable "network_adapter_type" {
  description = "The network adapter type for the virtual machine."
  type        = string
  default     = "e1000"
}

variable "guest_os_type" {
  description = "The guest operating system type identifier for the virtual machine."
  type        = string
  default     = "debian-64"
}

variable "version" {
  description = "The virtual machine hardware version compatibility level."
  type        = number
  default     = 21
}

variable "iso_url" {
  description = "The URL or local path to the ISO file for the operating system installation."
  type        = string
}

variable "iso_checksum" {
  description = "The checksum of the ISO file to verify integrity after the download."
  type        = string
}

variable "cpus" {
  description = "The number of virtual CPUs to assign to the virtual machine."
  type        = number
  default     = 2
}

variable "memory" {
  description = "The amount of memory in megabytes to assign to the virtual machine."
  type        = number
  default     = 2048
}

variable "vmx_data" {
  description = "The additional data to add to the virtual machine configuration file."
  type        = map(string)
  default     = {}
}

variable "vm_guest_os_language" {
  description = "The language setting for the guest operating system installation."
  type        = string
  default     = "en"
}

variable "vm_guest_os_keyboard" {
  description = "The keyboard layout for the guest operating system."
  type        = string
  default     = "us"
}

variable "vm_guest_os_timezone" {
  description = "The timezone setting for the guest operating system."
  type        = string
  default     = "UTC"
}

variable "output_directory" {
  description = "The output directory for the virtual machine."
  type        = string
  default     = null
}


