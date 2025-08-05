<!-- markdownlint-disable first-line-h1 no-inline-html -->

The Packer Plugin for VMware is a plugin that can be used to create virtual machine images for use
with VMware [desktop hypervisors][desktop-hypervisors] (VMware Fusion Pro, VMware Workstation Pro,
and VMware Workstation Player [^1]) and [VMware vSphere Hypervisor][vsphere-hypervisor] [^2].

### Installation

To install this plugin, add following to your Packer configuration and run
[`packer init`](/packer/docs/commands/init).

```hcl
packer {
  required_plugins {
    vmware = {
      version = "~> 1"
      source = "github.com/hashicorp/vmware"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
packer plugins install github.com/hashicorp/vmware
```

### Components

The plugin includes two builders which are able to create images, depending on your desired
strategy.

#### Builders

- `vmware-iso` - This builder creates a virtual machine, installs an operating system from an ISO,
  provisions software within the operating system, and then exports the virtual machine as an image.
  This is best for those who want to start by creating an image.

- `vmware-vmx` - This builder imports an existing virtual machine (from a`.vmx` file), runs
  provisioners on the virtual machine, and then exports the virtual machine as an image. This is
  best for those who want to start from an existing virtual machine as the source. You can feed the
  artifact of this builder back into Packer to iterate on an image.

[^1]:
    Support for VMware Workstation Player is deprecated in v1 and will be removed in the next major release.
    Read more about [discontinuation of VMware Workstation Player][footnote-player-discontinuation].

[^2]:
    Support for VMware vSphere Hypervisor (ESXi) is deprecated in v1 and will be removed in the next major release.
    Please transition to using the [Packer Plugin for VMware vSphere][footnote-packer-plugin-vsphere].

[vsphere-hypervisor]: https://www.vmware.com/products/vsphere-hypervisor.html
[desktop-hypervisors]: https://www.vmware.com/products/desktop-hypervisor.html
[known-issues-so]: https://stackoverflow.com/questions/31985348/vix-vmrun-doesnt-work-with-vmware-player
[footnote-player-discontinuation]: https://blogs.vmware.com/workstation/2024/05/vmware-workstation-pro-now-available-free-for-personal-use.html
[footnote-packer-plugin-vsphere]: https://developer.hashicorp.com/packer/integrations/hashicorp/vsphere
