# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

#_preseed_V1
# Automatic installation
d-i auto-install/enable boolean true

# Preseeding only locale sets language, country and locale.
d-i debian-installer/language string ${vm_guest_os_language}
d-i debian-installer/country string US
d-i debian-installer/locale string en_US.UTF-8

d-i console-setup/ask_detect boolean false
d-i debconf/frontend select noninteractive

# Keyboard selection.
d-i keyboard-configuration/xkb-keymap select ${vm_guest_os_keyboard}
d-i keymap select ${vm_guest_os_keyboard}

choose-mirror-bin mirror/http/proxy string
d-i apt-setup/use_mirror boolean true
d-i base-installer/kernel/override-image string linux-server

### Clock and time zone setup
d-i clock-setup/utc boolean true
d-i clock-setup/utc-auto boolean true
d-i time/zone string ${vm_guest_os_timezone}

# Avoid that last message about the install being complete.
d-i finish-install/reboot_in_progress note

# This is fairly safe to set, it makes grub install automatically to the MBR
# if no other operating system is detected on the machine.
d-i grub-installer/only_debian boolean true

# This one makes grub-installer install to the MBR if it also finds some other
# OS, which is less safe as it might not be able to boot that other OS.
d-i grub-installer/with_other_os boolean true

# Set dev for grub boot
d-i grub-installer/bootdev string /dev/sda

### Mirror settings
# If you select ftp, the mirror/country string does not need to be set.
d-i mirror/country string manual
d-i mirror/http/directory string /debian/
d-i mirror/http/hostname string httpredir.debian.org
d-i mirror/http/proxy string

# This makes partman automatically partition without confirmation.
d-i partman-efi/non_efi_system boolean true
d-i partman-auto-lvm/guided_size string max
d-i partman-auto/choose_recipe select atomic
d-i partman-auto/method string lvm
d-i partman-lvm/confirm boolean true
d-i partman-lvm/confirm_nooverwrite boolean true
d-i partman-lvm/device_remove_lvm boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true
d-i partman/confirm_write_new_label boolean true

### Account setup
d-i passwd/root-login boolean false
d-i passwd/user-fullname string ${build_username}
d-i passwd/username string ${build_username}
d-i passwd/user-uid string 1000
d-i passwd/user-password password ${build_password}
d-i passwd/user-password-again password ${build_password}

# The installer will warn about weak passwords. If you are sure you know
# what you're doing and want to override it, uncomment this.
d-i user-setup/allow-password-weak boolean true
d-i user-setup/encrypt-home boolean false

### Package selection
tasksel tasksel/first multiselect standard, ssh-server
d-i pkgsel/include string openssh-server sudo bzip2 acpid cryptsetup zlib1g-dev wget curl dkms fuse make nfs-common net-tools cifs-utils rsync
d-i pkgsel/install-language-support boolean false

# Prevent packaged version of VirtualBox Guest Additions being installed:
d-i preseed/early_command string sed -i \
'/in-target/idiscover(){/sbin/discover|grep -v VirtualBox;}' \
/usr/lib/pre-pkgsel.d/20install-hwpackages

# Do not scan additional CDs
apt-cdrom-setup apt-setup/cdrom/set-first boolean false

# Use network mirror
apt-mirror-setup apt-setup/use_mirror boolean true

# disable automatic package updates
d-i pkgsel/update-policy select none
d-i pkgsel/upgrade select full-upgrade

# Disable popularity contest
popularity-contest popularity-contest/participate boolean false

# Select base install
tasksel tasksel/first multiselect standard, ssh-server

# Setup passwordless sudo for packer user
d-i preseed/late_command string \
echo "vagrant ALL=(ALL:ALL) NOPASSWD:ALL" > /target/etc/sudoers.d/vagrant && chmod 0440 /target/etc/sudoers.d/vagrant; \
# remove cdrom from apt sources
sed -i '/^deb cdrom:/s/^/#/' /target/etc/apt/sources.list
