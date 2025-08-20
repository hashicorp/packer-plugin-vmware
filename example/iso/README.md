# Examples: `vmware-iso`

Build virtual machines from scratch using ISO installation media.

[← Back to all examples](../)

## Running These Examples

### VMware Fusion Pro

**Apple Silicon (`arm64`):**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-arm64.pkrvars.hcl .
```

**Intel (`amd64`):**

```shell
packer init .
packer build -var-file=pkrvars/debian/fusion-amd64.pkrvars.hcl .
```

### VMware Workstation Pro

```shell
packer init .
packer build -var-file=pkrvars/debian/workstation-amd64.pkrvars.hcl .
```

## Platform Notes

### VMware Fusion Pro on Apple Silicon

Apple Silicon-based Macs have specific adapter requirements:

- **CD-ROM Adapter:** Must use `sata`.
- **Disk Adapter:** Must use `sata` or `nvme`.
- **Network Adapter:** Use `e1000e`.
- VMware Fusion Pro on Apple Silicon does not support the `lsilogic` adapter type.

### VMware Fusion Pro on Intel

Intel-based Macs support standard adapter types:

- **CD-ROM Adapter:** `ide` or `sata`.
- **Disk Adapter:** `lsilogic`, `scsi`, or `sata`.
- **Network Adapter:** `e1000` or `e1000e`.

### VMware Workstation Pro

Windows and Linux hosts with VMware Workstation Pro support all standard adapter types.
