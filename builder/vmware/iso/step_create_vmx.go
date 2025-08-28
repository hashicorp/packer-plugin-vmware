// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package iso

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

type vmxTemplateData struct {
	Name    string
	GuestOS string
	ISOPath string
	Version string

	Firmware   string
	SecureBoot string

	CpuCount   string
	MemorySize string

	DiskName string
	common.DiskAndCDConfigData

	NetworkType    string
	NetworkDevice  string
	NetworkAdapter string

	SoundPresent string
	UsbPresent   string

	SerialPresent  string
	SerialType     string
	SerialEndpoint string
	SerialHost     string
	SerialYield    string
	SerialFilename string
	SerialAuto     string

	ParallelPresent       string
	ParallelBidirectional string
	ParallelFilename      string
	ParallelAuto          string

	HardwareAssistedVirtualization bool
}

type additionalDiskTemplateData struct {
	DiskUnit   int
	DiskNumber int
	DiskName   string
	DiskType   string
}

// This step creates the VMX file for the VM.
type stepCreateVMX struct {
	tempDir string
}

/* regular steps */

func (s *stepCreateVMX) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	isoPath := state.Get("iso_path").(string)
	ui := state.Get("ui").(packersdk.Ui)

	// Ensure ISO path is absolute.
	absISOPath, err := filepath.Abs(isoPath)
	if err != nil {
		err := fmt.Errorf("error making ISO path absolute: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Use the absolute ISO path directly.
	isoPath = absISOPath

	ui.Say("Generating the .vmx file...")

	vmxTemplate := DefaultVMXTemplate
	if config.VMXTemplatePath != "" {
		f, err := os.Open(config.VMXTemplatePath)
		if err != nil {
			err := fmt.Errorf("error reading VMX template: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		defer f.Close()

		rawBytes, err := io.ReadAll(f)
		if err != nil {
			err := fmt.Errorf("error reading VMX template: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		vmxTemplate = string(rawBytes)
	}

	diskAndCDConfigData := common.DefaultDiskAndCDROMTypes(config.DiskAdapterType, config.CdromAdapterType)
	ictx := config.ctx

	// Mount extra VMDKs we created earlier.
	if len(config.AdditionalDiskSize) > 0 {

		incrementer := 1

		// Extra VMDKs after primary disk and CDROM.
		unitSkip := 2

		// If the CD-ROM is on a different bus we only have to skip the primary disk's unit.
		if diskAndCDConfigData.CdromType != diskAndCDConfigData.DiskType {
			unitSkip = 1
		}

		for i := range config.AdditionalDiskSize {
			// Slot 7 is special and reserved, so we need to skip that index.
			if i+1 == 7 {
				incrementer = 2
				unitSkip++
			}
			ictx.Data = &additionalDiskTemplateData{
				DiskUnit:   i + unitSkip,
				DiskNumber: i + incrementer,
				DiskName:   config.DiskName,
				DiskType:   diskAndCDConfigData.DiskType,
			}

			diskTemplate := DefaultAdditionalDiskTemplate
			if config.VMXDiskTemplatePath != "" {
				rawBytes, err := os.ReadFile(config.VMXDiskTemplatePath)
				if err != nil {
					err := fmt.Errorf("error reading VMX disk template: %s", err)
					state.Put("error", err)
					ui.Error(err.Error())
					return multistep.ActionHalt
				}
				diskTemplate = string(rawBytes)
			}

			diskContents, err := interpolate.Render(diskTemplate, &ictx)
			if err != nil {
				err := fmt.Errorf("error preparing VMX template for additional disk: %s", err)
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}

			vmxTemplate += diskContents
		}
	}

	templateData := vmxTemplateData{
		Name:           config.VMName,
		GuestOS:        config.GuestOSType,
		DiskName:       config.DiskName,
		Version:        strconv.Itoa(config.Version),
		ISOPath:        isoPath,
		NetworkAdapter: config.NetworkAdapterType,

		SoundPresent: map[bool]string{true: "TRUE", false: "FALSE"}[config.Sound],
		UsbPresent:   map[bool]string{true: "TRUE", false: "FALSE"}[config.USB],

		SerialPresent:   "FALSE",
		ParallelPresent: "FALSE",
	}

	templateData.DiskAndCDConfigData = diskAndCDConfigData

	// Now that we figured out the CD-ROM device to add, store it
	// to the list of temporary build devices in our statebag
	tmpBuildDevices := state.Get("temporaryDevices").([]string)
	tmpCdromDevice := fmt.Sprintf("%s0:%s", templateData.CdromType, templateData.CdromTypePrimarySecondary)
	tmpBuildDevices = append(tmpBuildDevices, tmpCdromDevice)
	state.Put("temporaryDevices", tmpBuildDevices)

	// Assign the network adapter type into the template if one was specified.
	networkAdapter := strings.ToLower(config.NetworkAdapterType)
	if networkAdapter != "" {
		templateData.NetworkAdapter = networkAdapter
	}

	// Check the network type that the user specified
	network := config.Network

	driver := state.Get("driver").(common.Driver).GetVmwareDriver()

	// check to see if the driver implements a network mapper for mapping
	// the network-type to its device-name.
	if driver.NetworkMapper != nil {

		// read network map configuration into a NetworkNameMapper.
		netmap, err := driver.NetworkMapper()
		if err != nil {
			err := fmt.Errorf("error reading network map configuration: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// try and convert the specified network to a device.
		devices, err := netmap.NameIntoDevices(network)

		if err == nil && len(devices) > 0 {
			// If multiple devices exist, for example for network "nat", VMware chooses
			// the actual device. Only type "custom" allows the exact choice of a
			// specific virtual network (see below). We allow VMware to choose the device
			// and for device-specific operations like GuestIP, try to go over all
			// devices that match a name (e.g. "nat").
			templateData.NetworkType = network
			templateData.NetworkDevice = ""
		} else {
			// otherwise, we were unable to find the type, so assume it's a custom device
			templateData.NetworkType = "custom"
			templateData.NetworkDevice = network
		}
	} else {
		templateData.NetworkType = common.DefaultNetworkType
		templateData.NetworkDevice = network
		network = common.DefaultNetworkType
	}

	// store the network so that we can later figure out what ip address to bind to
	state.Put("vmnetwork", network)

	// check if serial port has been configured
	if !config.HasSerial() {
		templateData.SerialPresent = "FALSE"
	} else {
		serial, err := config.ReadSerial()
		if err != nil {
			err := fmt.Errorf("error processing VMX template: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		templateData.SerialPresent = "TRUE"
		templateData.SerialFilename = ""
		templateData.SerialYield = ""
		templateData.SerialEndpoint = ""
		templateData.SerialHost = ""
		templateData.SerialAuto = "FALSE"

		switch config.Firmware {
		case common.FirmwareTypeBios:
			templateData.Firmware = common.FirmwareTypeBios
		case common.FirmwareTypeUEFI, common.FirmwareTypeUEFISecure:
			templateData.Firmware = common.FirmwareTypeUEFI
			if config.Firmware == common.FirmwareTypeUEFISecure {
				templateData.SecureBoot = "TRUE"
			}
		}

		if config.CpuCount > 0 {
			templateData.CpuCount = strconv.Itoa(config.CpuCount)
		}

		if config.MemorySize > 0 {
			templateData.MemorySize = strconv.Itoa(config.MemorySize)
		} else {
			templateData.MemorySize = strconv.Itoa(common.DefaultMemorySize)
		}

		switch serial.Union.(type) {
		case *common.SerialConfigPipe:
			templateData.SerialType = "pipe"
			templateData.SerialEndpoint = serial.Pipe.Endpoint
			templateData.SerialHost = serial.Pipe.Host
			templateData.SerialYield = serial.Pipe.Yield
			templateData.SerialFilename = filepath.FromSlash(serial.Pipe.Filename)
		case *common.SerialConfigFile:
			templateData.SerialType = "file"
			templateData.SerialFilename = filepath.FromSlash(serial.File.Filename)
		case *common.SerialConfigDevice:
			templateData.SerialType = "device"
			templateData.SerialFilename = filepath.FromSlash(serial.Device.Devicename)
		case *common.SerialConfigAuto:
			templateData.SerialType = "device"
			templateData.SerialFilename = filepath.FromSlash(serial.Auto.Devicename)
			templateData.SerialYield = serial.Auto.Yield
			templateData.SerialAuto = "TRUE"
		case nil:
			templateData.SerialPresent = "FALSE"
		default:
			err := fmt.Errorf("error processing VMX template: %v", serial)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Check if parallel port has been configured.
	if !config.HasParallel() {
		templateData.ParallelPresent = "FALSE"
	} else {
		parallel, err := config.ReadParallel()
		if err != nil {
			err := fmt.Errorf("error processing VMX template: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		templateData.ParallelAuto = "FALSE"
		switch parallel.Union.(type) {
		case *common.ParallelPortFile:
			templateData.ParallelPresent = "TRUE"
			templateData.ParallelFilename = filepath.FromSlash(parallel.File.Filename)
		case *common.ParallelPortDevice:
			templateData.ParallelPresent = "TRUE"
			templateData.ParallelBidirectional = parallel.Device.Bidirectional
			templateData.ParallelFilename = filepath.FromSlash(parallel.Device.Devicename)
		case *common.ParallelPortAuto:
			templateData.ParallelPresent = "TRUE"
			templateData.ParallelAuto = "TRUE"
			templateData.ParallelBidirectional = parallel.Auto.Bidirectional
		case nil:
			templateData.ParallelPresent = "FALSE"
		default:
			err := fmt.Errorf("error processing VMX template: %v", parallel)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	// Set virtual hardware-assisted virtualization.
	templateData.HardwareAssistedVirtualization = config.HardwareAssistedVirtualization

	ictx.Data = &templateData

	// render the .vmx template
	vmxContents, err := interpolate.Render(vmxTemplate, &ictx)
	if err != nil {
		err := fmt.Errorf("error processing VMX template: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	vmxDir := config.OutputDir

	// Now to handle options that will modify the template without using "vmxTemplateData"
	vmxData := common.ParseVMX(vmxContents)

	// If no cpus were specified, then remove the entry to use the default
	if vmxData["numvcpus"] == "" {
		delete(vmxData, "numvcpus")
	}

	// If some number of cores were specified, then update "cpuid.coresPerSocket" with the requested value
	if config.CoreCount > 0 {
		vmxData["cpuid.corespersocket"] = strconv.Itoa(config.CoreCount)
	}

	// Write the vmxData to the vmxPath
	vmxPath := filepath.Join(vmxDir, config.VMName+".vmx")
	if err := common.WriteVMX(vmxPath, vmxData); err != nil {
		err = fmt.Errorf("error creating VMX file: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("vmx_path", vmxPath)

	return multistep.ActionContinue
}

func (s *stepCreateVMX) Cleanup(multistep.StateBag) {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// DefaultVMXTemplate is a constant string template for generating the configuration file with dynamic placeholders.
const DefaultVMXTemplate = `
.encoding = "UTF-8"

displayName = "{{ .Name }}"

// Firmware
{{ if .Firmware }}firmware = "{{ .Firmware }}"{{ end }}
{{ if .SecureBoot }}uefi.secureBoot.enabled = "TRUE"{{ end }}

// Hardware
numvcpus = "{{ .CpuCount }}"
memsize = "{{ .MemorySize }}"

config.version = "8"
virtualHW.productCompatibility = "hosted"
virtualHW.version = "{{ .Version }}"

// Virtual Hardware-Assisted Virtualization
vhv.enable = "{{if .HardwareAssistedVirtualization -}} TRUE {{- else -}} FALSE {{- end -}}"

// Bootup
nvram = "{{ .Name }}.nvram"

floppy0.present = "FALSE"
bios.bootOrder = "hdd,cdrom"

// Configuration
extendedConfigFile = "{{ .Name }}.vmxf"
gui.fullScreenAtPowerOn = "FALSE"
gui.viewModeAtPowerOn = "windowed"
hgfs.linkRootShare = "TRUE"
hgfs.mapRootShare = "TRUE"
isolation.tools.hgfs.disable = "FALSE"
proxyApps.publishToHost = "FALSE"
replay.filename = ""
replay.supported = "FALSE"

checkpoint.vmState = ""
vmotion.checkpointFBSize = "65536000"

// Power control
cleanShutdown = "TRUE"
powerType.powerOff = "soft"
powerType.powerOn = "soft"
powerType.reset = "soft"
powerType.suspend = "soft"

// Tools
guestOS = "{{ .GuestOS }}"
tools.syncTime = "TRUE"
tools.upgrade.policy = "upgradeAtPowerCycle"

// Bus
pciBridge0.pciSlotNumber = "17"
pciBridge0.present = "TRUE"
pciBridge4.functions = "8"
pciBridge4.pciSlotNumber = "21"
pciBridge4.present = "TRUE"
pciBridge4.virtualDev = "pcieRootPort"
pciBridge5.functions = "8"
pciBridge5.pciSlotNumber = "22"
pciBridge5.present = "TRUE"
pciBridge5.virtualDev = "pcieRootPort"
pciBridge6.functions = "8"
pciBridge6.pciSlotNumber = "23"
pciBridge6.present = "TRUE"
pciBridge6.virtualDev = "pcieRootPort"
pciBridge7.functions = "8"
pciBridge7.pciSlotNumber = "24"
pciBridge7.present = "TRUE"
pciBridge7.virtualDev = "pcieRootPort"

ehci.present = "TRUE"
ehci.pciSlotNumber = "34"

vmci0.present = "TRUE"
vmci0.id = "1861462627"
vmci0.pciSlotNumber = "35"

hpet0.present = "TRUE"

// Network Adapter
ethernet0.addressType = "generated"
ethernet0.bsdName = "en0"
ethernet0.connectionType = "{{ .NetworkType }}"
ethernet0.vnet = "{{ .NetworkDevice }}"
ethernet0.displayName = "Ethernet"
ethernet0.linkStatePropagation.enable = "FALSE"
ethernet0.pciSlotNumber = "33"
ethernet0.present = "TRUE"
ethernet0.virtualDev = "{{ .NetworkAdapter }}"
ethernet0.wakeOnPcktRcv = "FALSE"

// Hard disks
scsi0.present = "{{ .ScsiPresent }}"
scsi0.virtualDev = "{{ .ScsiDiskAdapterType }}"
scsi0.pciSlotNumber = "16"
scsi0:0.redo = ""
sata0.present = "{{ .SataPresent }}"
nvme0.present = "{{ .NvmePresent }}"

{{ .DiskType }}0:0.present = "TRUE"
{{ .DiskType }}0:0.fileName = "{{ .DiskName }}.vmdk"

{{ .CdromType }}0:{{ .CdromTypePrimarySecondary }}.present = "TRUE"
{{ .CdromType }}0:{{ .CdromTypePrimarySecondary }}.fileName = "{{ .ISOPath }}"
{{ .CdromType }}0:{{ .CdromTypePrimarySecondary }}.deviceType = "cdrom-image"

// Sound
sound.startConnected = "{{ .SoundPresent }}"
sound.present = "{{ .SoundPresent }}"
sound.fileName = "-1"
sound.autodetect = "TRUE"

// USB
usb.pciSlotNumber = "32"
usb.present = "{{ .UsbPresent }}"

// Serial
serial0.present = "{{ .SerialPresent }}"
serial0.startConnected = "{{ .SerialPresent }}"
serial0.fileName = "{{ .SerialFilename }}"
serial0.autodetect = "{{ .SerialAuto }}"
serial0.fileType = "{{ .SerialType }}"
serial0.yieldOnMsrRead = "{{ .SerialYield }}"
serial0.pipe.endPoint = "{{ .SerialEndpoint }}"
serial0.tryNoRxLoss = "{{ .SerialHost }}"

// Parallel
parallel0.present = "{{ .ParallelPresent }}"
parallel0.startConnected = "{{ .ParallelPresent }}"
parallel0.fileName = "{{ .ParallelFilename }}"
parallel0.autodetect = "{{ .ParallelAuto }}"
parallel0.bidirectional = "{{ .ParallelBidirectional }}"
`

const DefaultAdditionalDiskTemplate = `
{{ .DiskType }}0:{{ .DiskUnit }}.fileName = "{{ .DiskName}}-{{ .DiskNumber }}.vmdk
{{ .DiskType }}0:{{ .DiskUnit }}.present = "TRUE
{{ .DiskType }}0:{{ .DiskUnit }}.redo = "
`
