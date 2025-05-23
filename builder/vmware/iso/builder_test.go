// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package iso

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/common"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func testConfig() map[string]interface{} {
	return map[string]interface{}{
		"version":          21,
		"iso_checksum":     "md5:0B0F137F17AC10944716020B018F8126",
		"iso_url":          "http://www.packer.io",
		"shutdown_command": "foo",
		"ssh_username":     "foo",

		common.BuildNameConfigKey: "foo",
	}
}

func TestBuilder_ImplementsBuilder(t *testing.T) {
	var _ packersdk.Builder = &Builder{}
}

func TestBuilderPrepare_Defaults(t *testing.T) {
	var b Builder
	config := testConfig()
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}

	if b.config.DiskName != "disk" {
		t.Errorf("bad disk name: %s", b.config.DiskName)
	}

	if b.config.OutputDir != "output-foo" {
		t.Errorf("bad output dir: %s", b.config.OutputDir)
	}

	if b.config.Version < minimumHardwareVersion {
		t.Errorf("bad version: %d, minimum hardware version: %d", b.config.Version, minimumHardwareVersion)
	}

	if b.config.VMName != "packer-foo" {
		t.Errorf("bad vm name: %s", b.config.VMName)
	}
}

func TestBuilderPrepare_DiskSize(t *testing.T) {
	var b Builder
	config := testConfig()

	delete(config, "disk_size")
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("bad err: %s", err)
	}

	if b.config.DiskSize != 40000 {
		t.Fatalf("bad size: %d", b.config.DiskSize)
	}

	config["disk_size"] = 60000
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}

	if b.config.DiskSize != 60000 {
		t.Fatalf("bad size: %d", b.config.DiskSize)
	}
}

func TestBuilderPrepare_FloppyFiles(t *testing.T) {
	var b Builder
	config := testConfig()

	delete(config, "floppy_files")
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("bad err: %s", err)
	}

	if len(b.config.FloppyFiles) != 0 {
		t.Fatalf("bad: %#v", b.config.FloppyFiles)
	}

	floppiesPath := "testdata/floppies"
	config["floppy_files"] = []string{fmt.Sprintf("%s/bar.bat", floppiesPath), fmt.Sprintf("%s/foo.ps1", floppiesPath)}
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}

	expected := []string{fmt.Sprintf("%s/bar.bat", floppiesPath), fmt.Sprintf("%s/foo.ps1", floppiesPath)}
	if !reflect.DeepEqual(b.config.FloppyFiles, expected) {
		t.Fatalf("bad: %#v", b.config.FloppyFiles)
	}
}

func TestBuilderPrepare_InvalidFloppies(t *testing.T) {
	var b Builder
	config := testConfig()
	config["floppy_files"] = []string{"nonexistent.bat", "nonexistent.ps1"}

	b = Builder{}
	_, _, errs := b.Prepare(config)
	if errs == nil {
		t.Fatalf("Nonexistent floppies should trigger multierror")
	}

	if len(errs.(*packersdk.MultiError).Errors) != 2 {
		t.Fatalf("Multierror should work and report 2 errors")
	}
}

func TestBuilderPrepare_RemoteType(t *testing.T) {
	var b Builder
	config := testConfig()

	config["format"] = "ovf"
	config["remote_host"] = "esxi-01.example.com"
	config["remote_password"] = "VMware1!"
	config["skip_validate_credentials"] = true
	// Bad
	config["remote_type"] = "foobar"
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	config["remote_type"] = "esxi"
	// Bad
	config["remote_host"] = ""
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	// Good
	config["remote_type"] = ""
	config["format"] = ""
	config["remote_host"] = ""
	config["remote_password"] = ""
	config["remote_private_key_file"] = ""
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}

	// Good
	config["remote_type"] = "esxi"
	config["remote_host"] = "esxi-01.example.com"
	config["remote_password"] = "VMware1!"
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestBuilderPrepare_Export(t *testing.T) {
	type testCase struct {
		InputConfigVals         map[string]string
		ExpectedSkipExportValue bool
		ExpectedFormat          string
		ExpectedErr             bool
		Reason                  string
	}
	testCases := []testCase{
		{
			InputConfigVals: map[string]string{
				"remote_type": "",
				"format":      "",
			},
			ExpectedSkipExportValue: true,
			ExpectedFormat:          "vmx",
			ExpectedErr:             false,
			Reason:                  "should have defaulted format to vmx.",
		},
		{
			InputConfigVals: map[string]string{
				"remote_type":     "esxi",
				"format":          "",
				"remote_host":     "esxi-01.example.com",
				"remote_username": "root",
				"remote_password": "VMware1!",
			},
			ExpectedSkipExportValue: false,
			ExpectedFormat:          "ovf",
			ExpectedErr:             false,
			Reason:                  "should have defaulted format to ovf with remote set to esxi.",
		},
		{
			InputConfigVals: map[string]string{
				"remote_type": "esxi",
				"format":      "",
			},
			ExpectedSkipExportValue: false,
			ExpectedFormat:          "ovf",
			ExpectedErr:             true,
			Reason:                  "should have errored because remote host isn't set for remote build.",
		},
		{
			InputConfigVals: map[string]string{
				"remote_type":     "invalid",
				"format":          "",
				"remote_host":     "esxi-01.example.com",
				"remote_username": "root",
				"remote_password": "VMware1!",
			},
			ExpectedSkipExportValue: false,
			ExpectedFormat:          "ovf",
			ExpectedErr:             true,
			Reason:                  "should error with invalid remote type",
		},
		{
			InputConfigVals: map[string]string{
				"remote_type": "",
				"format":      "invalid",
			},
			ExpectedSkipExportValue: false,
			ExpectedFormat:          "invalid",
			ExpectedErr:             true,
			Reason:                  "should error with invalid format",
		},
		{
			InputConfigVals: map[string]string{
				"remote_type": "",
				"format":      "ova",
			},
			ExpectedSkipExportValue: false,
			ExpectedFormat:          "ova",
			ExpectedErr:             false,
			Reason:                  "should set user-given ova format",
		},
		{
			InputConfigVals: map[string]string{
				"remote_type":     "esxi",
				"format":          "ova",
				"remote_host":     "esxi-01.example.com",
				"remote_username": "root",
				"remote_password": "VMware1!",
			},
			ExpectedSkipExportValue: false,
			ExpectedFormat:          "ova",
			ExpectedErr:             false,
			Reason:                  "should set user-given ova format",
		},
	}
	for _, tc := range testCases {
		config := testConfig()
		for k, v := range tc.InputConfigVals {
			config[k] = v
		}
		config["skip_validate_credentials"] = true
		outCfg := &Config{}
		warns, errs := (outCfg).Prepare(config)

		if len(warns) > 0 {
			t.Fatalf("bad: %#v", warns)
		}

		if (errs != nil) != tc.ExpectedErr {
			t.Fatalf("received error: \n %s \n but 'expected err' was %t", errs, tc.ExpectedErr)
		}

		if outCfg.Format != tc.ExpectedFormat {
			t.Fatalf("Expected: %s. Actual: %s. Reason: %s", tc.ExpectedFormat,
				outCfg.Format, tc.Reason)
		}
		if outCfg.SkipExport != tc.ExpectedSkipExportValue {
			t.Fatalf("For SkipExport expected %t but received %t",
				tc.ExpectedSkipExportValue, outCfg.SkipExport)
		}
	}
}

func TestBuilderPrepare_RemoteExport(t *testing.T) {
	var b Builder
	config := testConfig()

	config["remote_type"] = "esxi"
	config["remote_host"] = "esxi-01.example.com"
	config["skip_validate_credentials"] = true
	// Bad
	config["remote_password"] = ""
	_, warns, err := b.Prepare(config)
	if len(warns) != 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	// Good
	config["remote_password"] = "VMware1!"
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) != 0 {
		t.Fatalf("err: %s", err)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestBuilderPrepare_Format(t *testing.T) {
	var b Builder
	config := testConfig()

	// Bad
	config["format"] = "foobar"
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	goodFormats := []string{"ova", "ovf", "vmx"}

	for _, format := range goodFormats {
		// Good
		config["format"] = format
		config["remote_type"] = "esxi"
		config["remote_host"] = "hosty.hostface"
		config["remote_password"] = "password"
		config["skip_validate_credentials"] = true

		b = Builder{}
		_, warns, err = b.Prepare(config)
		if len(warns) > 0 {
			t.Fatalf("bad: %#v", warns)
		}
		if err != nil {
			t.Fatalf("should not have error: %s", err)
		}
	}
}

func TestBuilderPrepare_InvalidKey(t *testing.T) {
	var b Builder
	config := testConfig()

	// Add a random key
	config["i_should_not_be_valid"] = true
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestBuilderPrepare_OutputDir(t *testing.T) {
	var b Builder
	config := testConfig()

	// Test with existing dir
	dir, err := os.MkdirTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(dir)

	config["output_directory"] = dir
	b = Builder{}
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Test with a good one
	config["output_directory"] = "i-hope-i-dont-exist"
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestBuilderPrepare_ToolsUploadPath(t *testing.T) {
	var b Builder
	config := testConfig()

	// Test a default
	delete(config, "tools_upload_path")
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if b.config.ToolsUploadPath == "" {
		t.Fatalf("bad value: %s", b.config.ToolsUploadPath)
	}

	// Test with a bad value
	config["tools_upload_path"] = "{{{nope}"
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	// Test with a good one
	config["tools_upload_path"] = "hey"
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestBuilderPrepare_VMXTemplatePath(t *testing.T) {
	var b Builder
	config := testConfig()

	// Test bad
	config["vmx_template_path"] = "/i/dont/exist/forreal"
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	// Test good
	tf, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Remove(tf.Name())
	defer tf.Close()

	if _, err := tf.Write([]byte("HELLO!")); err != nil {
		t.Fatalf("err: %s", err)
	}

	config["vmx_template_path"] = tf.Name()
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}

	// Bad template
	tf2, err := os.CreateTemp("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Remove(tf2.Name())
	defer tf2.Close()

	if _, err := tf2.Write([]byte("{{foo}")); err != nil {
		t.Fatalf("err: %s", err)
	}

	config["vmx_template_path"] = tf2.Name()
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}
}

func TestBuilderPrepare_VNCPort(t *testing.T) {
	var b Builder
	config := testConfig()

	// Bad
	config["vnc_port_min"] = 1000
	config["vnc_port_max"] = 500
	_, warns, err := b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	// Bad
	config["vnc_port_min"] = -500
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err == nil {
		t.Fatal("should have error")
	}

	// Good
	config["vnc_port_min"] = 500
	config["vnc_port_max"] = 1000
	b = Builder{}
	_, warns, err = b.Prepare(config)
	if len(warns) > 0 {
		t.Fatalf("bad: %#v", warns)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestBuilderCheckCollisions(t *testing.T) {
	config := testConfig()
	config["vmx_data"] = map[string]string{
		"no.collision":    "awesomesauce",
		"ide0:0.fileName": "is a collision",
		"displayName":     "also a collision",
	}
	{
		var b Builder
		_, warns, _ := b.Prepare(config)
		if len(warns) != 1 {
			t.Fatalf("Should have warning about two collisions.")
		}
	}
	{
		config["vmx_template_path"] = "some/path.vmx"
		var b Builder
		_, warns, _ := b.Prepare(config)
		if len(warns) != 0 {
			t.Fatalf("Should not check for collisions with custom template.")
		}
	}

}

func TestBuilderPrepare_CommConfig(t *testing.T) {
	// Test Winrm
	{
		config := testConfig()
		config["communicator"] = "winrm"
		config["winrm_username"] = "username"
		config["winrm_password"] = "password"
		config["winrm_host"] = "1.2.3.4"

		var b Builder
		_, warns, err := b.Prepare(config)
		if len(warns) > 0 {
			t.Fatalf("bad: %#v", warns)
		}
		if err != nil {
			t.Fatalf("should not have error: %s", err)
		}

		if b.config.Comm.WinRMUser != "username" {
			t.Errorf("bad winrm_username: %s", b.config.Comm.WinRMUser)
		}
		if b.config.Comm.WinRMPassword != "password" {
			t.Errorf("bad winrm_password: %s", b.config.Comm.WinRMPassword)
		}
		if host := b.config.Comm.Host(); host != "1.2.3.4" {
			t.Errorf("bad host: %s", host)
		}
	}

	// Test SSH
	{
		config := testConfig()
		config["communicator"] = "ssh"
		config["ssh_username"] = "username"
		config["ssh_password"] = "password"
		config["ssh_host"] = "1.2.3.4"

		var b Builder
		_, warns, err := b.Prepare(config)
		if len(warns) > 0 {
			t.Fatalf("bad: %#v", warns)
		}
		if err != nil {
			t.Fatalf("should not have error: %s", err)
		}

		if b.config.Comm.SSHUsername != "username" {
			t.Errorf("bad ssh_username: %s", b.config.Comm.SSHUsername)
		}
		if b.config.Comm.SSHPassword != "password" {
			t.Errorf("bad ssh_password: %s", b.config.Comm.SSHPassword)
		}
		if host := b.config.Comm.Host(); host != "1.2.3.4" {
			t.Errorf("bad host: %s", host)
		}
	}
}
