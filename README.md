<!--
© Broadcom. All Rights Reserved.
The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
SPDX-License-Identifier: MPL-2.0
-->

<!-- markdownlint-disable first-line-h1 no-inline-html -->

# Packer Plugin for VMware Desktop Hypervisors

The Packer Plugin for VMware Desktop Hypervisors is a plugin for creating virtual
machine images for use with VMware [desktop hypervisors][desktop-hypervisors], VMware
Fusion Pro and VMware Workstation Pro.

The plugin includes two builders for creating virtual machine images, depending on your
desired strategy:

- [`vmware-iso`][docs-vmware-iso] - This builder creates a virtual machine, installs a
  guest operating system from an ISO, provisions software within the guest operating
  system, and then exports the virtual machine as an image. Use this builder to start by
  creating a new image.

- [`vmware-vmx`][docs-vmware-vmx] - This builder imports an existing virtual machine,
  runs provisioners on the virtual machine, and then exports the virtual machine as an
  image. Use this builder to start from an existing image as the source.

## Requirements

**Hypervisors**:

This plugin supports the following desktop hypervisors.

- VMware Fusion Pro 13 (13.6.0 and later) for macOS
- VMware Workstation Pro 17 (17.6.0 and later) for Linux and Windows

> [!TIP]
> Refer to the product documentation of the supported desktop hypervisors for system
> requirements.

> [!TIP]
> To use the export functionality of the plugin, you must install [VMware OVF Tool][download-vmware-ovftool]
> 4.6.0 or later.

> [!IMPORTANT]
> The plugin no longer supports VMware ESX as of version v2.0.0. 
>
> For VMware ESX support, please use the [Packer plugin for VMware vSphere][packer-plugin-vsphere].

**Go**:

- [Go 1.24.13][golang-install] is required to build the plugin from source.

## Installation

### Using the Releases

#### Automatic Installation

Include the following in your configuration to automatically install the plugin when you
run `packer init`.

```hcl
packer {
  required_version = ">= 1.7.0"
  required_plugins {
    vmware = {
      version = ">= 2.0.0"
      source  = "github.com/vmware/vmware"
    }
  }
}
```

For more information, please refer to the Packer [documentation][docs-packer-init].

#### Manual Installation

You can install the plugin using the `packer plugins install` command.

Examples:

1. Install the latest version of the plugin:

    ```shell
    packer plugins install github.com/vmware/vmware
    ```

2. Install a specific version of the plugin:

    ```shell
    packer plugins install github.com/vmware/vmware@v2.0.0
    ```

### Using the Source

You can build from source by cloning the GitHub repository and running `make build` from
the repository root. After a successful build, the `packer-plugin-vmware` binary is
created in the root directory.

To install the compiled plugin, please refer to the Packer [documentation][docs-packer-plugin-install].

## Documentation

- Please refer to the plugin [documentation][docs-vmware-plugin] for more information on
the plugin usage.

- Please refer to the repository [`example`](example/) directory for usage examples.

## Contributing

Please read the [code of conduct][code-of-conduct] and [contribution guidelines][contributing]
to get started.

## Support

The Packer Plugin for VMware Desktop Hypervisors is supported by the maintainers and the
plugin community.

## License

© Broadcom. All Rights Reserved.</br>
The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.</br>
Licensed under the [Mozilla Public License, version 2.0][license].

[license]: LICENSE
[contributing]: .github/CONTRIBUTING.md
[code-of-conduct]: .github/CODE_OF_CONDUCT.md
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
