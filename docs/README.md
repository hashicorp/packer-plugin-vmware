# VMware Plugin

## Installation

### Using pre-built releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration .
Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    vmware = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/vmware"
    }
  }
}
```

#### Manual installation

You can find pre-built binary releases of the plugin [here](https://github.com/hashicorp/packer-plugin-name/releases).
Once you have downloaded the latest archive corresponding to your target OS,
uncompress it to retrieve the plugin binary file corresponding to your platform.
To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


#### From Source

If you prefer to build the plugin from its source code, clone the GitHub
repository locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-vmware` plugin
binary file can be found in the root directory.
To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).


## Plugin Components

The VMware Packer Plugin is able to create VMware virtual machines for use
with any VMware product.

The plugin comes with multiple builders able to create VMware machines,
depending on the strategy you want to use to build the image. The supported VMware builders are:

- [vmware-iso](/docs/builders/vmware-iso) - Starts from an ISO file,
  creates a brand new VMware VM, installs an OS, provisions software within
  the OS, then exports that machine to create an image. This is best for
  people who want to start from scratch.

- [vmware-vmx](/docs/builders/vmware-vmx) - This builder imports an
  existing VMware machine (from a VMX file), runs provisioners on top of that
  VM, and exports that machine to create an image. This is best if you have
  an existing VMware VM you want to use as the source. As an additional
  benefit, you can feed the artifact of this builder back into Packer to
  iterate on a machine.