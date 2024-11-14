# Packer Plugin for VMware

The Packer Plugin for VMware is a plugin that can be used to create virtual machine images for use
with VMware [desktop hypervisors][desktop-hypervisors] ( VMware Fusion Pro, VMware Workstation Pro,
and VMware Workstation Player[^1] ) and [VMware vSphere Hypervisor][vsphere-hypervisor] [^2].

The plugin includes two builders which are able to create images, depending on your desired
strategy:

- [`vmware-iso`][docs-vmware-iso] - This builder creates a virtual machine, installs an operating
  system from an ISO, provisions software within the operating system, and then exports the virtual
  machine as an image. This is best for those who want to start by creating an image.

- [`vmware-vmx`][docs-vmware-vmx] - This builder imports an existing virtual machine (from a`.vmx`
  file), runs provisioners on the virtual machine, and then exports the virtual machine as an image.
  This is best for those who want to start from an existing virtual machine as the source. You can
  feed the artifact of this builder back into Packer to iterate on an image.

## Supported Hypervisors

The following hypervisors are supported by this plugin.

**Desktop Hypervisor**:

- VMware Fusion Pro (macOS)
- VMware Workstation Pro (Linux and Windows)
- VMware Workstation Player (Linux and Windows) [^1]

**Bare Metal Hypervisor**:

- VMware vSphere Hypervisor [^2]

The plugin supports versions in accordance with the [Broadcom Product Lifecycle][product-lifecycle].

## Requirements

**Go**:

- [Go 1.22.8][golang-install]

  Required if building the plugin.

## Usage

For a examples on how to use this plugin with Packer refer to the [example](example/) directory of
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
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/vmware"
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

### Configuration

For more information on how to configure the plugin, please see the plugin documentation.

- `vmware-iso` [builder documentation][docs-vmware-iso]
- `vmware-vmx` [builder documentation][docs-vmware-vmx]

## Contributing

If you discover a bug or would like to suggest a feature or an enhancement, please use the GitHub
[issues][issues]. Issues are monitored by the maintainers and are prioritized based on the
criticality and community reactions.

Before opening an issue, please check existing open or recently closed issues to avoid duplicates.

When opening an issue, please include as much information as possible, such as:

- A minimal reproducible example or a series of reproduction steps.
- Details about your environment or deployment that might be unusual.

Please review the [contribution guidelines][contributing] before submitting a pull request.

For enhancements or features, please open an issue to discuss before submitting.

For comprehensive details on contributing, refer to the [contribution guidelines][contributing].

[^1]:
    Support for VMware Workstation Player is deprecated and will be removed in a future release.
    Read more about [discontinuation of VMware Workstation Player][footnote-player-discontinuation].
    The project will continue to provide bug fixes; however, enhancements for this platform will
    no longer be addressed.

[^2]:
    Support for VMware vSphere Hypervisor (ESXi) is deprecated and will be removed in a future release.
    Please transition to using the [Packer Plugin for VMware vSphere][footnote-packer-plugin-vsphere].
    The project will continue to provide bug fixes; however, enhancements for this platform will
    no longer be addressed.

[contributing]: .github/CONTRIBUTING.md
[issues]: https://github.com/hashicorp/packer-plugin-vmware/issues
[vsphere-hypervisor]: https://www.vmware.com/products/vsphere-hypervisor.html
[desktop-hypervisors]: https://www.vmware.com/products/desktop-hypervisor.html
[docs-packer-init]: https://developer.hashicorp.com/packer/docs/commands/init
[docs-packer-plugin-install]: https://developer.hashicorp.com/packer/docs/plugins/install-plugins
[docs-vmware-iso]: https://developer.hashicorp.com/packer/plugins/builders/vmware/iso
[docs-vmware-vmx]: https://developer.hashicorp.com/packer/plugins/builders/vmware/vmx
[golang-install]: https://golang.org/doc/install
[releases-vmware-plugin]: https://github.com/hashicorp/packer-plugin-vmware/releases
[product-lifecycle]: https://support.broadcom.com/group/ecx/productlifecycle
[footnote-player-discontinuation]: https://blogs.vmware.com/workstation/2024/05/vmware-workstation-pro-now-available-free-for-personal-use.html
[footnote-packer-plugin-vsphere]: https://developer.hashicorp.com/packer/integrations/hashicorp/vsphere
