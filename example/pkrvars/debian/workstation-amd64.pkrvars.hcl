# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

network_adapter_type = "e1000"
iso_url              = "https://mirrors.ocf.berkeley.edu/debian-cd/13.1.0/amd64/iso-cd/debian-13.1.0-amd64-netinst.iso"
iso_checksum         = "file:https://mirrors.ocf.berkeley.edu/debian-cd/13.1.0/amd64/iso-cd/SHA256SUMS"
data_directory       = "data/debian"
guest_os_type        = "debian-64"
boot_command         = ["<wait><esc><wait>auto preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg netcfg/get_hostname={{ .Name }}<enter>"]
vm_name              = "debian-amd64"
