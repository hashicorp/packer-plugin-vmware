iso_url          = "https://cdimage.debian.org/debian-cd/current/amd64/iso-dvd/debian-11.6.0-amd64-DVD-1.iso"
iso_checksum     = "55f6f49b32d3797621297a9481a6cc3e21b3142f57d8e1279412ff5a267868d8"
data_directory   = "data/debian"
guest_os_type    = "debian-64"
hardware_version = 19
boot_command     = ["<wait><esc><wait>auto preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg netcfg/get_hostname={{ .Name }}<enter>"]
vm_name          = "debian_x86_64"

