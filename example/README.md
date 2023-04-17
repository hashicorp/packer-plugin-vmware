# VMware Packer Plugin Examples

This directory contains a set of templates to illustrate how to run the plugin for different versions of VMware products, 
as VMware ESXi, VMware Fusion, VMware Player, and VMware Workstation all work slightly differently. 

_Big thanks to @tenthirtyam and @Stromweld for their help in getting these templates working._

## Example Directory Structure
The source files are spread across multiple files: 
 - source.pkr.hcl contains the source block definition for the build image. 
 - variables.pkr.hcl contains a set of defined variables needed for building the image. 
 - build.pkr.hcl is the main entry point for Packer to build the VMware image defined in source.pkr.hcl.
 - pkrvars/ contains a set of Packer variable definition files (*.pkrvars.hcl) partitioned by guest_os/vmware-version.

## Running VMware Fusion Examples
  > **Note**
  >
  > VMware Fusion 13 does not support lsilogic adapter types and require additional vmx_data configurations for supporting ARM based builds. 
  > Below are the adapter types and addition vmx_data configuration options that have been found to work. 
  > ```
  > vmx_data = {
  > "svga.autodetect"         = true
  > "usb_xhci.present"        = true
  > }
  > cdrom_adapter_type   = "sata"
  > disk_adapter_type    = "sata"
  > ```
  > Please refer to the [Fusion 13 pkvars.hcl file](pkrvars/debian/fusion-13.pkrvars.hcl) for exact details. 

**For Fusion 12 builds**
```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-12.pkrvars.hcl .
```

**For Fusion 13 builds**
```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-13.pkrvars.hcl .
```

