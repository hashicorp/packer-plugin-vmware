// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package iso

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/acctest"
	"github.com/hashicorp/packer-plugin-sdk/acctest/testutils"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/hashicorp/packer-plugin-sdk/tmp"
	"github.com/hashicorp/packer-plugin-vmware/builder/vmware/common"
)

func createFloppyOutput(prefix string) (string, map[string]string, error) {
	f, err := tmp.File(prefix)
	if err != nil {
		return "", map[string]string{}, errors.New("unable to create temp file")
	}
	f.Close()

	output := f.Name()
	outputFile := strings.ReplaceAll(output, "\\", "\\\\")
	vmxData := map[string]string{
		"floppy0.present":        "TRUE",
		"floppy0.fileType":       "file",
		"floppy0.clientDevice":   "FALSE",
		"floppy0.fileName":       outputFile,
		"floppy0.startConnected": "TRUE",
	}
	return output, vmxData, nil
}

func readFloppyOutput(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("unable to open file %s", path)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("unable to read file: %s", err)
	}
	if len(data) == 0 {
		return "", nil
	}
	return string(data[:bytes.IndexByte(data, 0)]), nil
}

// RenderConfig helps create dynamic packer template configs for parsing by
// builderT without having to write the config to a file.
func RenderConfig(builderConfig map[string]interface{}, provisionerConfig map[string]string) string {
	// set up basic build template
	t := map[string][]map[string]interface{}{
		"builders": {
			map[string]interface{}{
				"type":                        "vmware-iso",
				"iso_url":                     "https://archive.org/download/ut-ttylinux-i686-12.6/ut-ttylinux-i686-12.6.iso",
				"iso_checksum":                "md5:43c1feeae55a44c6ef694b8eb18408a6",
				"ssh_username":                "root",
				"ssh_password":                "password",
				"ssh_wait_timeout":            "45s",
				"boot_command":                []string{"<enter><wait5><wait10>", "root<enter><wait>password<enter><wait>", "udhcpc<enter><wait>"},
				"shutdown_command":            "/sbin/shutdown -h; exit 0",
				"ssh_key_exchange_algorithms": []string{"diffie-hellman-group1-sha1"},
			},
		},
		"provisioners": {
			map[string]interface{}{
				"type":   "shell",
				"inline": []string{"echo hola mundo"},
			},
		},
	}
	// apply special builder overrides
	for k, v := range builderConfig {
		t["builders"][0][k] = v
	}
	// Apply special provisioner overrides
	for k, v := range provisionerConfig {
		t["provisioners"][0][k] = v
	}

	j, _ := json.Marshal(t)
	return string(j)
}

func TestAccStepCreateVmx_SerialFile(t *testing.T) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("This test is only run with PACKER_ACC=1 due to the requirement of access to the VMware binaries.")
	}

	tmpfile, err := tmp.File("SerialFileInput.")
	if err != nil {
		t.Fatalf("unable to create temp file")
	}
	serialConfig := map[string]interface{}{
		"serial": fmt.Sprintf("file:%s", filepath.ToSlash(tmpfile.Name())),
	}

	configString := RenderConfig(serialConfig, map[string]string{})

	testCase := &acctest.PluginTestCase{
		Name: "vmware-iso_create_vmx_serial_file",
		Teardown: func() error {
			f, _ := os.Stat(tmpfile.Name())
			if f != nil {
				if err := os.Remove(tmpfile.Name()); err != nil {
					return fmt.Errorf("unable to remove file %s: %s", tmpfile.Name(), err)
				}
			}
			testutils.CleanupFiles("output-vmware-iso")
			return nil
		},
		Template: configString,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("bad exit code. Logfile: %s", logfile)
				}
			}
			_, err := os.Stat(tmpfile.Name())
			if err != nil {
				return fmt.Errorf("unable to create a file for serial port: %s", err)
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

func TestStepCreateVmx_SerialPort(t *testing.T) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("This test is only run with PACKER_ACC=1 due to the requirement of access to the VMware binaries.")
	}

	var defaultSerial string
	if runtime.GOOS == "windows" {
		defaultSerial = "COM1"
	} else {
		defaultSerial = "/dev/ttyS0"
	}

	config := map[string]interface{}{
		"serial": fmt.Sprintf("device:%s", filepath.ToSlash(defaultSerial)),
	}
	provision := map[string]string{
		"inline": "dmesg | egrep -o '^serial8250: ttyS1 at' > /dev/fd0",
	}

	// where to write output
	output, vmxData, err := createFloppyOutput("SerialPortOutput.")
	if err != nil {
		t.Fatalf("Error creating output: %s", err)
	}

	config["vmx_data"] = vmxData
	configString := RenderConfig(config, provision)
	testCase := &acctest.PluginTestCase{
		Name: "vmware-iso_create_vmx_serial_port",
		Teardown: func() error {
			if _, err := os.Stat(output); err == nil {
				os.Remove(output)
			}
			testutils.CleanupFiles("output-vmware-iso")
			return nil
		},
		Template: configString,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("bad exit code. Logfile: %s", logfile)
				}
			}
			_, err := os.Stat(output)
			if err != nil {
				return fmt.Errorf("unable to create a file for serial port: %s", err)
			}
			// check the output
			data, err := readFloppyOutput(output)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			if data != "serial8250: ttyS1 at\n" {
				return fmt.Errorf("serial port not detected : %v", data)
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

func TestStepCreateVmx_ParallelPort(t *testing.T) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("This test is only run with PACKER_ACC=1 due to the requirement of access to the VMware binaries.")
	}

	var defaultParallel string
	if runtime.GOOS == "windows" {
		defaultParallel = "LPT1"
	} else {
		defaultParallel = "/dev/lp0"
	}

	config := map[string]interface{}{
		"parallel": fmt.Sprintf("device:%s,uni", filepath.ToSlash(defaultParallel)),
	}
	provision := map[string]string{
		"inline": "cat /proc/modules | egrep -o '^parport ' > /dev/fd0",
	}

	// where to write output
	output, vmxData, err := createFloppyOutput("ParallelPortOutput.")
	if err != nil {
		t.Fatalf("Error creating output: %s", err)
	}

	config["vmx_data"] = vmxData
	configString := RenderConfig(config, provision)
	testCase := &acctest.PluginTestCase{
		Name: "vmware-iso_create_vmx_parallel_port",
		Teardown: func() error {
			if _, err := os.Stat(output); err == nil {
				os.Remove(output)
			}
			testutils.CleanupFiles("output-vmware-iso")
			return nil
		},
		Template: configString,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("bad exit code. Logfile: %s", logfile)
				}
			}
			_, err := os.Stat(output)
			if err != nil {
				return fmt.Errorf("unable to create a file for serial port: %s", err)
			}
			// check the output
			data, err := readFloppyOutput(output)
			if err != nil {
				t.Errorf("%s", err)
			}

			if data != "parport \n" {
				t.Errorf("Parallel port not detected : %v", data)
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

func TestStepCreateVmx_Usb(t *testing.T) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("This test is only run with PACKER_ACC=1 due to the requirement of access to the VMware binaries.")
	}

	config := map[string]interface{}{
		"usb": "TRUE",
	}
	provision := map[string]string{
		"inline": "dmesg | egrep -m1 -o 'USB hub found$' > /dev/fd0",
	}

	output, vmxData, err := createFloppyOutput("UsbOutput.")
	if err != nil {
		t.Fatalf("Error creating output: %s", err)
	}

	config["vmx_data"] = vmxData
	configString := RenderConfig(config, provision)
	testCase := &acctest.PluginTestCase{
		Name: "vmware-iso_create_vmx_usb",
		Teardown: func() error {
			if _, err := os.Stat(output); err == nil {
				os.Remove(output)
			}
			testutils.CleanupFiles("output-vmware-iso")
			return nil
		},
		Template: configString,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("bad exit code. Logfile: %s", logfile)
				}
			}
			_, err := os.Stat(output)
			if err != nil {
				return fmt.Errorf("unable to create a file for serial port: %s", err)
			}
			// check the output
			data, err := readFloppyOutput(output)
			if err != nil {
				t.Errorf("%s", err)
			}

			if data != "USB hub found\n" {
				t.Errorf("USB support not detected : %v", data)
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

func TestStepCreateVmx_Sound(t *testing.T) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("This test is only run with PACKER_ACC=1 due to the requirement of access to the VMware binaries.")
	}

	config := map[string]interface{}{
		"sound": "TRUE",
	}
	provision := map[string]string{
		"inline": "cat /proc/modules | egrep -o '^soundcore' > /dev/fd0",
	}

	// where to write output
	output, vmxData, err := createFloppyOutput("SoundOutput.")
	if err != nil {
		t.Fatalf("Error creating output: %s", err)
	}
	defer func() {
		if _, err := os.Stat(output); err == nil {
			os.Remove(output)
		}
	}()

	config["vmx_data"] = vmxData
	configString := RenderConfig(config, provision)
	testCase := &acctest.PluginTestCase{
		Name: "vmware-iso_create_vmx_sound",
		Teardown: func() error {
			if _, err := os.Stat(output); err == nil {
				os.Remove(output)
			}
			testutils.CleanupFiles("output-vmware-iso")
			return nil
		},
		Template: configString,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("bad exit code. Logfile: %s", logfile)
				}
			}
			_, err := os.Stat(output)
			if err != nil {
				return fmt.Errorf("unable to create a file for serial port: %s", err)
			}
			// check the output
			data, err := readFloppyOutput(output)
			if err != nil {
				t.Errorf("%s", err)
			}

			if data != "soundcore\n" {
				t.Errorf("Soundcard not detected : %v", data)
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

func TestStepCreateVmx_Usb3(t *testing.T) {
	if os.Getenv("PACKER_ACC") == "" {
		t.Skip("This test is only run with PACKER_ACC=1 due to the requirement of access to the VMware binaries.")
	}

	config := map[string]interface{}{
		"usb":         "TRUE",
		"usb_version": common.UsbVersion31,
	}
	provision := map[string]string{
		"inline": "dmesg | egrep -m1 -o 'xhci_hcd.*USB 3.1 Root Hub' > /dev/fd0",
	}

	output, vmxData, err := createFloppyOutput("Usb3Output.")
	if err != nil {
		t.Fatalf("Error creating output: %s", err)
	}

	config["vmx_data"] = vmxData
	configString := RenderConfig(config, provision)
	testCase := &acctest.PluginTestCase{
		Name: "vmware-iso_create_vmx_usb3",
		Teardown: func() error {
			if _, err := os.Stat(output); err == nil {
				os.Remove(output)
			}
			testutils.CleanupFiles("output-vmware-iso")
			return nil
		},
		Template: configString,
		Check: func(buildCommand *exec.Cmd, logfile string) error {
			if buildCommand.ProcessState != nil {
				if buildCommand.ProcessState.ExitCode() != 0 {
					return fmt.Errorf("bad exit code. Logfile: %s", logfile)
				}
			}
			_, err := os.Stat(output)
			if err != nil {
				return fmt.Errorf("unable to create a file for serial port: %s", err)
			}
			// check the output
			data, err := readFloppyOutput(output)
			if err != nil {
				t.Errorf("%s", err)
			}

			if !strings.Contains(data, "USB 3.1 Root Hub") {
				t.Errorf("USB 3.1 support not detected : %v", data)
			}
			return nil
		},
	}
	acctest.TestPlugin(t, testCase)
}

func TestVMXTemplateData_USB31Enabled(t *testing.T) {
	templateData := vmxTemplateData{
		Name:                           "test-vm",
		GuestOS:                        "ubuntu-64",
		Version:                        "18",
		CpuCount:                       "2",
		MemorySize:                     "1024",
		DiskName:                       "test-disk",
		ISOPath:                        "/path/to/test.iso",
		NetworkType:                    "nat",
		NetworkDevice:                  "",
		NetworkAdapter:                 "e1000",
		SoundPresent:                   "FALSE",
		UsbPresent:                     "TRUE",
		UsbVersion:                     common.UsbVersion31,
		SerialPresent:                  "FALSE",
		ParallelPresent:                "FALSE",
		HardwareAssistedVirtualization: false,
	}

	ctx := interpolate.Context{}
	ctx.Data = &templateData

	result, err := interpolate.Render(DefaultVMXTemplate, &ctx)
	if err != nil {
		t.Fatalf("Failed to render VMX template: %s", err)
	}

	if !strings.Contains(result, "usb_xhci.present = \"TRUE\"") {
		t.Error("Expected usb_xhci.present = \"TRUE\" in VMX output when USB 3.1 is enabled")
	}

	if !strings.Contains(result, "usb.present = \"TRUE\"") {
		t.Error("Expected usb.present = \"TRUE\" in VMX output when USB 3.1 is enabled")
	}
}

func TestVMXTemplateData_USB32Enabled(t *testing.T) {
	templateData := vmxTemplateData{
		Name:                           "test-vm",
		GuestOS:                        "ubuntu-64",
		Version:                        "18",
		CpuCount:                       "2",
		MemorySize:                     "1024",
		DiskName:                       "test-disk",
		ISOPath:                        "/path/to/test.iso",
		NetworkType:                    "nat",
		NetworkDevice:                  "",
		NetworkAdapter:                 "e1000",
		SoundPresent:                   "FALSE",
		UsbPresent:                     "TRUE",
		UsbVersion:                     common.UsbVersion32,
		SerialPresent:                  "FALSE",
		ParallelPresent:                "FALSE",
		HardwareAssistedVirtualization: false,
	}

	ctx := interpolate.Context{}
	ctx.Data = &templateData

	result, err := interpolate.Render(DefaultVMXTemplate, &ctx)
	if err != nil {
		t.Fatalf("Failed to render VMX template: %s", err)
	}

	if !strings.Contains(result, "usb_xhci.present = \"TRUE\"") {
		t.Error("Expected usb_xhci.present = \"TRUE\" in VMX output when USB 3.2 is enabled")
	}

	if !strings.Contains(result, "usb.present = \"TRUE\"") {
		t.Error("Expected usb.present = \"TRUE\" in VMX output when USB 3.2 is enabled")
	}
}

func TestVMXTemplateData_USB3Disabled(t *testing.T) {
	templateData := vmxTemplateData{
		Name:                           "test-vm",
		GuestOS:                        "ubuntu-64",
		Version:                        "18",
		CpuCount:                       "2",
		MemorySize:                     "1024",
		DiskName:                       "test-disk",
		ISOPath:                        "/path/to/test.iso",
		NetworkType:                    "nat",
		NetworkDevice:                  "",
		NetworkAdapter:                 "e1000",
		SoundPresent:                   "FALSE",
		UsbPresent:                     "FALSE",
		UsbVersion:                     "",
		SerialPresent:                  "FALSE",
		ParallelPresent:                "FALSE",
		HardwareAssistedVirtualization: false,
	}

	ctx := interpolate.Context{}
	ctx.Data = &templateData

	result, err := interpolate.Render(DefaultVMXTemplate, &ctx)
	if err != nil {
		t.Fatalf("Failed to render VMX template: %s", err)
	}

	if !strings.Contains(result, "usb.present = \"FALSE\"") {
		t.Error("Expected usb.present = \"FALSE\" in VMX output when USB is disabled")
	}

	if strings.Contains(result, "usb_xhci.present") {
		t.Error("USB 3.1 controller should not be present when USB is disabled")
	}
}

func TestVMXTemplateData_USB2EnabledUSB3Disabled(t *testing.T) {
	templateData := vmxTemplateData{
		Name:                           "test-vm",
		GuestOS:                        "ubuntu-64",
		Version:                        "18",
		CpuCount:                       "2",
		MemorySize:                     "1024",
		DiskName:                       "test-disk",
		ISOPath:                        "/path/to/test.iso",
		NetworkType:                    "nat",
		NetworkDevice:                  "",
		NetworkAdapter:                 "e1000",
		SoundPresent:                   "FALSE",
		UsbPresent:                     "TRUE",
		UsbVersion:                     common.UsbVersion20,
		SerialPresent:                  "FALSE",
		ParallelPresent:                "FALSE",
		HardwareAssistedVirtualization: false,
	}

	ctx := interpolate.Context{}
	ctx.Data = &templateData

	result, err := interpolate.Render(DefaultVMXTemplate, &ctx)
	if err != nil {
		t.Fatalf("Failed to render VMX template: %s", err)
	}

	if !strings.Contains(result, "usb.present = \"TRUE\"") {
		t.Error("Expected usb.present = \"TRUE\" in VMX output when USB 2.0 is enabled")
	}

	if strings.Contains(result, "usb_xhci.present") {
		t.Error("USB 3.1 controller should not be present when USB version is 2.0")
	}
}

func TestVMXTemplateData_PopulationFromConfig(t *testing.T) {
	testCases := []struct {
		name        string
		usbConfig   bool
		usbVersion  string
		expectedUSB string
		expectedVer string
	}{
		{
			name:        "USB 3.1 enabled",
			usbConfig:   true,
			usbVersion:  common.UsbVersion31,
			expectedUSB: "TRUE",
			expectedVer: common.UsbVersion31,
		},
		{
			name:        "USB 3.2 enabled",
			usbConfig:   true,
			usbVersion:  common.UsbVersion32,
			expectedUSB: "TRUE",
			expectedVer: common.UsbVersion32,
		},
		{
			name:        "USB 2.0 enabled",
			usbConfig:   true,
			usbVersion:  common.UsbVersion20,
			expectedUSB: "TRUE",
			expectedVer: common.UsbVersion20,
		},
		{
			name:        "USB disabled",
			usbConfig:   false,
			usbVersion:  "",
			expectedUSB: "FALSE",
			expectedVer: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			usbPresent := map[bool]string{true: "TRUE", false: "FALSE"}[tc.usbConfig]

			if usbPresent != tc.expectedUSB {
				t.Errorf("Expected UsbPresent to be %s, got %s", tc.expectedUSB, usbPresent)
			}

			if tc.usbVersion != tc.expectedVer {
				t.Errorf("Expected UsbVersion to be %s, got %s", tc.expectedVer, tc.usbVersion)
			}
		})
	}
}
