# Examples

This directory contains a set of templates to illustrate how to run the plugin to build a machine image on thw VMware
desktop hypervisors, VMware Fusion Pro and VMware Workstation Pro.

## Directory Structure

The source files are spread across multiple files:

 - `source.pkr.hcl` contains the source block definition for the machine image.
 - `variables.pkr.hcl` contains a set of defined variables needed for building the machine image.
 - `build.pkr.hcl` is the main entry point to build the machine image defined in `source.pkr.hcl`.
 - `pkrvars/` contains a set of variable definition files (`*.pkrvars.hcl`) partitioned by `guest_os/product-version`.

## Running the Examples

### VMware Fusion Pro

**Apple Silicon-based Macs (`arm64`/`aarch64`)**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-arm64.pkrvars.hcl .
```

  > **Note**
  >
  > VMware Fusion Pro on Apple Silicon does not support the `lsilogic` adapter type and requires additional
  > `vmx_data ` configurations to support the build.
  >
  > Below are the adapter types and addition configuration options required for VMware Fusion Pro 13 on Apple Silicon:
  >
  > ```hcl
  > cdrom_adapter_type   = "sata"
  > disk_adapter_type    = "nvme"
  >
  > Please refer to the example `fusion-arm64.pkrvars.hcl` for complete details.

**Intel-based Macs (`amd64`/`x86_64`)**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-amd64.pkrvars.hcl .
```

## VMware Workstation Pro Examples

```shell
packer init .
packer build -var-file=pkrvars/debian/workstation-amd64.pkrvars.hcl .
```
