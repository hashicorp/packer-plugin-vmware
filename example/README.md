# Examples

This directory contains a set of templates to illustrate how to run the plugin to build a machine image on thw VMware
desktop hypervisors, VMware Fusion Pro and VMware Workstation.

## Directory Structure

The source files are spread across multiple files: 

 - `source.pkr.hcl` contains the source block definition for the build image. 
 - `variables.pkr.hcl` contains a set of defined variables needed for building the image. 
 - `build.pkr.hcl` is the main entry point for Packer to build the VMware image defined in `source.pkr.hcl`.
 - `pkrvars/` contains a set of Packer variable definition files (`*.pkrvars.hcl`) partitioned by `guest_os/product-version`.

## Running the Examples

### VMware Fusion Pro

**VApple Silicon-based Macs (`aarch64`)**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-13-aarch64.pkrvars.hcl .
```

  > **Note**
  >
  > VMware Fusion Pro 13 on Apple Silicon does not support the `lsilogic` adapter type and requirea additional
  > `vmx_data ` configurations to support `aarch64` based builds. 
  > 
  > Below are the adapter types and addition configuration options required for VMware Fusion Pro 13 on Apple Silicon:
  > 
  > ```hcl
  > cdrom_adapter_type   = "sata"
  > disk_adapter_type    = "sata"
  > 
  > vmx_data = {
  >   "svga.autodetect"  = true
  >   "usb_xhci.present" = true
  > }
  > 
  > Please refer to the exanple `fusion-13-aarch64.pkrvars.hcl` for complete details. 


**Intel-based Macs (`x86_64`)**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-13-x86_64.pkrvars.hcl .
```

## VMware Workstation Examples

**Workstation 17 (`x86_64`)**

```shell
packer init .
packer build -var-file=pkrvars/debian/workstation-17-x86_64.pkrvars.hcl .
```