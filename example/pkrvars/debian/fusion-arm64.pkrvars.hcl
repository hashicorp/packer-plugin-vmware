# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

cdrom_adapter_type   = "sata"
disk_adapter_type    = "sata"
network_adapter_type = "e1000e"
iso_url              = "https://mirrors.ocf.berkeley.edu/debian-cd/13.0.0/arm64/iso-cd/debian-13.0.0-arm64-netinst.iso"
iso_checksum         = "file:https://mirrors.ocf.berkeley.edu/debian-cd/13.0.0/arm64/iso-cd/SHA256SUMS"
data_directory       = "data/debian"
guest_os_type        = "arm-debian-64"
boot_command         = ["<wait><up>e<wait><down><down><down><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><right><wait>install <wait> preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg <wait>debian-installer=en_US.UTF-8 <wait>auto <wait>locale=en_US.UTF-8 <wait>kbd-chooser/method=us <wait>keyboard-configuration/xkb-keymap=us <wait>netcfg/get_hostname={{ .Name }} <wait>netcfg/get_domain={{ .Name }} <wait>fb=false <wait>debconf/frontend=noninteractive <wait>console-setup/ask_detect=false <wait>console-keymaps-at/keymap=us <wait>grub-installer/bootdev=/dev/sda <wait><f10><wait>"]
vm_name              = "debian-arm64"
vmx_data = {
  "usb_xhci.present" = "TRUE"
  "svga.autodetect"  = "TRUE"
}
