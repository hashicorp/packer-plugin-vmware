<!-- markdownlint-disable first-line-h1 no-inline-html -->

The Packer Plugin for VMware Desktop Hypervisors is a plugin that can be used to create
virtual machine images for use with VMware [desktop hypervisors][desktop-hypervisors],
VMware Fusion Pro and VMware Workstation Pro.

### Installation

To install this plugin, add following to your Packer configuration and run
[`packer init`](/packer/docs/commands/init).

```hcl
packer {
  required_plugins {
    vmware = {
      version = "~> 1"
      source = "github.com/vmware/vmware"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this
plugin.

```sh
packer plugins install github.com/vmware/vmware
```

### Components

The plugin includes two builders which are able to create images, depending on your desired
strategy.

#### Builders

- `vmware-iso` - This builder creates a virtual machine, installs a guest operating
  system from an ISO, provisions software within the guest operating system, and then
  exports the virtual machine as an image. Use this builder to start by creating a new
  image.

- `vmware-vmx` - This builder imports an existing virtual machine, runs provisioners on
  the virtual machine, and then exports the virtual machine as an image. Use this
  builder to start from an existing image as the source.

[desktop-hypervisors]: https://www.vmware.com/products/desktop-hypervisor.html
