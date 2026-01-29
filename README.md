<!--
© Broadcom. All Rights Reserved.
The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
SPDX-License-Identifier: MPL-2.0
-->

<!-- markdownlint-disable first-line-h1 no-inline-html -->

# Packer Plugin for VMware Desktop Hypervisors

The Packer Plugin for VMware Desktop Hypervisors is a plugin for creating virtual machine images
for use with VMware [desktop hypervisors][desktop-hypervisors], VMware Fusion Pro and VMware
Workstation Pro.

The plugin includes two builders for creating virtual machine images, depending on your desired
strategy:

- [`vmware-iso`][docs-vmware-iso] - This builder creates a virtual machine, installs an operating
  system from an ISO, provisions software within the operating system, and then exports the virtual
  machine as an image. This is best for those who want to start by creating an image.

- [`vmware-vmx`][docs-vmware-vmx] - This builder imports an existing virtual machine (from a`.vmx`
  file), runs provisioners on the virtual machine, and then exports the virtual machine as an image.
  This is best for those who want to start from an existing virtual machine as the source. You can
  feed the artifact of this builder back into Packer to iterate on an image.

## Supported Hypervisors

This plugin supports the following desktop hypervisors.

- VMware Fusion Pro 13 (13.6.0 and later) for macOS
- VMware Workstation Pro 17 (17.6.0 and later) for Linux and Windows

> [!TIP]
> Refer to the product documentation of the supported desktop hypervisors for system requirements.

> [!TIP]
> To use the export functionality of the plugin, you must install [VMware OVF Tool][download-vmware-ovftool] 4.6.0 or
> later.

> [!IMPORTANT]
> The plugin no longer supports VMware ESX as of version v2.0.0. 
>
> For VMware ESX support, please use the [Packer plugin for VMware vSphere][packer-plugin-vsphere].

## Requirements

**Go**:

- [Go 1.24.11][golang-install]

  Required if building the plugin.

## Usage

For examples on how to use this plugin with Packer refer to the [example](example/) directory of
the repository.

## Installation

### Using Pre-built Releases

#### Automatic Installation

Packer v1.7.0 and later supports the `packer init` command which enables the automatic installation
of Packer plugins. For more information, see the [Packer documentation][docs-packer-init].

To install this plugin, copy and paste this code (HCL2) into your Packer configuration and run
`packer init`.

```hcl
packer {
  required_version = ">= 1.7.0"
  required_plugins {
    vmware = {
      version = ">= 1.2.0"
      source  = "github.com/vmware/vmware"
    }
  }
}
```

#### Manual Installation

You can download the plugin from the GitHub [releases][releases-vmware-plugin]. Once you have
downloaded the latest release archive for your target operating system and architecture, extract the
release archive to retrieve the plugin binary file for your platform.

To install the downloaded plugin, please follow the Packer documentation on
[installing a plugin][docs-packer-plugin-install].

### Using the Source

If you prefer to build the plugin from sources, clone the GitHub repository locally and run the
command `go build` from the repository root directory. Upon successful compilation, a
`packer-plugin-vmware` plugin binary file can be found in the root directory.

To install the compiled plugin, please follow the Packer documentation on
[installing a plugin][docs-packer-plugin-install].

### Documentation

For more information on how to use the plugin, please refer to the [documentation][docs-vmware-plugin].

## Contributing

The Packer Plugin for VMware Desktop Hypervisors is the work of many contributors and the project team appreciates your help!

If you discover a bug or would like to suggest an enhancement, submit [an issue][issues].

If you would like to submit a pull request, please read the [contribution guidelines][contributing] to get started. In case of enhancement or feature contribution, we kindly ask you to open an issue to discuss it beforehand.

## Support

The Packer Plugin for VMware Desktop Hypervisors is supported by the maintainers and the plugin community.

## License

© Broadcom. All Rights Reserved.
The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.

The Packer Plugin for VMware Desktop Hypervisors is available under the [Mozilla Public License, version 2.0][license] license.

[license]: LICENSE
[contributing]: .github/CONTRIBUTING.md
[issues]: https://github.com/vmware/packer-plugin-vmware/issues
[desktop-hypervisors]: https://www.vmware.com/products/desktop-hypervisor.html
[docs-packer-init]: https://developer.hashicorp.com/packer/docs/commands/init
[docs-packer-plugin-install]: https://developer.hashicorp.com/packer/docs/plugins/install-plugins
[docs-vmware-plugin]: https://developer.hashicorp.com/packer/integrations/vmware/vmware/latest/
[docs-vmware-iso]: https://developer.hashicorp.com/packer/integrations/vmware/vmware/latest/components/builder/iso
[docs-vmware-vmx]: https://developer.hashicorp.com/packer/integrations/vmware/vmware/latest/components/builder/vmx
[golang-install]: https://golang.org/doc/install
[releases-vmware-plugin]: https://github.com/vmware/packer-plugin-vmware/releases
[packer-plugin-vsphere]: https://developer.hashicorp.com/packer/integrations/vmware/vsphere
[download-vmware-ovftool]: https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest
