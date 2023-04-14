# Packer Plugin for VMware

The Packer Plugin for VMware is a multi-component plugin that can be used with
[Packer][packer] to create virtual machine images for use with VMware products.

The plugin includes two builders which are able to create images, depending on
your desired strategy:

- [`vmware-iso`][docs-vmware-iso] - This builder creates a virtual machine,
  installs an operating system from an ISO, provisions software within the
  operating system, and then exports the virtual machine as an image. This
  is best for those who want to start by creating a base image.

- [`vmware-vmx`][docs-vmware-vmx] - This builder imports an existing virtual
  machine (from a`.vmx` file), runs provisioners on the virtual machine, and
  then exports the virtual machine as an image. This is best for those who want
  to start from an existing virtual machine as the source. You can feed the
  artifact of this builder back into Packer to iterate on a machine image.

## Requirements

**Desktop Hypervisor**:

- VMware Fusion Pro (macOS)
- VMware Workstation Pro (Linux and Windows)
- VMware Workstation Player (Linux)

**Bare Metal Hypervisor**:

- VMware vSphere Hypervisor

The plugin supports versions in accordance with the VMware Product Lifecycle
Matrix from General Availability to End of General Support. Learn more:
[VMware Product Lifecycle Matrix][vmware-product-lifecycle-matrix]

**Go**:

- [Go 1.18][golang-install]

    Required if building the plugin.

## Usage

For a few examples on how to use this plugin with Packer refer to the [example](example/) template directory.
## Installation

### Using Pre-built Releases

#### Automatic Installation

Packer v1.7.0 and later supports the `packer init` command which enables the
automatic installation of Packer plugins. For more information, see the
[Packer documentation][docs-packer-init].

To install this plugin, copy and paste this code (HCL2) into your Packer
configuration and run `packer init`.

```hcl
packer {
  required_version = ">= 1.7.0"
  required_plugins {
    vmware = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/vmware"
    }
  }
}
```

#### Manual Installation

Packer v1.8.0 and later supports the `packer plugins` command which enables the
management of external plugins required by a configuration.

For example, to download and install the latest available version of this plugin,
run the following:

```console
packer plugins install github.com/hashicorp/vmware
```

For environments where the Packer host can not communicate with GitHub
(_e.g._, a dark-site), you can download [pre-built binary releases][releases-vmware-plugin]
of the plugin from GitHub. Once you have downloaded the latest release archive
for your target operating system and architecture, uncompress to retrieve the
plugin binary file for your platform.

To transfer and install the downloaded plugin, please follow the Packer
documentation on [installing a plugin][docs-packer-plugin-install].

### Using the Source

If you prefer to build the plugin from sources, clone the GitHub repository
locally and run the command `go build` from the repository root directory.
Upon successful compilation, a `packer-plugin-vmware` plugin binary file can be
found in the root directory.

To install the compiled plugin, please follow the Packer documentation on
[installing a plugin][docs-packer-plugin-install].

### Configuration

For more information on how to configure the plugin, please see the plugin
documentation.

- `vmware-iso` [builder documentation][docs-vmware-iso]
- `vmware-vmx` [builder documentation][docs-vmware-vmx]

## Contributing

- If you think you've found a bug in the code or you have a question regarding
the usage of this software, please reach out to us by opening an issue in this
GitHub repository.

- Contributions to this project are welcome: if you want to add a feature or a
fix a bug, please do so by opening a pull request in this GitHub repository.
In case of feature contribution, we kindly ask you to open an issue to discuss
it beforehand.

[docs-packer-init]: https://developer.hashicorp.com/packer/docs/commands/init
[docs-packer-plugin-install]: https://developer.hashicorp.com/packer/docs/plugins/install-plugins
[docs-vmware-iso]: https://developer.hashicorp.com/packer/plugins/builders/vmware/iso
[docs-vmware-vmx]: https://developer.hashicorp.com/packer/plugins/builders/vmware/vmx
[docs-vmware-plugin]: https://developer.hashicorp.com/packer/plugins/builders/vmware
[golang-install]: https://golang.org/doc/install
[packer]: https://www.packer.io
[releases-vmware-plugin]: https://github.com/hashicorp/packer-plugin-vmware/releases
[vmware-product-lifecycle-matrix]: https://lifecycle.vmware.com
