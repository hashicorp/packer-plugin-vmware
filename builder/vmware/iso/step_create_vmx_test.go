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
	"github.com/hashicorp/packer-plugin-sdk/tmp"
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
