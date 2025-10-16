# Examples: `vmware-vmx`

Build virtual machines by cloning and customizing an existing virtual machine.

[← Back to all examples](../)

## Running These Examples

### Cloning from `.vmx`

**VMware Fusion Pro - Apple Silicon (`arm64`):**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-arm64-vmx.pkrvars.hcl .
```

**VMware Fusion Pro - Intel (`amd64`):**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-amd64-vmx.pkrvars.hcl .
```

**VMware Workstation Pro:**

```shell
packer init .
packer build -var-file=pkrvars/debian/workstation-amd64-vmx.pkrvars.hcl .
```

### Cloning from `.ova` or `.ovf`

**VMware Fusion Pro - Apple Silicon (`arm64`):**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-arm64-ova.pkrvars.hcl .
# or
packer build -var-file=pkrvars/debian/fusion-arm64-ovf.pkrvars.hcl .
```

**VMware Fusion Pro - Intel (`amd64`):**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-amd64-ova.pkrvars.hcl .
# or
packer build -var-file=pkrvars/debian/fusion-amd64-ovf.pkrvars.hcl .
```

**VMware Workstation Pro:**

```shell
packer init .
packer build -var-file=pkrvars/debian/workstation-amd64-ova.pkrvars.hcl .
# or
packer build -var-file=pkrvars/debian/workstation-amd64-ovf.pkrvars.hcl .
```

## Notes

### `.vmx` Source
- `guest_os_type` is optional (preserves source guest OS if not specified).
- `version` is ignored (hardware version from source is preserved).

### `.ova`/`ovf` Source
- `guest_os_type` is **required** when cloning from OVF/OVA files.
- `version` controls hardware version (default: 21, minimum: 19)

See the root [example README](../) for more information on platform requirements.
