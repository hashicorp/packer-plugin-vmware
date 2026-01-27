Type: `vmware-iso`

This builder creates a virtual machine, installs an operating system from an ISO, provisions
software within the operating system, and then exports the virtual machine as an image. This is best
for those who want to start by creating an image for use with VMware [desktop hypervisors][desktop-hypervisors]
(VMware Fusion Pro and VMware Workstation Pro).

| Hypervisor Type     | Artifact BuilderId     |
|---------------------|------------------------|
| Desktop Hypervisor  | `vmware.desktop`       |

## Basic Example

This example builds a virtual machine from a guest operating system ISO file. The builder will start the
virtual machine, boot from the ISO, and install the operating system, and run any provisioners, and then
export the virtual machine as an image.

~> **Note**: This is not a complete working example.

HCL Example:

```hcl
source "vmware-iso" "example" {
  iso_url = "https://releases.ubuntu.com/yy.mm/ubuntu-yy.mm-live-server-amd64.iso"
  iso_checksum = "sha256:8762f7e74e4d64d72fceb5f70682e6b069932deedb4949c6975d0f0fe0a91be3"
  ssh_username = "packer"
  ssh_password = "password"
  shutdown_command = "shutdown -P now"
}

build {
  sources = [
    "source.vmware-iso.example"
  ]
}
```

JSON Example:

```json
{
  "type": "vmware-iso",
  "iso_url": "https://releases.ubuntu.com/yy.mm/ubuntu-yy.mm-live-server-amd64.iso",
  "iso_checksum": "sha256:8762f7e74e4d64d72fceb5f70682e6b069932deedb4949c6975d0f0fe0a91be3",
  "ssh_username": "packer",
  "ssh_password": "password",
  "shutdown_command": "shutdown -P now"
}
```

## Configuration Reference

**Optional**:

<!-- Code generated from the comments of the Config struct in builder/vmware/iso/config.go; DO NOT EDIT MANUALLY -->

- `disk_size` (uint) - The size of the disk in megabytes. The builder uses expandable virtual
  hard disks. The file that backs the virtual disk will only grow as needed
  up to this size. Default is 40000 (~40 GB).

- `cdrom_adapter_type` (string) - The type of controller to use for the CD-ROM device.
  Allowed values are `ide`, `sata`, and `scsi`.

- `guest_os_type` (string) - The guest operating system identifier for the virtual machine.
  Defaults to `other-64` on amd64 and `arm-other-64` on arm64.

- `version` (int) - The virtual machine hardware version. Refer to [KB 315655](https://knowledge.broadcom.com/external/article?articleNumber=315655)
  for more information on supported virtual hardware versions.
  Default is 21. Minimum is 19.

- `vm_name` (string) - The name of the virtual machine. This represents the name of the virtual
  machine `.vmx` configuration file without the file extension.
  Default is `packer-<BUILDNAME>`, where `<BUILDNAME>` is the name of the
  build.

- `vmx_disk_template_path` (string) - The path to a [configuration template](/packer/docs/templates/legacy_json_templates/engine)
  for defining the contents of a virtual machine `.vmx` configuration file
  for a virtual disk. Template variables `{{ .DiskType }}`, `{{ .DiskUnit }}`,
  `{{ .DiskName }}`, and `{{ .DiskNumber }}` are available for use within
  the template.
  
  ~> **Note:** This option is intended for advanced users, as incorrect
  configurations can lead to non-functional virtual machines.

- `vmx_template_path` (string) - The path to a [configuration template](/packer/docs/templates/legacy_json_templates/engine)
  for defining the contents of a virtual machine `.vmx` configuration file.
  
  ~> **Note:** This option is intended for advanced users, as incorrect
  configurations can lead to non-functional virtual machines. For simpler
  modifications of the virtual machine`.vmx` configuration file, consider
  using `vmx_data` option.

- `snapshot_name` (string) - The name of the virtual machine snapshot to be created.
  If this field is left empty, no snapshot will be created.

- `vhv_enabled` (bool) - Enable virtual hardware-assisted virtualization for the virtual machine.
  Defaults to `false`.

<!-- End of code generated from the comments of the Config struct in builder/vmware/iso/config.go; -->


### Hardware Configuration

**Optional**:

<!-- Code generated from the comments of the HWConfig struct in builder/vmware/common/hw_config.go; DO NOT EDIT MANUALLY -->

- `firmware` (string) - The firmware type for the virtual machine.
  Allowed values are `bios`, `efi`, and `efi-secure` (for secure boot).
  Defaults to the recommended firmware type for the guest operating system.

- `cpus` (int) - The number of virtual CPUs cores for the virtual machine.

- `cores` (int) - The number of virtual CPU cores per socket for the virtual machine.

- `memory` (int) - The amount of memory for the virtual machine in MB. Defaults to `512`.

- `network` (string) - The network which the virtual machine will connect for desktop
  hypervisors. Use the generic values that map to a device, such as
  `hostonly`, `nat`, or `bridged`. Defaults to `nat`.
  
  ~> **Note:** If not set to one of these generic values, then it is
  assumed to be a network device (_e.g._, `VMnet0..x`).

- `network_name` (string) - The network which the virtual machine will connect on a remote
  hypervisor.

- `network_adapter_type` (string) - The network adapter type for the virtual machine.
  Allowed values are `vmxnet3`, `e1000e`, and `e1000`
  
  Refer to the VMware desktop hypervisor product documentation for
  the network adapter types supported by the guest operating system
  and the CPU architecture (`amd64/x86_64` vs `arm64/aarch64`).

- `sound` (bool) - Enable virtual sound card device. Defaults to `false`.

- `usb` (bool) - Enable USB controller for the virtual machine.
  Defaults to `false`.
  
  ~> **Note:** Automatically enabled on Apple Silicon-based systems to
  ensure plugin functionality.

- `usb_version` (string) - USB version to use when USB is enabled. Defaults to `3.1`.
  Allowed values are `2.0`, `3.1`, and `3.2`.
  
  ~> **Note:** Both `3.2` and `3.1` produce an identical configuration
  for the virtual machine. VMware Fusion and Workstation 25H2 and
  later use `3.2`, whereas previous versions use `3.1`.
  
  ~> **Note:** Automatically set on Apple Silicon-based systems to ensure
  plugin functionality.

- `serial` (string) - Add a serial port to the virtual machine. Use a format of
  `Type:option1,option2,...`. Allowed values for the field `Type` include:
  `FILE`, `DEVICE`, `PIPE`, `AUTO`, or `NONE`.
  
  * `FILE:path(,yield)` - Specifies the path to the local file to be used
    as the serial port.
  
    * `yield` (bool) - This is an optional boolean that specifies
      whether the virtual machine should yield the CPU when polling the
      port. By default, the builder will assume this as `FALSE`.
  
  * `DEVICE:path(,yield)` - Specifies the path to the local device to be
       used as the serial port. If `path` is empty, then default to the first
     serial port.
  
    * `yield` (bool) - This is an optional boolean that specifies
      whether the virtual machine should yield the CPU when polling the
      port. By default, the builder will assume this as `FALSE`.
  
  * `PIPE:path,endpoint,host(,yield)` - Specifies to use the named-pipe
     "path" as a serial port. This has a few options that determine how the
     VM should use the named-pipe.
  
    * `endpoint` (string) - Chooses the type of the VM-end, which can be
      either a `client` or `server`.
  
    * `host` (string) - Chooses the type of the host-end, which can be
      either `app` (application) or `vm` (another virtual-machine).
  
    * `yield` (bool) - This is an optional boolean that specifies whether
      the virtual machine should yield the CPU when polling the port. By
      default, the builder will assume this as `FALSE`.
  
  * `AUTO: (yield)` - Specifies to use auto-detection to determine the
     serial port to use. This has one option to determine how the virtual
     machine should support the serial port.
  
    * `yield` (bool) - This is an optional boolean that specifies whether
      the virtual machine should yield the CPU when polling the port. By
      default, the builder will assume this as `FALSE`.
  
  * `NONE` - Specifies to not use a serial port. (default)

- `parallel` (string) - Add a parallel port to add to the virtual machine. Use a format of
  `Type:option1,option2,...`. Allowed values for the field `Type` include:
  `FILE`, `DEVICE`, `AUTO`, or `NONE`.
  
  * `FILE:path` - Specifies the path to the local file to be used for the
     parallel port.
  
  * `DEVICE:path` - Specifies the path to the local device to be used for
     the parallel port.
  
  * `AUTO:direction` - Specifies to use auto-detection to determine the
     parallel port. Direction can be `BI` to specify bidirectional
     communication or `UNI` to specify unidirectional communication.
  
  * `NONE` - Specifies to not use a parallel port. (default)

<!-- End of code generated from the comments of the HWConfig struct in builder/vmware/common/hw_config.go; -->


### Extra Disk Configuration

**Optional**:

<!-- Code generated from the comments of the DiskConfig struct in builder/vmware/common/disk_config.go; DO NOT EDIT MANUALLY -->

- `disk_additional_size` ([]uint) - The size(s) of additional virtual hard disks in MB. If not specified,
  the virtual machine will contain only a primary hard disk.

- `disk_adapter_type` (string) - The adapter type for additional virtual disk(s). Available options
   are `ide`, `sata`, `nvme`, or `scsi`.
  
  ~> **Note:** When specifying `scsi` as the adapter type, the default
  adapter type is set to `lsilogic`. If another option is specified, the
  plugin will assume it is a `scsi` interface of that specified type.
  
  ~> **Note:** This option is intended for advanced users.

- `vmdk_name` (string) - The filename for the virtual disk to create _without_ the `.vmdk`
  extension. Defaults to `disk`.

- `disk_type_id` (string) - The type of virtual disk to create. The available options include:
  
    | Type ID | Description                                                             |
    |---------|-------------------------------------------------------------------------|
    | `0`     | Growable virtual disk contained in a single file (monolithic sparse).   |
    | `1`     | Growable virtual disk split into 2GB files (split sparse).              |
    | `2`     | Preallocated virtual disk contained in a single file (monolithic flat). |
    | `3`     | Preallocated virtual disk split into 2GB files (split flat).            |
    | `4`     | Preallocated virtual disk compatible with ESXi (VMFS flat).             |
    | `5`     | Compressed disk optimized for streaming.                                |
  
    Defaults to `1`
  
    ~> **Note:** Set `skip_compaction` to `true` when using `zeroedthick`
    or `eagerzeroedthick` due to default disk compaction behavior.
  
  ~> **Note:** This option is intended for advanced users.

<!-- End of code generated from the comments of the DiskConfig struct in builder/vmware/common/disk_config.go; -->


**Optional**:

<!-- Code generated from the comments of the ISOConfig struct in multistep/commonsteps/iso_config.go; DO NOT EDIT MANUALLY -->

- `iso_urls` ([]string) - Multiple URLs for the ISO to download. Packer will try these in order.
  If anything goes wrong attempting to download or while downloading a
  single URL, it will move on to the next. All URLs must point to the same
  file (same checksum). By default this is empty and `iso_url` is used.
  Only one of `iso_url` or `iso_urls` can be specified.

- `iso_target_path` (string) - The path where the iso should be saved after download. By default will
  go in the packer cache, with a hash of the original filename and
  checksum as its name.

- `iso_target_extension` (string) - The extension of the iso file after download. This defaults to `iso`.

<!-- End of code generated from the comments of the ISOConfig struct in multistep/commonsteps/iso_config.go; -->


### VMware Tools Configuration

**Optional**:

<!-- Code generated from the comments of the ToolsConfig struct in builder/vmware/common/tools_config.go; DO NOT EDIT MANUALLY -->

- `tools_mode` (string) - The mode for providing VMware Tools to the virtual machine. Must be
  explicitly specified when using any tools configuration. Allowed values are:
  - `upload`: Uploads VMware Tools ISO to the virtual machine during the build.
    Requires either `tools_upload_flavor` or `tools_source_path` to be specified.
  - `attach`: Attaches the VMware Tools ISO to the virtual machine as a CD-ROM
    device during the build and removes the device upon build completion.
    Requires `tools_source_path` to be specified.
  - `disable`: No VMware Tools ISO is provided to the virtual machine.
    Any other tools configuration fields are ignored.

- `tools_source_path` (string) - The absolute local path on your machine to the VMware Tools ISO file.
  Can be used with `tools_mode` set to `attach` or `upload`. When used with
  `upload` mode, cannot be used together with `tools_upload_flavor`.
  
  Must be a path accessible during the build (e.g., "/path/to/vmware-tools.iso".)

- `tools_upload_flavor` (string) - The flavor of VMware Tools to upload into the virtual machine based on the
  guest operating system. Can only be used when `tools_mode` is set to
  `upload`. Cannot be used together with `tools_source_path`. Allowed
  values include: `darwin` (macOS), `linux`, and `windows`.
  
  The plugin will load the VMware Tools ISO from the desktop hypervisor's
  default installation directory based on the specified flavor, if available.

- `tools_upload_path` (string) - The absolute path in the virtual machine guest operating system where the
  VMware Tools ISO will be uploaded. Only used when `tools_mode` is set to
  `upload`. This is a [configuration template](/packer/docs/templates/legacy_json_templates/engine)
  that has a single valid variable: `Flavor`, which will be the value of
  `tools_upload_flavor`. Defaults to `{{.Flavor}}.iso` when
  `tools_upload_flavor` is specified.
  
  Must be an absolute path in the guest operating system (e.g., "/tmp/vmware-tools.iso").

<!-- End of code generated from the comments of the ToolsConfig struct in builder/vmware/common/tools_config.go; -->


### Floppy Configuration

<!-- Code generated from the comments of the FloppyConfig struct in multistep/commonsteps/floppy_config.go; DO NOT EDIT MANUALLY -->

A floppy can be made available for your build. This is most useful for
unattended Windows installs, which look for an Autounattend.xml file on
removable media. By default, no floppy will be attached. All files listed in
this setting get placed into the root directory of the floppy and the floppy
is attached as the first floppy device. The summary size of the listed files
must not exceed 1.44 MB. The supported ways to move large files into the OS
are using `http_directory` or [the file
provisioner](/packer/docs/provisioner/file).

<!-- End of code generated from the comments of the FloppyConfig struct in multistep/commonsteps/floppy_config.go; -->


**Optional**:

<!-- Code generated from the comments of the FloppyConfig struct in multistep/commonsteps/floppy_config.go; DO NOT EDIT MANUALLY -->

- `floppy_files` ([]string) - A list of files to place onto a floppy disk that is attached when the VM
  is booted. Currently, no support exists for creating sub-directories on
  the floppy. Wildcard characters (\\*, ?, and \[\]) are allowed. Directory
  names are also allowed, which will add all the files found in the
  directory to the floppy.

- `floppy_dirs` ([]string) - A list of directories to place onto the floppy disk recursively. This is
  similar to the `floppy_files` option except that the directory structure
  is preserved. This is useful for when your floppy disk includes drivers
  or if you just want to organize it's contents as a hierarchy. Wildcard
  characters (\\*, ?, and \[\]) are allowed. The maximum summary size of
  all files in the listed directories are the same as in `floppy_files`.

- `floppy_content` (map[string]string) - Key/Values to add to the floppy disk. The keys represent the paths, and
  the values contents. It can be used alongside `floppy_files` or
  `floppy_dirs`, which is useful to add large files without loading them
  into memory. If any paths are specified by both, the contents in
  `floppy_content` will take precedence.
  
  Usage example (HCL):
  
  ```hcl
  floppy_files = ["vendor-data"]
  floppy_content = {
    "meta-data" = jsonencode(local.instance_data)
    "user-data" = templatefile("user-data", { packages = ["nginx"] })
  }
  floppy_label = "cidata"
  ```

- `floppy_label` (string) - Floppy Label

<!-- End of code generated from the comments of the FloppyConfig struct in multistep/commonsteps/floppy_config.go; -->


### CD-ROM Configuration

<!-- Code generated from the comments of the CDConfig struct in multistep/commonsteps/extra_iso_config.go; DO NOT EDIT MANUALLY -->

- `cd_files` ([]string) - A list of files to place onto a CD that is attached when the VM is
  booted. This can include either files or directories; any directories
  will be copied onto the CD recursively, preserving directory structure
  hierarchy. Symlinks will have the link's target copied into the directory
  tree on the CD where the symlink was. File globbing is allowed.
  
  Usage example (JSON):
  
  ```json
  "cd_files": ["./somedirectory/meta-data", "./somedirectory/user-data"],
  "cd_label": "cidata",
  ```
  
  Usage example (HCL):
  
  ```hcl
  cd_files = ["./somedirectory/meta-data", "./somedirectory/user-data"]
  cd_label = "cidata"
  ```
  
  The above will create a CD with two files, user-data and meta-data in the
  CD root. This specific example is how you would create a CD that can be
  used for an Ubuntu 20.04 autoinstall.
  
  Since globbing is also supported,
  
  ```hcl
  cd_files = ["./somedirectory/*"]
  cd_label = "cidata"
  ```
  
  Would also be an acceptable way to define the above cd. The difference
  between providing the directory with or without the glob is whether the
  directory itself or its contents will be at the CD root.
  
  Use of this option assumes that you have a command line tool installed
  that can handle the iso creation. Packer will use one of the following
  tools:
  
    * xorriso
    * mkisofs
    * hdiutil (normally found in macOS)
    * oscdimg (normally found in Windows as part of the Windows ADK)

- `cd_content` (map[string]string) - Key/Values to add to the CD. The keys represent the paths, and the values
  contents. It can be used alongside `cd_files`, which is useful to add large
  files without loading them into memory. If any paths are specified by both,
  the contents in `cd_content` will take precedence.
  
  Usage example (HCL):
  
  ```hcl
  cd_files = ["vendor-data"]
  cd_content = {
    "meta-data" = jsonencode(local.instance_data)
    "user-data" = templatefile("user-data", { packages = ["nginx"] })
  }
  cd_label = "cidata"
  ```

- `cd_label` (string) - CD Label

<!-- End of code generated from the comments of the CDConfig struct in multistep/commonsteps/extra_iso_config.go; -->


### ISO Configuration

<!-- Code generated from the comments of the ISOConfig struct in multistep/commonsteps/iso_config.go; DO NOT EDIT MANUALLY -->

By default, Packer will symlink, download or copy image files to the Packer
cache into a "`hash($iso_url+$iso_checksum).$iso_target_extension`" file.
Packer uses [hashicorp/go-getter](https://github.com/hashicorp/go-getter) in
file mode in order to perform a download.

go-getter supports the following protocols:

* Local files
* Git
* Mercurial
* HTTP
* Amazon S3

Examples:
go-getter can guess the checksum type based on `iso_checksum` length, and it is
also possible to specify the checksum type.

In JSON:

```json

	"iso_checksum": "946a6077af6f5f95a51f82fdc44051c7aa19f9cfc5f737954845a6050543d7c2",
	"iso_url": "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

```json

	"iso_checksum": "file:ubuntu.org/..../ubuntu-14.04.1-server-amd64.iso.sum",
	"iso_url": "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

```json

	"iso_checksum": "file://./shasums.txt",
	"iso_url": "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

```json

	"iso_checksum": "file:./shasums.txt",
	"iso_url": "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

In HCL2:

```hcl

	iso_checksum = "946a6077af6f5f95a51f82fdc44051c7aa19f9cfc5f737954845a6050543d7c2"
	iso_url = "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

```hcl

	iso_checksum = "file:ubuntu.org/..../ubuntu-14.04.1-server-amd64.iso.sum"
	iso_url = "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

```hcl

	iso_checksum = "file://./shasums.txt"
	iso_url = "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

```hcl

	iso_checksum = "file:./shasums.txt",
	iso_url = "ubuntu.org/.../ubuntu-14.04.1-server-amd64.iso"

```

<!-- End of code generated from the comments of the ISOConfig struct in multistep/commonsteps/iso_config.go; -->


**Required**:

<!-- Code generated from the comments of the ISOConfig struct in multistep/commonsteps/iso_config.go; DO NOT EDIT MANUALLY -->

- `iso_checksum` (string) - The checksum for the ISO file or virtual hard drive file. The type of
  the checksum is specified within the checksum field as a prefix, ex:
  "md5:{$checksum}". The type of the checksum can also be omitted and
  Packer will try to infer it based on string length. Valid values are
  "none", "{$checksum}", "md5:{$checksum}", "sha1:{$checksum}",
  "sha256:{$checksum}", "sha512:{$checksum}" or "file:{$path}". Here is a
  list of valid checksum values:
   * md5:090992ba9fd140077b0661cb75f7ce13
   * 090992ba9fd140077b0661cb75f7ce13
   * sha1:ebfb681885ddf1234c18094a45bbeafd91467911
   * ebfb681885ddf1234c18094a45bbeafd91467911
   * sha256:ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
   * ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
   * file:http://releases.ubuntu.com/20.04/SHA256SUMS
   * file:file://./local/path/file.sum
   * file:./local/path/file.sum
   * none
  Although the checksum will not be verified when it is set to "none",
  this is not recommended since these files can be very large and
  corruption does happen from time to time.

- `iso_url` (string) - A URL to the ISO containing the installation image or virtual hard drive
  (VHD or VHDX) file to clone.

<!-- End of code generated from the comments of the ISOConfig struct in multistep/commonsteps/iso_config.go; -->


### HTTP Configuration

<!-- Code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; DO NOT EDIT MANUALLY -->

Packer will create an http server serving `http_directory` when it is set, a
random free port will be selected and the architecture of the directory
referenced will be available in your builder.

Example usage from a builder:

```
wget http://{{ .HTTPIP }}:{{ .HTTPPort }}/foo/bar/preseed.cfg
```

<!-- End of code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; -->


**Optional**:

<!-- Code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; DO NOT EDIT MANUALLY -->

- `http_directory` (string) - Path to a directory to serve using an HTTP server. The files in this
  directory will be available over HTTP that will be requestable from the
  virtual machine. This is useful for hosting kickstart files and so on.
  By default this is an empty string, which means no HTTP server will be
  started. The address and port of the HTTP server will be available as
  variables in `boot_command`. This is covered in more detail below.

- `http_content` (map[string]string) - Key/Values to serve using an HTTP server. `http_content` works like and
  conflicts with `http_directory`. The keys represent the paths and the
  values contents, the keys must start with a slash, ex: `/path/to/file`.
  `http_content` is useful for hosting kickstart files and so on. By
  default this is empty, which means no HTTP server will be started. The
  address and port of the HTTP server will be available as variables in
  `boot_command`. This is covered in more detail below.
  Example:
  ```hcl
    http_content = {
      "/a/b"     = file("http/b")
      "/foo/bar" = templatefile("${path.root}/preseed.cfg", { packages = ["nginx"] })
    }
  ```

- `http_port_min` (int) - These are the minimum and maximum port to use for the HTTP server
  started to serve the `http_directory`. Because Packer often runs in
  parallel, Packer will choose a randomly available port in this range to
  run the HTTP server. If you want to force the HTTP server to be on one
  port, make this minimum and maximum port the same. By default the values
  are `8000` and `9000`, respectively.

- `http_port_max` (int) - HTTP Port Max

- `http_bind_address` (string) - This is the bind address for the HTTP server. Defaults to 0.0.0.0 so that
  it will work with any network interface.

- `http_network_protocol` (string) - Defines the HTTP Network protocol. Valid options are `tcp`, `tcp4`, `tcp6`,
  `unix`, and `unixpacket`. This value defaults to `tcp`.

<!-- End of code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; -->


### Shutdown Configuration

**Optional**:

<!-- Code generated from the comments of the ShutdownConfig struct in shutdowncommand/config.go; DO NOT EDIT MANUALLY -->

- `shutdown_command` (string) - The command to use to gracefully shut down the machine once all
  provisioning is complete. By default this is an empty string, which
  tells Packer to just forcefully shut down the machine. This setting can
  be safely omitted if for example, a shutdown command to gracefully halt
  the machine is configured inside a provisioning script. If one or more
  scripts require a reboot it is suggested to leave this blank (since
  reboots may fail) and instead specify the final shutdown command in your
  last script.

- `shutdown_timeout` (duration string | ex: "1h5m2s") - The amount of time to wait after executing the shutdown_command for the
  virtual machine to actually shut down. If the machine doesn't shut down
  in this time it is considered an error. By default, the time out is "5m"
  (five minutes).

<!-- End of code generated from the comments of the ShutdownConfig struct in shutdowncommand/config.go; -->


### Export Configuration

**Optional**:

<!-- Code generated from the comments of the ExportConfig struct in builder/vmware/common/export_config.go; DO NOT EDIT MANUALLY -->

- `format` (string) - The output format of the exported virtual machine. Allowed values are
  `ova`, `ovf`, or `vmx`. Defaults to `vmx`.
  
  ~> **Note:** Ensure VMware OVF Tool is installed. For the latest version,
  visit [VMware OVF Tool](https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest).
  
  ~> **Note:** The plugin will create a `.vmx` and supporting files in the
  output directory and will then export the virtual machine to the specified
  format. These files are **not** automatically cleaned up after the export process.

- `ovftool_options` ([]string) - Additional command-line arguments to send to VMware OVF Tool during the
  export process. Each string in the array represents a separate
  command-line argument.
  
  ~> **Important:** The plugin does not preset any VMware OVF Tool options
  by default.
  
  ~> **Note:** Ensure VMware OVF Tool is installed. For the latest version,
  visit [VMware OVF Tool](https://developer.broadcom.com/tools/open-virtualization-format-ovf-tool/latest).

- `skip_export` (bool) - Skips the export of the virtual machine. This is useful if the build
  output is not the resultant image, but created inside the virtual
  machine. This is useful for debugging purposes. Defaults to `false`.

- `skip_compaction` (bool) - At the end of the build process, the plugin defragments and compacts the
  disks using `vmware-vdiskmanager`. In some cases, this process may result
  in slightly larger disk sizes. If this occurs, you can opt to skip the
  disk compaction step by using this setting. Defaults to `false`.

<!-- End of code generated from the comments of the ExportConfig struct in builder/vmware/common/export_config.go; -->


### Output Configuration

**Optional**:

<!-- Code generated from the comments of the OutputConfig struct in builder/vmware/common/output_config.go; DO NOT EDIT MANUALLY -->

- `output_directory` (string) - This is the path on your local machine to the directory where the
  resulting virtual machine will be created. This may be relative or
  absolute. If relative, the path is relative to the working directory
  when Packer is run.
  
  By default, this is `output-BUILDNAME` where `BUILDNAME` is the name of
  the build.
  
  ~> **Note:** This directory must not exist or be empty before running the
  build.

<!-- End of code generated from the comments of the OutputConfig struct in builder/vmware/common/output_config.go; -->


### Hypervisor Configuration

**Optional**:

<!-- Code generated from the comments of the DriverConfig struct in builder/vmware/common/driver_config.go; DO NOT EDIT MANUALLY -->

- `fusion_app_path` (string) - The installation path of the VMware Fusion application.
  
  ~> **Note:** This is only required if you are using VMware Fusion as a
  desktop hypervisor and have installed it in a non-default location.

- `remote_type` (string) - No longer supported.
  
  ~> **Important:** VMware ESX is not supported by the plugin as of v2.0.0.
  Please use the [Packer plugin for VMware vSphere](https://developer.hashicorp.com/packer/integrations/hashicorp/vsphere).

<!-- End of code generated from the comments of the DriverConfig struct in builder/vmware/common/driver_config.go; -->


### Advanced Configuration

**Optional**:

<!-- Code generated from the comments of the VMXConfig struct in builder/vmware/common/vmx_config.go; DO NOT EDIT MANUALLY -->

- `vmx_data` (map[string]string) - Key-value pairs that will be inserted into the virtual machine `.vmx`
  file before the virtual machine is started. This is useful for
  setting advanced properties that are not supported by the plugin.
  
  ~> **Note**: This option is intended for advanced users who understand
  the ramifications of making changes to the `.vmx` file. This option is
  not necessary for most users.

- `vmx_data_post` (map[string]string) - Key-value pairs that will be inserted into the virtual machine `.vmx`
  file after the virtual machine build is complete. This is useful for
  setting advanced properties that are not supported by the plugin.
  
  ~> **Note**: This option is intended for advanced users who understand
  the ramifications of making changes to the `.vmx` file. This option is
  not necessary for most users.

- `vmx_remove_ethernet_interfaces` (bool) - Remove all network adapters from virtual machine `.vmx` file after the
  virtual machine build is complete. Defaults to `false`.
  
  ~> **Note**: This option is useful when building Vagrant boxes since
  Vagrant will create interfaces when provisioning a box.

- `display_name` (string) - The inventory display name for the virtual machine. If set, the value
  provided will override any value set in the `vmx_data` option or in the
  `.vmx` file. This option is useful if you are chaining builds and want to
  ensure that the display name of each step in the chain is unique.

<!-- End of code generated from the comments of the VMXConfig struct in builder/vmware/common/vmx_config.go; -->


## Boot Configuration

<!-- Code generated from the comments of the BootConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

The boot configuration is very important: `boot_command` specifies the keys
to type when the virtual machine is first booted in order to start the OS
installer. This command is typed after boot_wait, which gives the virtual
machine some time to actually load.

The boot_command is an array of strings. The strings are all typed in
sequence. It is an array only to improve readability within the template.

There are a set of special keys available. If these are in your boot
command, they will be replaced by the proper key:

-   `<bs>` - Backspace

-   `<del>` - Delete

-   `<enter> <return>` - Simulates an actual "enter" or "return" keypress.

-   `<esc>` - Simulates pressing the escape key.

-   `<tab>` - Simulates pressing the tab key.

-   `<f1> - <f12>` - Simulates pressing a function key.

-   `<up> <down> <left> <right>` - Simulates pressing an arrow key.

-   `<spacebar>` - Simulates pressing the spacebar.

-   `<insert>` - Simulates pressing the insert key.

-   `<home> <end>` - Simulates pressing the home and end keys.

  - `<pageUp> <pageDown>` - Simulates pressing the page up and page down
    keys.

-   `<menu>` - Simulates pressing the Menu key.

-   `<leftAlt> <rightAlt>` - Simulates pressing the alt key.

-   `<leftCtrl> <rightCtrl>` - Simulates pressing the ctrl key.

-   `<leftShift> <rightShift>` - Simulates pressing the shift key.

-   `<leftSuper> <rightSuper>` - Simulates pressing the ⌘ or Windows key.

  - `<wait> <wait5> <wait10>` - Adds a 1, 5 or 10 second pause before
    sending any additional keys. This is useful if you have to generally
    wait for the UI to update before typing more.

  - `<waitXX>` - Add an arbitrary pause before sending any additional keys.
    The format of `XX` is a sequence of positive decimal numbers, each with
    optional fraction and a unit suffix, such as `300ms`, `1.5h` or `2h45m`.
    Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`. For
    example `<wait10m>` or `<wait1m20s>`.

  - `<XXXOn> <XXXOff>` - Any printable keyboard character, and of these
    "special" expressions, with the exception of the `<wait>` types, can
    also be toggled on or off. For example, to simulate ctrl+c, use
    `<leftCtrlOn>c<leftCtrlOff>`. Be sure to release them, otherwise they
    will be held down until the machine reboots. To hold the `c` key down,
    you would use `<cOn>`. Likewise, `<cOff>` to release.

  - `{{ .HTTPIP }} {{ .HTTPPort }}` - The IP and port, respectively of an
    HTTP server that is started serving the directory specified by the
    `http_directory` configuration parameter. If `http_directory` isn't
    specified, these will be blank!

-   `{{ .Name }}` - The name of the VM.

Example boot command. This is actually a working boot command used to start an
CentOS 6.4 installer:

In JSON:

```json
"boot_command": [

	   "<tab><wait>",
	   " ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/centos6-ks.cfg<enter>"
	]

```

In HCL2:

```hcl
boot_command = [

	   "<tab><wait>",
	   " ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/centos6-ks.cfg<enter>"
	]

```

The example shown below is a working boot command used to start an Ubuntu
12.04 installer:

In JSON:

```json
"boot_command": [

	"<esc><esc><enter><wait>",
	"/install/vmlinuz noapic ",
	"preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg ",
	"debian-installer=en_US auto locale=en_US kbd-chooser/method=us ",
	"hostname={{ .Name }} ",
	"fb=false debconf/frontend=noninteractive ",
	"keyboard-configuration/modelcode=SKIP keyboard-configuration/layout=USA ",
	"keyboard-configuration/variant=USA console-setup/ask_detect=false ",
	"initrd=/install/initrd.gz -- <enter>"

]
```

In HCL2:

```hcl
boot_command = [

	"<esc><esc><enter><wait>",
	"/install/vmlinuz noapic ",
	"preseed/url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg ",
	"debian-installer=en_US auto locale=en_US kbd-chooser/method=us ",
	"hostname={{ .Name }} ",
	"fb=false debconf/frontend=noninteractive ",
	"keyboard-configuration/modelcode=SKIP keyboard-configuration/layout=USA ",
	"keyboard-configuration/variant=USA console-setup/ask_detect=false ",
	"initrd=/install/initrd.gz -- <enter>"

]
```

For more examples of various boot commands, see the sample projects from our
[community templates page](https://packer.io/community-tools#templates).

<!-- End of code generated from the comments of the BootConfig struct in bootcommand/config.go; -->


<!-- Code generated from the comments of the VNCConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

The boot command "typed" character for character over a VNC connection to
the machine, simulating a human actually typing the keyboard.

Keystrokes are typed as separate key up/down events over VNC with a default
100ms delay. The delay alleviates issues with latency and CPU contention.
You can tune this delay on a per-builder basis by specifying
"boot_key_interval" in your Packer template.

<!-- End of code generated from the comments of the VNCConfig struct in bootcommand/config.go; -->


-> **Note**: For the `HTTPIP` to be resolved, the `network` interface type must
be set to either `hostonly` or `nat`. It is recommended to leave the default
network configuration while you are building the virtual machine, and use the
`vmx_data_post` hook to modify the network configuration after the virtual
machine build is complete.

**Optional**:

<!-- Code generated from the comments of the BootConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

- `boot_keygroup_interval` (duration string | ex: "1h5m2s") - Time to wait after sending a group of key pressses. The value of this
  should be a duration. Examples are `5s` and `1m30s` which will cause
  Packer to wait five seconds and one minute 30 seconds, respectively. If
  this isn't specified, a sensible default value is picked depending on
  the builder type.

- `boot_wait` (duration string | ex: "1h5m2s") - The time to wait after booting the initial virtual machine before typing
  the `boot_command`. The value of this should be a duration. Examples are
  `5s` and `1m30s` which will cause Packer to wait five seconds and one
  minute 30 seconds, respectively. If this isn't specified, the default is
  `10s` or 10 seconds. To set boot_wait to 0s, use a negative number, such
  as "-1s"

- `boot_command` ([]string) - This is an array of commands to type when the virtual machine is first
  booted. The goal of these commands should be to type just enough to
  initialize the operating system installer. Special keys can be typed as
  well, and are covered in the section below on the boot command. If this
  is not specified, it is assumed the installer will start itself.

<!-- End of code generated from the comments of the BootConfig struct in bootcommand/config.go; -->


<!-- Code generated from the comments of the VNCConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

- `disable_vnc` (bool) - Whether to create a VNC connection or not. A boot_command cannot be used
  when this is true. Defaults to false.

- `boot_key_interval` (duration string | ex: "1h5m2s") - Time in ms to wait between each key press

<!-- End of code generated from the comments of the VNCConfig struct in bootcommand/config.go; -->


<!-- Code generated from the comments of the RunConfig struct in builder/vmware/common/run_config.go; DO NOT EDIT MANUALLY -->

- `headless` (bool) - The plugin defaults to building virtual machines by launching the
  desktop hypervisor's graphical user interface (GUI) to display the
  console of the virtual machine being built. When this value is set to
  `true`, the virtual machine will start without a console; however, the
  plugin will output VNC connection information in case you need to connect
  to the console to debug the build process. Defaults to `false`.

- `vnc_bind_address` (string) - The IP address to use for VNC access to the virtual machine. Defaults to
  `127.0.0.1`.
  
  ~> **Note:** To bind to all interfaces, use `0.0.0.0`.

- `vnc_port_min` (int) - The minimum port number to use for VNC access to the virtual machine.
  The plugin uses VNC to type the `boot_command`. Defaults to `5900`.

- `vnc_port_max` (int) - The maximum port number to use for VNC access to the virtual machine.
  The plugin uses VNC to type the `boot_command`. Defaults to `6000`.
  
  ~> **Note:** The plugin randomly selects port within the inclusive range
  specified by `vnc_port_min` and `vnc_port_max`.

- `vnc_disable_password` (bool) - Disables the auto-generation of a VNC password that is used to secure the
  VNC communication with the virtual machine. Defaults to `false`.

<!-- End of code generated from the comments of the RunConfig struct in builder/vmware/common/run_config.go; -->


### Communicator Configuration

**Optional**:

##### Common

<!-- Code generated from the comments of the Config struct in communicator/config.go; DO NOT EDIT MANUALLY -->

- `communicator` (string) - Packer currently supports three kinds of communicators:
  
  -   `none` - No communicator will be used. If this is set, most
      provisioners also can't be used.
  
  -   `ssh` - An SSH connection will be established to the machine. This
      is usually the default.
  
  -   `winrm` - A WinRM connection will be established.
  
  In addition to the above, some builders have custom communicators they
  can use. For example, the Docker builder has a "docker" communicator
  that uses `docker exec` and `docker cp` to execute scripts and copy
  files.

- `pause_before_connecting` (duration string | ex: "1h5m2s") - We recommend that you enable SSH or WinRM as the very last step in your
  guest's bootstrap script, but sometimes you may have a race condition
  where you need Packer to wait before attempting to connect to your
  guest.
  
  If you end up in this situation, you can use the template option
  `pause_before_connecting`. By default, there is no pause. For example if
  you set `pause_before_connecting` to `10m` Packer will check whether it
  can connect, as normal. But once a connection attempt is successful, it
  will disconnect and then wait 10 minutes before connecting to the guest
  and beginning provisioning.

<!-- End of code generated from the comments of the Config struct in communicator/config.go; -->


##### SSH

<!-- Code generated from the comments of the SSH struct in communicator/config.go; DO NOT EDIT MANUALLY -->

- `ssh_host` (string) - The address to SSH to. This usually is automatically configured by the
  builder.

- `ssh_port` (int) - The port to connect to SSH. This defaults to `22`.

- `ssh_username` (string) - The username to connect to SSH with. Required if using SSH.

- `ssh_password` (string) - A plaintext password to use to authenticate with SSH.

- `ssh_ciphers` ([]string) - This overrides the value of ciphers supported by default by Golang.
  The default value is [
    "aes128-gcm@openssh.com",
    "chacha20-poly1305@openssh.com",
    "aes128-ctr", "aes192-ctr", "aes256-ctr",
  ]
  
  Valid options for ciphers include:
  "aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com",
  "chacha20-poly1305@openssh.com",
  "arcfour256", "arcfour128", "arcfour", "aes128-cbc", "3des-cbc",

- `ssh_clear_authorized_keys` (bool) - If true, Packer will attempt to remove its temporary key from
  `~/.ssh/authorized_keys` and `/root/.ssh/authorized_keys`. This is a
  mostly cosmetic option, since Packer will delete the temporary private
  key from the host system regardless of whether this is set to true
  (unless the user has set the `-debug` flag). Defaults to "false";
  currently only works on guests with `sed` installed.

- `ssh_key_exchange_algorithms` ([]string) - If set, Packer will override the value of key exchange (kex) algorithms
  supported by default by Golang. Acceptable values include:
  "curve25519-sha256@libssh.org", "ecdh-sha2-nistp256",
  "ecdh-sha2-nistp384", "ecdh-sha2-nistp521",
  "diffie-hellman-group14-sha1", and "diffie-hellman-group1-sha1".

- `ssh_certificate_file` (string) - Path to user certificate used to authenticate with SSH.
  The `~` can be used in path and will be expanded to the
  home directory of current user.

- `ssh_pty` (bool) - If `true`, a PTY will be requested for the SSH connection. This defaults
  to `false`.

- `ssh_timeout` (duration string | ex: "1h5m2s") - The time to wait for SSH to become available. Packer uses this to
  determine when the machine has booted so this is usually quite long.
  Example value: `10m`.
  This defaults to `5m`, unless `ssh_handshake_attempts` is set.

- `ssh_disable_agent_forwarding` (bool) - If true, SSH agent forwarding will be disabled. Defaults to `false`.

- `ssh_handshake_attempts` (int) - The number of handshakes to attempt with SSH once it can connect.
  This defaults to `10`, unless a `ssh_timeout` is set.

- `ssh_bastion_host` (string) - A bastion host to use for the actual SSH connection.

- `ssh_bastion_port` (int) - The port of the bastion host. Defaults to `22`.

- `ssh_bastion_agent_auth` (bool) - If `true`, the local SSH agent will be used to authenticate with the
  bastion host. Defaults to `false`.

- `ssh_bastion_username` (string) - The username to connect to the bastion host.

- `ssh_bastion_password` (string) - The password to use to authenticate with the bastion host.

- `ssh_bastion_interactive` (bool) - If `true`, the keyboard-interactive used to authenticate with bastion host.

- `ssh_bastion_private_key_file` (string) - Path to a PEM encoded private key file to use to authenticate with the
  bastion host. The `~` can be used in path and will be expanded to the
  home directory of current user.

- `ssh_bastion_certificate_file` (string) - Path to user certificate used to authenticate with bastion host.
  The `~` can be used in path and will be expanded to the
  home directory of current user.

- `ssh_file_transfer_method` (string) - `scp` or `sftp` - How to transfer files, Secure copy (default) or SSH
  File Transfer Protocol.
  
  **NOTE**: Guests using Windows with Win32-OpenSSH v9.1.0.0p1-Beta, scp
  (the default protocol for copying data) returns a a non-zero error code since the MOTW
  cannot be set, which cause any file transfer to fail. As a workaround you can override the transfer protocol
  with SFTP instead `ssh_file_transfer_method = "sftp"`.

- `ssh_proxy_host` (string) - A SOCKS proxy host to use for SSH connection

- `ssh_proxy_port` (int) - A port of the SOCKS proxy. Defaults to `1080`.

- `ssh_proxy_username` (string) - The optional username to authenticate with the proxy server.

- `ssh_proxy_password` (string) - The optional password to use to authenticate with the proxy server.

- `ssh_keep_alive_interval` (duration string | ex: "1h5m2s") - How often to send "keep alive" messages to the server. Set to a negative
  value (`-1s`) to disable. Example value: `10s`. Defaults to `5s`.

- `ssh_read_write_timeout` (duration string | ex: "1h5m2s") - The amount of time to wait for a remote command to end. This might be
  useful if, for example, packer hangs on a connection after a reboot.
  Example: `5m`. Disabled by default.

- `ssh_remote_tunnels` ([]string) - Remote tunnels forward a port from your local machine to the instance.
  Format: ["REMOTE_PORT:LOCAL_HOST:LOCAL_PORT"]
  Example: "9090:localhost:80" forwards localhost:9090 on your machine to port 80 on the instance.

- `ssh_local_tunnels` ([]string) - Local tunnels forward a port from the instance to your local machine.
  Format: ["LOCAL_PORT:REMOTE_HOST:REMOTE_PORT"]
  Example: "8080:localhost:3000" allows the instance to access your local machine’s port 3000 via localhost:8080.

<!-- End of code generated from the comments of the SSH struct in communicator/config.go; -->


<!-- Code generated from the comments of the SSHTemporaryKeyPair struct in communicator/config.go; DO NOT EDIT MANUALLY -->

- `temporary_key_pair_type` (string) - `dsa` | `ecdsa` | `ed25519` | `rsa` ( the default )
  
  Specifies the type of key to create. The possible values are 'dsa',
  'ecdsa', 'ed25519', or 'rsa'.
  
  NOTE: DSA is deprecated and no longer recognized as secure, please
  consider other alternatives like RSA or ED25519.

- `temporary_key_pair_bits` (int) - Specifies the number of bits in the key to create. For RSA keys, the
  minimum size is 1024 bits and the default is 4096 bits. Generally, 3072
  bits is considered sufficient. DSA keys must be exactly 1024 bits as
  specified by FIPS 186-2. For ECDSA keys, bits determines the key length
  by selecting from one of three elliptic curve sizes: 256, 384 or 521
  bits. Attempting to use bit lengths other than these three values for
  ECDSA keys will fail. Ed25519 keys have a fixed length and bits will be
  ignored.
  
  NOTE: DSA is deprecated and no longer recognized as secure as specified
  by FIPS 186-5, please consider other alternatives like RSA or ED25519.

<!-- End of code generated from the comments of the SSHTemporaryKeyPair struct in communicator/config.go; -->


##### Windows Remote Management (WinRM)

<!-- Code generated from the comments of the WinRM struct in communicator/config.go; DO NOT EDIT MANUALLY -->

- `winrm_username` (string) - The username to use to connect to WinRM.

- `winrm_password` (string) - The password to use to connect to WinRM.

- `winrm_host` (string) - The address for WinRM to connect to.
  
  NOTE: If using an Amazon EBS builder, you can specify the interface
  WinRM connects to via
  [`ssh_interface`](/packer/integrations/hashicorp/amazon/latest/components/builder/ebs#ssh_interface)

- `winrm_no_proxy` (bool) - Setting this to `true` adds the remote
  `host:port` to the `NO_PROXY` environment variable. This has the effect of
  bypassing any configured proxies when connecting to the remote host.
  Default to `false`.

- `winrm_port` (int) - The WinRM port to connect to. This defaults to `5985` for plain
  unencrypted connection and `5986` for SSL when `winrm_use_ssl` is set to
  true.

- `winrm_timeout` (duration string | ex: "1h5m2s") - The amount of time to wait for WinRM to become available. This defaults
  to `30m` since setting up a Windows machine generally takes a long time.

- `winrm_use_ssl` (bool) - If `true`, use HTTPS for WinRM.

- `winrm_insecure` (bool) - If `true`, do not check server certificate chain and host name.

- `winrm_use_ntlm` (bool) - If `true`, NTLMv2 authentication (with session security) will be used
  for WinRM, rather than default (basic authentication), removing the
  requirement for basic authentication to be enabled within the target
  guest. Further reading for remote connection authentication can be found
  [here](https://msdn.microsoft.com/en-us/library/aa384295(v=vs.85).aspx).

<!-- End of code generated from the comments of the WinRM struct in communicator/config.go; -->


### SSH Key Pair Automation

The builders can inject the current SSH key pair's public key into the template
using the `SSHPublicKey` template engine. This is the SSH public key as a line
in OpenSSH `authorized_keys` format.

When a private key is provided using `ssh_private_key_file`, the key's
corresponding public key can be accessed using the above engine.

- `ssh_private_key_file` (string) - Path to a PEM encoded private key file to use to authenticate with SSH.
  The `~` can be used in path and will be expanded to the home directory
  of current user.


If `ssh_password` and `ssh_private_key_file` are not specified, Packer will
automatically generate en ephemeral key pair. The key pair's public key can be
accessed using the template engine.

For example, the public key can be provided in the boot command as a URL encoded
string by appending `| urlquery` to the variable:

HCL Example:

```hcl
boot_command = [
  "<up><wait><tab> text ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg PACKER_USER={{ user `username` }} PACKER_AUTHORIZED_KEY={{ .SSHPublicKey | urlquery }}<enter>"
]
```

JSON Example

```json
"boot_command": [
  "<up><wait><tab> text ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg PACKER_USER={{ user `username` }} PACKER_AUTHORIZED_KEY={{ .SSHPublicKey | urlquery }}<enter>"
]
```

A kickstart can use those fields from the kernel command line by decoding the
URL-encoded public key:

```shell
%post

# Newly created users need the file/folder framework for SSH key authentication.
umask 0077
mkdir /etc/skel/.ssh
touch /etc/skel/.ssh/authorized_keys

# Loop over the command line. Set interesting variables.
for x in $(cat /proc/cmdline)
do
  case $x in
    PACKER_USER=*)
      PACKER_USER="${x#*=}"
      ;;
    PACKER_AUTHORIZED_KEY=*)
      # URL decode $encoded into $PACKER_AUTHORIZED_KEY
      encoded=$(echo "${x#*=}" | tr '+' ' ')
      printf -v PACKER_AUTHORIZED_KEY '%b' "${encoded//%/\\x}"
      ;;
  esac
done

# Create/configure packer user, if any.
if [ -n "$PACKER_USER" ]
then
  useradd $PACKER_USER
  echo "%$PACKER_USER ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers.d/$PACKER_USER
  [ -n "$PACKER_AUTHORIZED_KEY" ] && echo $PACKER_AUTHORIZED_KEY >> $(eval echo ~"$PACKER_USER")/.ssh/authorized_keys
fi

%end
```
