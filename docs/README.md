# Plugin Components

The Packer Plugin for VMware is a multi-component plugin can be used with
Packer to create virtual machine images for use with VMware products.

The plugin includes two builders which are able to create images, depending on
your desired strategy:

## Builders

- [vmware-iso](/docs/builders/iso.mdx) - Starts from an ISO file,
  creates a brand new VMware VM, installs an OS, provisions software within
  the OS, then exports that machine to create an image. This is best for
  people who want to start from scratch.

- [vmware-vmx](/docs/builders/vmx.mdx) - This builder imports an
  existing VMware machine (from a VMX file), runs provisioners on top of that
  VM, and exports that machine to create an image. This is best if you have
  an existing VMware VM you want to use as the source. As an additional
  benefit, you can feed the artifact of this builder back into Packer to
  iterate on a machine.
