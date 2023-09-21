Type: `vmware-iso`
Artifact BuilderId: `mitchellh.vmware`
If remote_type is esx: Artifact BuilderId: `mitchellh.vmware-esx`

This VMware Packer builder is able to create VMware virtual machines from an ISO
file as a source. It currently supports building virtual machines on hosts
running [VMware Fusion](https://www.vmware.com/products/fusion/overview.html) for
OS X, [VMware
Workstation](https://www.vmware.com/products/workstation/overview.html) for Linux
and Windows, and [VMware Player](https://www.vmware.com/products/player/) on
Linux. It can also build machines directly on [VMware vSphere
Hypervisor](https://www.vmware.com/products/vsphere-hypervisor/) using SSH as
opposed to the vSphere API.

The builder builds a virtual machine by creating a new virtual machine from
scratch, booting it, installing an OS, provisioning software within the OS, then
shutting it down. The result of the VMware builder is a directory containing all
the files necessary to run the virtual machine.

## Basic Example

Here is a basic example. This example is not functional. It will start the OS
installer but then fail because we don't provide the preseed file for Ubuntu to
self-install. Still, the example serves to show the basic configuration:

**JSON**

```json
{
  "type": "vmware-iso",
  "iso_url": "http://old-releases.ubuntu.com/releases/precise/ubuntu-12.04.2-server-amd64.iso",
  "iso_checksum": "md5:af5f788aee1b32c4b2634734309cc9e9",
  "ssh_username": "packer",
  "ssh_password": "packer",
  "shutdown_command": "shutdown -P now"
}
```

**HCL2**

```hcl
source "vmware-iso" "basic-example" {
  iso_url = "http://old-releases.ubuntu.com/releases/precise/ubuntu-12.04.2-server-amd64.iso"
  iso_checksum = "md5:af5f788aee1b32c4b2634734309cc9e9"
  ssh_username = "packer"
  ssh_password = "packer"
  shutdown_command = "shutdown -P now"
}

build {
  sources = ["sources.vmware-iso.basic-example"]
}
```


## VMware-ISO Builder Configuration Reference

There are many configuration options available for the builder. In addition to
the items listed here, you will want to look at the general configuration
references for [ISO](#iso-configuration),
[HTTP](#http-directory-configuration),
[Floppy](#floppy-configuration),
[CD](#cd-configuration),
[Boot](#boot-configuration),
[Driver](#driver-configuration),
[Hardware](#hardware-configuration),
[Output](#output-configuration),
[Run](#run-configuration),
[Shutdown](#shutdown-configuration),
[Communicator](#communicator-configuration),
[Tools](#tools-configuration),
[vmx](#vmx-configuration),
[Export](#export-configuration),
configuration references, which are
necessary for this build to succeed and can be found further down the page.

### Optional:

<!-- Code generated from the comments of the Config struct in builder/vmware/iso/config.go; DO NOT EDIT MANUALLY -->

- `disk_size` (uint) - The size of the hard disk for the VM in megabytes.
  The builder uses expandable, not fixed-size virtual hard disks, so the
  actual file representing the disk will not use the full size unless it
  is full. By default this is set to 40000 (about 40 GB).

- `cdrom_adapter_type` (string) - The adapter type (or bus) that will be used
  by the cdrom device. This is chosen by default based on the disk adapter
  type. VMware tends to lean towards ide for the cdrom device unless
  sata is chosen for the disk adapter and so Packer attempts to mirror
  this logic. This field can be specified as either ide, sata, or scsi.

- `guest_os_type` (string) - The guest OS type being installed. This will be
  set in the VMware VMX. By default this is other. By specifying a more
  specific OS type, VMware may perform some optimizations or virtual hardware
  changes to better support the operating system running in the
  virtual machine. Valid values differ by platform and version numbers, and may
  not match other VMware API's representation of the guest OS names. Consult your
  platform for valid values.

- `version` (string) - The [vmx hardware
  version](http://kb.vmware.com/selfservice/microsites/search.do?language=en_US&cmd=displayKC&externalId=1003746)
  for the new virtual machine. Only the default value has been tested, any
  other value is experimental. Default value is `9`.

- `vm_name` (string) - This is the name of the VMX file for the new virtual
  machine, without the file extension. By default this is packer-BUILDNAME,
  where "BUILDNAME" is the name of the build.

- `vmx_disk_template_path` (string) - VMX Disk Template Path

- `vmx_template_path` (string) - Path to a [configuration template](/packer/docs/templates/legacy_json_templates/engine) that
  defines the contents of the virtual machine VMX file for VMware. The
  engine has access to the template variables `{{ .DiskNumber }}` and
  `{{ .DiskName }}`.
  
  This is for **advanced users only** as this can render the virtual machine
  non-functional. See below for more information. For basic VMX
  modifications, try `vmx_data` first.

- `snapshot_name` (string) - This is the name of the initial snapshot created after provisioning and cleanup.
  if left blank, no initial snapshot will be created

<!-- End of code generated from the comments of the Config struct in builder/vmware/iso/config.go; -->


### Extra Disk Configuration

#### Optional:

<!-- Code generated from the comments of the DiskConfig struct in builder/vmware/common/disk_config.go; DO NOT EDIT MANUALLY -->

- `disk_additional_size` ([]uint) - The size(s) of any additional
  hard disks for the VM in megabytes. If this is not specified then the VM
  will only contain a primary hard disk. The builder uses expandable, not
  fixed-size virtual hard disks, so the actual file representing the disk will
  not use the full size unless it is full.

- `disk_adapter_type` (string) - The adapter type of the VMware virtual disk to create. This option is
  for advanced usage, modify only if you know what you're doing. Some of
  the options you can specify are `ide`, `sata`, `nvme` or `scsi` (which
  uses the "lsilogic" scsi interface by default). If you specify another
  option, Packer will assume that you're specifying a `scsi` interface of
  that specified type. For more information, please consult [Virtual Disk
  Manager User's Guide](http://www.vmware.com/pdf/VirtualDiskManager.pdf)
  for desktop VMware clients. For ESXi, refer to the proper ESXi
  documentation.

- `vmdk_name` (string) - The filename of the virtual disk that'll be created,
  without the extension. This defaults to "disk".

- `disk_type_id` (string) - The type of VMware virtual disk to create. This
  option is for advanced usage.
  
    For desktop VMware clients:
  
    Type ID | Description
    ------- | ---
    `0`     | Growable virtual disk contained in a single file (monolithic sparse).
    `1`     | Growable virtual disk split into 2GB files (split sparse).
    `2`     | Preallocated virtual disk contained in a single file (monolithic flat).
    `3`     | Preallocated virtual disk split into 2GB files (split flat).
    `4`     | Preallocated virtual disk compatible with ESX server (VMFS flat).
    `5`     | Compressed disk optimized for streaming.
  
    The default is `1`.
  
    For ESXi, this defaults to `zeroedthick`. The available options for ESXi
    are: `zeroedthick`, `eagerzeroedthick`, `thin`. `rdm:dev`, `rdmp:dev`,
    `2gbsparse` are not supported. Due to default disk compaction, when using
    `zeroedthick` or `eagerzeroedthick` set `skip_compaction` to `true`.
  
    For more information, please consult the [Virtual Disk Manager User's
    Guide](https://www.vmware.com/pdf/VirtualDiskManager.pdf) for desktop
    VMware clients. For ESXi, refer to the proper ESXi documentation.

<!-- End of code generated from the comments of the DiskConfig struct in builder/vmware/common/disk_config.go; -->


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


#### Required:

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


#### Optional:

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


### Http directory configuration

<!-- Code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; DO NOT EDIT MANUALLY -->

Packer will create an http server serving `http_directory` when it is set, a
random free port will be selected and the architecture of the directory
referenced will be available in your builder.

Example usage from a builder:

```
wget http://{{ .HTTPIP }}:{{ .HTTPPort }}/foo/bar/preseed.cfg
```

<!-- End of code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; -->


#### Optional:

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

<!-- End of code generated from the comments of the HTTPConfig struct in multistep/commonsteps/http_config.go; -->


### Floppy configuration

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


#### Optional:

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


### CD configuration

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


### Shutdown configuration

#### Optional:

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


### Driver configuration

#### Optional:

<!-- Code generated from the comments of the DriverConfig struct in builder/vmware/common/driver_config.go; DO NOT EDIT MANUALLY -->

- `cleanup_remote_cache` (bool) - When set to true, Packer will cleanup the cache folder where the ISO file is stored during the build on the remote machine.
  By default, this is set to false.

- `fusion_app_path` (string) - Path to "VMware Fusion.app". By default this is
  /Applications/VMware Fusion.app but this setting allows you to
  customize this.

- `remote_type` (string) - The type of remote machine that will be used to
  build this VM rather than a local desktop product. The only value accepted
  for this currently is esx5. If this is not set, a desktop product will
  be used. By default, this is not set.

- `remote_datastore` (string) - The path to the datastore where the VM will be stored
  on the ESXi machine.

- `remote_cache_datastore` (string) - The path to the datastore where supporting files
  will be stored during the build on the remote machine.

- `remote_cache_directory` (string) - The path where the ISO and/or floppy files will
  be stored during the build on the remote machine. The path is relative to
  the remote_cache_datastore on the remote machine.

- `remote_host` (string) - The host of the remote machine used for access.
  This is only required if remote_type is enabled.

- `remote_port` (int) - The SSH port of the remote machine

- `remote_username` (string) - The SSH username used to access the remote machine.

- `remote_password` (string) - The SSH password for access to the remote machine.

- `remote_private_key_file` (string) - The SSH key for access to the remote machine.

- `skip_validate_credentials` (bool) - When Packer is preparing to run a
  remote esxi build, and export is not disable, by default it runs a no-op
  ovftool command to make sure that the remote_username and remote_password
  given are valid. If you set this flag to true, Packer will skip this
  validation. Default: false.

<!-- End of code generated from the comments of the DriverConfig struct in builder/vmware/common/driver_config.go; -->


### Hardware configuration

#### Optional:

<!-- Code generated from the comments of the HWConfig struct in builder/vmware/common/hw_config.go; DO NOT EDIT MANUALLY -->

- `cpus` (int) - The number of cpus to use when building the VM.

- `memory` (int) - The amount of memory to use when building the VM in megabytes.

- `cores` (int) - The number of cores per socket to use when building the VM. This
  corresponds to the cpuid.coresPerSocket option in the .vmx file.

- `network` (string) - This is the network type that the virtual machine will be created with.
  This can be one of the generic values that map to a device such as
  hostonly, nat, or bridged. If the network is not one of these values,
  then it is assumed to be a VMware network device. (VMnet0..x)

- `network_adapter_type` (string) - This is the ethernet adapter type the the virtual machine will be
  created with. By default the `e1000` network adapter type will be used
  by Packer. For more information, please consult [Choosing a network
  adapter for your virtual
  machine](https://kb.vmware.com/s/article/1001805) for desktop VMware
  clients. For ESXi, refer to the proper ESXi documentation.

- `network_name` (string) - The custom name of the network. Sets the vmx value "ethernet0.networkName"

- `sound` (bool) - Specify whether to enable VMware's virtual soundcard device when
  building the VM. Defaults to false.

- `usb` (bool) - Enable VMware's USB bus when building the guest VM. Defaults to false.
  To enable usage of the XHCI bus for USB 3 (5 Gbit/s), one can use the
  vmx_data option to enable it by specifying true for the usb_xhci.present
  property.

- `serial` (string) - This specifies a serial port to add to the VM. It has a format of
  `Type:option1,option2,...`. The field `Type` can be one of the following
  values: `FILE`, `DEVICE`, `PIPE`, `AUTO`, or `NONE`.
  
  * `FILE:path(,yield)` - Specifies the path to the local file to be used
    as the serial port.
  
    * `yield` (bool) - This is an optional boolean that specifies
      whether the vm should yield the cpu when polling the port. By
      default, the builder will assume this as `FALSE`.
  
  * `DEVICE:path(,yield)` - Specifies the path to the local device to be
     used as the serial port. If `path` is empty, then default to the first
    serial port.
  
    * `yield` (bool) - This is an optional boolean that specifies
      whether the vm should yield the cpu when polling the port. By
      default, the builder will assume this as `FALSE`.
  
  * `PIPE:path,endpoint,host(,yield)` - Specifies to use the named-pipe
    "path" as a serial port. This has a few options that determine how the
    VM should use the named-pipe.
  
    * `endpoint` (string) - Chooses the type of the VM-end, which can be
      either a `client` or `server`.
  
    * `host` (string)     - Chooses the type of the host-end, which can
      be either `app` (application) or `vm` (another virtual-machine).
  
    * `yield` (bool)      - This is an optional boolean that specifies
      whether the vm should yield the cpu when polling the port. By
      default, the builder will assume this as `FALSE`.
  
  * `AUTO:(yield)` - Specifies to use auto-detection to determine the
    serial port to use. This has one option to determine how the VM should
    support the serial port.
  
    * `yield` (bool) - This is an optional boolean that specifies
      whether the vm should yield the cpu when polling the port. By
      default, the builder will assume this as `FALSE`.
  
  * `NONE` - Specifies to not use a serial port. (default)

- `parallel` (string) - This specifies a parallel port to add to the VM. It has the format of
  `Type:option1,option2,...`. Type can be one of the following values:
  `FILE`, `DEVICE`, `AUTO`, or `NONE`.
  
  * `FILE:path` 		- Specifies the path to the local file to be used
    for the parallel port.
  
  * `DEVICE:path`	 	- Specifies the path to the local device to be used
    for the parallel port.
  
  * `AUTO:direction`   - Specifies to use auto-detection to determine the
    parallel port. Direction can be `BI` to specify bidirectional
    communication or `UNI` to specify unidirectional communication.
  
  * `NONE` 			- Specifies to not use a parallel port. (default)

<!-- End of code generated from the comments of the HWConfig struct in builder/vmware/common/hw_config.go; -->


### Output configuration

#### Optional:

<!-- Code generated from the comments of the OutputConfig struct in builder/vmware/common/output_config.go; DO NOT EDIT MANUALLY -->

- `output_directory` (string) - This is the path on your local machine (the one running Packer) to the
  directory where the resulting virtual machine will be created.
  This may be relative or absolute. If relative, the path is relative to
  the working directory when packer is executed.
  
  If you are running a remote esx build, the output_dir is the path on your
  local machine (the machine running Packer) to which Packer will export
  the vm if you have `"skip_export": false`. If you want to manage the
  virtual machine's path on the remote datastore, use `remote_output_dir`.
  
  This directory must not exist or be empty prior to running
  the builder.
  
  By default this is output-BUILDNAME where "BUILDNAME" is the name of the
  build.

- `remote_output_directory` (string) - This is the directoy on your remote esx host where you will save your
  vm, relative to your remote_datastore.
  
  This option's default value is your `vm_name`, and the final path of your
  vm will be vmfs/volumes/$remote_datastore/$vm_name/$vm_name.vmx where
  `$remote_datastore` and `$vm_name` match their corresponding template
  options
  
  For example, setting `"remote_output_directory": "path/to/subdir`
  will create a directory `/vmfs/volumes/remote_datastore/path/to/subdir`.
  
  Packer will not create the remote datastore for you; it must already
  exist. However, Packer will create all directories defined in the option
  that do not currently exist.
  
  This option will be ignored unless you are building on a remote esx host.

<!-- End of code generated from the comments of the OutputConfig struct in builder/vmware/common/output_config.go; -->


### Run configuration

<!-- Code generated from the comments of the RunConfig struct in builder/vmware/common/run_config.go; DO NOT EDIT MANUALLY -->

~> **Note:** If [vnc_over_websocket](#vnc_over_websocket) is set to true, any other VNC configuration will be ignored.

<!-- End of code generated from the comments of the RunConfig struct in builder/vmware/common/run_config.go; -->


#### Optional:

<!-- Code generated from the comments of the RunConfig struct in builder/vmware/common/run_config.go; DO NOT EDIT MANUALLY -->

- `headless` (bool) - Packer defaults to building VMware virtual machines
  by launching a GUI that shows the console of the machine being built. When
  this value is set to true, the machine will start without a console. For
  VMware machines, Packer will output VNC connection information in case you
  need to connect to the console to debug the build process.
  Some users have experienced issues where Packer cannot properly connect
  to a VM if it is headless; this appears to be a result of not ever having
  launched the VMWare GUI and accepting the evaluation license, or
  supplying a real license. If you experience this, launching VMWare and
  accepting the license should resolve your problem.

- `vnc_bind_address` (string) - The IP address that should be
  binded to for VNC. By default packer will use 127.0.0.1 for this. If you
  wish to bind to all interfaces use 0.0.0.0.

- `vnc_port_min` (int) - The minimum and maximum port
  to use for VNC access to the virtual machine. The builder uses VNC to type
  the initial boot_command. Because Packer generally runs in parallel,
  Packer uses a randomly chosen port in this range that appears available. By
  default this is 5900 to 6000. The minimum and maximum ports are
  inclusive.

- `vnc_port_max` (int) - VNC Port Max

- `vnc_disable_password` (bool) - Don't auto-generate a VNC password that
  is used to secure the VNC communication with the VM. This must be set to
  true if building on ESXi 6.5 and 6.7 with VNC enabled. Defaults to
  false.

- `vnc_over_websocket` (bool) - When set to true, Packer will connect to the remote VNC server over a websocket connection
  and any other VNC configuration option will be ignored.
  Remote builds using ESXi 6.7+ allows to connect to the VNC server only over websocket,
  for these the `vnc_over_websocket` must be set to true.

- `insecure_connection` (bool) - Do not validate VNC over websocket server's TLS certificate. Defaults to `false`.

<!-- End of code generated from the comments of the RunConfig struct in builder/vmware/common/run_config.go; -->


### Tools configuration

#### Optional:

<!-- Code generated from the comments of the ToolsConfig struct in builder/vmware/common/tools_config.go; DO NOT EDIT MANUALLY -->

- `tools_upload_flavor` (string) - The flavor of the VMware Tools ISO to
  upload into the VM. Valid values are darwin, linux, and windows. By
  default, this is empty, which means VMware tools won't be uploaded.

- `tools_upload_path` (string) - The path in the VM to upload the VMware tools. This only takes effect if
  `tools_upload_flavor` is non-empty. This is a [configuration
  template](/packer/docs/templates/legacy_json_templates/engine) that has a single valid variable:
  `Flavor`, which will be the value of `tools_upload_flavor`. By default
  the upload path is set to `{{.Flavor}}.iso`. This setting is not used
  when `remote_type` is `esx5`.

- `tools_source_path` (string) - The path on your local machine to fetch the vmware tools from. If this
  is not set but the tools_upload_flavor is set, then Packer will try to
  load the VMWare tools from the VMWare installation directory.

<!-- End of code generated from the comments of the ToolsConfig struct in builder/vmware/common/tools_config.go; -->


### VMX configuration

#### Optional:

<!-- Code generated from the comments of the VMXConfig struct in builder/vmware/common/vmx_config.go; DO NOT EDIT MANUALLY -->

- `vmx_data` (map[string]string) - Arbitrary key/values to enter
  into the virtual machine VMX file. This is for advanced users who want to
  set properties that aren't yet supported by the builder.

- `vmx_data_post` (map[string]string) - Identical to vmx_data,
  except that it is run after the virtual machine is shutdown, and before the
  virtual machine is exported.

- `vmx_remove_ethernet_interfaces` (bool) - Remove all ethernet interfaces
  from the VMX file after building. This is for advanced users who understand
  the ramifications, but is useful for building Vagrant boxes since Vagrant
  will create ethernet interfaces when provisioning a box. Defaults to
  false.

- `display_name` (string) - The name that will appear in your vSphere client,
  and will be used for the vmx basename. This will override the "displayname"
  value in your vmx file. It will also override the "displayname" if you have
  set it in the "vmx_data" Packer option. This option is useful if you are
  chaining vmx builds and want to make sure that the display name of each step
  in the chain is unique.

<!-- End of code generated from the comments of the VMXConfig struct in builder/vmware/common/vmx_config.go; -->


### Export configuration

#### Optional:

<!-- Code generated from the comments of the ExportConfig struct in builder/vmware/common/export_config.go; DO NOT EDIT MANUALLY -->

- `format` (string) - Either "ovf", "ova" or "vmx", this specifies the output
  format of the exported virtual machine. This defaults to "ovf" for
  remote (esx) builds, and "vmx" for local builds.
  Before using this option, you need to install ovftool.
  Since ovftool is only capable of password based authentication
  remote_password must be set when exporting the VM from a remote instance.
  If you are building locally, Packer will create a vmx and then
  export that vm to an ovf or ova. Packer will not delete the vmx and vmdk
  files; this is left up to the user if you don't want to keep those
  files.

- `ovftool_options` ([]string) - Extra options to pass to ovftool during export. Each item in the array
  is a new argument. The options `--noSSLVerify`, `--skipManifestCheck`,
  and `--targetType` are used by Packer for remote exports, and should not
  be passed to this argument. For ovf/ova exports from local builds, Packer
  does not automatically set any ovftool options.

- `skip_export` (bool) - Defaults to `false`. When true, Packer will not export the VM. This can
  be useful if the build output is not the resultant image, but created
  inside the VM.

- `keep_registered` (bool) - Set this to true if you would like to keep a remotely-built
  VM registered with the remote ESXi server. If you do not need to export
  the vm, then also set `skip_export: true` in order to avoid unnecessarily
  using ovftool to export the vm. Defaults to false.

- `skip_compaction` (bool) - VMware-created disks are defragmented and
  compacted at the end of the build process using vmware-vdiskmanager or
  vmkfstools in ESXi. In certain rare cases, this might actually end up
  making the resulting disks slightly larger. If you find this to be the case,
  you can disable compaction using this configuration value. Defaults to
  false. Default to true for ESXi when disk_type_id is not explicitly
  defined and false otherwise.

<!-- End of code generated from the comments of the ExportConfig struct in builder/vmware/common/export_config.go; -->


### Communicator configuration

#### Optional common fields:

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


#### Optional SSH fields:

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
  with SFTP instead `ssh_file_transfer_protocol = "sftp"`.

- `ssh_proxy_host` (string) - A SOCKS proxy host to use for SSH connection

- `ssh_proxy_port` (int) - A port of the SOCKS proxy. Defaults to `1080`.

- `ssh_proxy_username` (string) - The optional username to authenticate with the proxy server.

- `ssh_proxy_password` (string) - The optional password to use to authenticate with the proxy server.

- `ssh_keep_alive_interval` (duration string | ex: "1h5m2s") - How often to send "keep alive" messages to the server. Set to a negative
  value (`-1s`) to disable. Example value: `10s`. Defaults to `5s`.

- `ssh_read_write_timeout` (duration string | ex: "1h5m2s") - The amount of time to wait for a remote command to end. This might be
  useful if, for example, packer hangs on a connection after a reboot.
  Example: `5m`. Disabled by default.

- `ssh_remote_tunnels` ([]string) - 

- `ssh_local_tunnels` ([]string) - 

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


#### Optional WinRM fields:

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


-> **Note**: for the `HTTPIP` to be resolved correctly, your VM's network
configuration has to include a `hostonly` or `nat` type network interface.
If you are using this feature, it is recommended to leave the default network
configuration while you are building the VM, and use the `vmx_data_post` hook
to modify the network configuration after the VM is done building.

### Optional:

<!-- Code generated from the comments of the VNCConfig struct in bootcommand/config.go; DO NOT EDIT MANUALLY -->

- `disable_vnc` (bool) - Whether to create a VNC connection or not. A boot_command cannot be used
  when this is true. Defaults to false.

- `boot_key_interval` (duration string | ex: "1h5m2s") - Time in ms to wait between each key press

<!-- End of code generated from the comments of the VNCConfig struct in bootcommand/config.go; -->


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


For more examples of various boot commands, see the sample projects from our
[community templates page](/community-tools#templates).

## VMX Template

The heart of a VMware machine is the "vmx" file. This contains all the virtual
hardware metadata necessary for the VM to function. Packer by default uses a
[safe, flexible VMX
file](https://github.com/hashicorp/packer/blob/20541a7eda085aa5cf35bfed5069592ca49d106e/builder/vmware/step_create_vmx.go#L84).
But for advanced users, this template can be customized. This allows Packer to
build virtual machines of effectively any guest operating system type.

~> **This is an advanced feature.** Modifying the VMX template can easily
cause your virtual machine to not boot properly. Please only modify the template
if you know what you're doing.

Within the template, a handful of variables are available so that your template
can continue working with the rest of the Packer machinery. Using these
variables isn't required, however.

- `Name` - The name of the virtual machine.
- `GuestOS` - The VMware-valid guest OS type.
- `DiskName` - The filename (without the suffix) of the main virtual disk.
- `ISOPath` - The path to the ISO to use for the OS installation.
- `Version` - The Hardware version VMWare will execute this vm under. Also
  known as the `virtualhw.version`.

## Building on a Remote vSphere Hypervisor

In addition to using the desktop products of VMware locally to build virtual
machines, Packer can use a remote VMware Hypervisor to build the virtual
machine.

-> **Note:** Packer supports ESXi 5.1 and above.

Before using a remote vSphere Hypervisor, you need to enable GuestIPHack by
running the following command:

```shell-session
$ esxcli system settings advanced set -o /Net/GuestIPHack -i 1
```

When using a remote VMware Hypervisor, the builder still downloads the ISO and
various files locally, and uploads these to the remote machine. Packer currently
uses SSH to communicate to the ESXi machine rather than the vSphere API.
If you want to use vSphere API, see the [vsphere-iso](/packer/integrations/hashicorp/vsphere/latest/components/builder/vsphere-iso) builder.

Packer also requires VNC to issue boot commands during a build, which may be
disabled on some remote VMware Hypervisors. Please consult the appropriate
documentation on how to update VMware Hypervisor's firewall to allow these
connections. VNC can be disabled by not setting a `boot_command` and setting
`disable_vnc` to `true`.

Please note that you should disable vMotion for the host you intend to run
Packer builds on; a vMotion event will cause the Packer build to fail.

To use a remote VMware vSphere Hypervisor to build your virtual machine, fill in
the required `remote_*` configurations:

- `remote_type` - This must be set to "esx5".

- `remote_host` - The host of the remote machine.

Additionally, there are some optional configurations that you'll likely have to
modify as well:

- `remote_port` - The SSH port of the remote machine

- `remote_datastore` - The path to the datastore where the VM will be stored
on the ESXi machine.

- `remote_cache_datastore` - The path to the datastore where supporting files
will be stored during the build on the remote machine.

- `remote_cache_directory` - The path where the ISO and/or floppy files will
be stored during the build on the remote machine. The path is relative to
the `remote_cache_datastore` on the remote machine.

- `remote_username` - The SSH username used to access the remote machine.

- `remote_password` - The SSH password for access to the remote machine.

- `remote_private_key_file` - The SSH key for access to the remote machine.

- `format` (string) - Either "ovf", "ova" or "vmx", this specifies the output
format of the exported virtual machine. This defaults to "ovf".
Before using this option, you need to install `ovftool`. This option
currently only works when option remote_type is set to "esx5".
Since ovftool is only capable of password based authentication
`remote_password` must be set when exporting the VM.

- `vnc_disable_password` - This must be set to "true" when using VNC with
ESXi 6.5 or 6.7.


### VNC port discovery

Packer needs to decide on a port to use for VNC when building remotely. To find
an open port, we try to connect to ports in the range of `vnc_port_min` to
`vnc_port_max`. If we notice something is listening on a port in the range, we
try to connect to the next one, and so on until we find a port that has nothing
listening on it. If you have many clients building on the ESXi host, there
might be competition for the VNC ports. You can adjust how long Packer waits
for a connection timeout by setting `PACKER_ESXI_VNC_PROBE_TIMEOUT`. This
defaults to 15 seconds. Set this shorter if VNC connections are refused, and
set it longer if Packer can't find an open port. This is intended as an
advanced configuration option. Please make sure your firewall settings are
correct before adjusting.

### Using a Floppy for Linux kickstart file or preseed

Depending on your network configuration, it may be difficult to use packer's
built-in HTTP server with ESXi. Instead, you can provide a kickstart or preseed
file by attaching a floppy disk. An example below, based on RHEL:

```json
{
  "builders": [
    {
      "type": "vmware-iso",
      "floppy_files": ["folder/ks.cfg"],
      "boot_command": "<tab> text ks=floppy <enter><wait>"
    }
  ]
}
```

It's also worth noting that `ks=floppy` has been deprecated. Later versions of
the Anaconda installer (used in RHEL/CentOS 7 and Fedora) may require
a different syntax to source a kickstart file from a mounted floppy image.

```json
{
  "builders": [
    {
      "type": "vmware-iso",
      "floppy_files": ["folder/ks.cfg"],
      "boot_command": "<tab> inst.text inst.ks=hd:fd0:/ks.cfg <enter><wait>"
    }
  ]
}
```

### SSH key pair automation

The VMware builders can inject the current SSH key pair's public key into
the template using the `SSHPublicKey` template engine. This is the SSH public
key as a line in OpenSSH authorized_keys format.

When a private key is provided using `ssh_private_key_file`, the key's
corresponding public key can be accessed using the above engine.

- `ssh_private_key_file` (string) - Path to a PEM encoded private key file to use to authenticate with SSH.
  The `~` can be used in path and will be expanded to the home directory
  of current user.


If `ssh_password` and `ssh_private_key_file` are not specified, Packer will
automatically generate en ephemeral key pair. The key pair's public key can
be accessed using the template engine.

For example, the public key can be provided in the boot command as a URL
encoded string by appending `| urlquery` to the variable:

In JSON:

```json
"boot_command": [
  "<up><wait><tab> text ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg PACKER_USER={{ user `username` }} PACKER_AUTHORIZED_KEY={{ .SSHPublicKey | urlquery }}<enter>"
]
```

In HCL2:

```hcl
boot_command = [
  "<up><wait><tab> text ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg PACKER_USER={{ user `username` }} PACKER_AUTHORIZED_KEY={{ .SSHPublicKey | urlquery }}<enter>"
]
```

A kickstart could then leverage those fields from the kernel command line by
decoding the URL-encoded public key:

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
