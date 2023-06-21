# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

iso_url          = "https://cdimage.debian.org/cdimage/archive/11.7.0/amd64/iso-dvd/debian-11.7.0-amd64-DVD-1.iso"
iso_checksum     = "cfbb1387d92c83f49420eca06e2d11a23e5a817a21a5d614339749634709a32f"
data_directory   = "data/debian"
guest_os_type    = "debian-64"
hardware_version = 19
boot_command     = ["<wait><esc><wait>auto preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg netcfg/get_hostname={{ .Name }}<enter>"]
vm_name          = "debian_x86_64"
