// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// Helper function to create a temporary file for testing
func createTempFile(t *testing.T) string {
	tmpFile, err := os.CreateTemp("", "tools_test_*.iso")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	return tmpFile.Name()
}

func TestToolsConfigPrepare_Empty(t *testing.T) {
	c := &ToolsConfig{}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) > 0 {
		t.Fatalf("err: %#v", errs)
	}

	if c.ToolsUploadPath != "{{ .Flavor }}.iso" {
		t.Fatal("should have defaulted tools upload path")
	}
}

func TestToolsConfigPrepare_SetUploadPath(t *testing.T) {
	c := &ToolsConfig{
		ToolsUploadPath: "path/to/tools.iso",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) > 0 {
		t.Fatalf("err: %#v", errs)
	}

	if c.ToolsUploadPath != "path/to/tools.iso" {
		t.Fatal("should have used given tools upload path")
	}
}

func TestToolsConfigPrepare_ErrorIfOnlySource(t *testing.T) {
	c := &ToolsConfig{
		ToolsSourcePath: "path/to/tools.iso",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Should have received an error because the flavor and " +
			"upload path aren't set")
	}
}

func TestToolsConfigPrepare_SourceSuccess(t *testing.T) {
	for _, c := range []*ToolsConfig{
		{
			ToolsSourcePath: "path/to/tools.iso",
			ToolsUploadPath: "partypath.iso",
		},
		{
			ToolsSourcePath:   "path/to/tools.iso",
			ToolsUploadFlavor: osLinux,
		},
	} {
		errs := c.Prepare(interpolate.NewContext())
		if len(errs) != 0 {
			t.Fatalf("Should not have received an error")
		}
	}
}

func TestToolsConfigPrepare_ToolsMode_Valid(t *testing.T) {
	testCases := []struct {
		name   string
		config *ToolsConfig
	}{
		{
			name: "upload mode",
			config: &ToolsConfig{
				ToolsMode: toolsModeUpload,
			},
		},
		{
			name: "attach mode with source path",
			config: &ToolsConfig{
				ToolsMode:       toolsModeAttach,
				ToolsSourcePath: createTempFile(t),
			},
		},
		{
			name: "attach mode with flavor",
			config: &ToolsConfig{
				ToolsMode:         toolsModeAttach,
				ToolsUploadFlavor: osLinux,
			},
		},
		{
			name: "disable mode",
			config: &ToolsConfig{
				ToolsMode: toolsModeDisable,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.config.Prepare(interpolate.NewContext())
			if len(errs) != 0 {
				t.Fatalf("Should not have received an error for %s: %v", tc.name, errs)
			}
		})
	}
}

func TestToolsConfigPrepare_ToolsMode_Invalid(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode: "invalid_mode",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Should have received exactly one error, got %d", len(errs))
	}

	if !strings.Contains(errs[0].Error(), "invalid 'tools_mode' specified") {
		t.Fatalf("Expected error about invalid tools_mode, got: %s", errs[0].Error())
	}
}

func TestToolsConfigPrepare_BackwardCompatibility(t *testing.T) {
	c := &ToolsConfig{
		ToolsUploadFlavor: osLinux,
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 0 {
		t.Fatalf("Should not have received an error: %v", errs)
	}

	if c.ToolsMode != toolsModeUpload {
		t.Fatalf("Expected tools_mode to default to 'upload', got: %s", c.ToolsMode)
	}
}

func TestToolsConfigPrepare_ConflictingConfiguration(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode:         toolsModeAttach,
		ToolsUploadFlavor: osLinux,
		ToolsSourcePath:   "path/to/tools.iso",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Should have received exactly one error, got %d", len(errs))
	}

	expectedError := "'tools_upload_flavor' and 'tools_source_path' cannot both be specified"
	if !strings.Contains(errs[0].Error(), expectedError) {
		t.Fatalf("Expected error about conflicting configuration, got: %s", errs[0].Error())
	}
}

func TestToolsConfigPrepare_AttachMode_MissingSource(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode: toolsModeAttach,
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Should have received exactly one error, got %d", len(errs))
	}

	expectedError := "'tools_source_path' or 'tools_upload_flavor' must be specified when 'tools_mode=\"attach\"'"
	if !strings.Contains(errs[0].Error(), expectedError) {
		t.Fatalf("Expected error about missing source, got: %s", errs[0].Error())
	}
}

func TestToolsConfigPrepare_DisableMode(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode: toolsModeDisable,
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 0 {
		t.Fatalf("Should not have received an error for disable mode: %v", errs)
	}
}

func TestToolsConfigPrepare_AttachMode_InvalidFlavor(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode:         toolsModeAttach,
		ToolsUploadFlavor: "invalid_flavor",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Should have received exactly one error, got %d", len(errs))
	}

	expectedError := "invalid 'tools_upload_flavor' specified: invalid_flavor"
	if !strings.Contains(errs[0].Error(), expectedError) {
		t.Fatalf("Expected error about invalid tools_upload_flavor, got: %s", errs[0].Error())
	}
}

func TestToolsConfigPrepare_AttachMode_ValidFlavors(t *testing.T) {
	validFlavors := []string{"darwin", "linux", "windows"}

	for _, flavor := range validFlavors {
		t.Run(fmt.Sprintf("flavor_%s", flavor), func(t *testing.T) {
			c := &ToolsConfig{
				ToolsMode:         toolsModeAttach,
				ToolsUploadFlavor: flavor,
			}

			errs := c.Prepare(interpolate.NewContext())
			if len(errs) != 0 {
				t.Fatalf("Should not have received an error for valid flavor %s: %v", flavor, errs)
			}
		})
	}
}

func TestToolsConfigPrepare_AttachMode_NonexistentSourcePath(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode:       toolsModeAttach,
		ToolsSourcePath: "/nonexistent/path/to/tools.iso",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Should have received exactly one error, got %d", len(errs))
	}

	expectedError := "tools source path does not exist"
	if !strings.Contains(errs[0].Error(), expectedError) {
		t.Fatalf("Expected error about nonexistent tools source path, got: %s", errs[0].Error())
	}
}

func TestToolsConfigPrepare_AttachMode_ExistingSourcePath(t *testing.T) {
	// Create a temporary file to test with
	tmpFile, err := os.CreateTemp("", "tools_test_*.iso")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	c := &ToolsConfig{
		ToolsMode:       toolsModeAttach,
		ToolsSourcePath: tmpFile.Name(),
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 0 {
		t.Fatalf("Should not have received an error for existing source path: %v", errs)
	}
}

func TestToolsConfigPrepare_UploadMode_InvalidFlavor(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode:         toolsModeUpload,
		ToolsUploadFlavor: "invalid_flavor",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Should have received exactly one error, got %d", len(errs))
	}

	expectedError := "invalid 'tools_upload_flavor' specified: invalid_flavor"
	if !strings.Contains(errs[0].Error(), expectedError) {
		t.Fatalf("Expected error about invalid tools_upload_flavor, got: %s", errs[0].Error())
	}
}

func TestToolsConfigPrepare_BackwardCompatibilityScenarios(t *testing.T) {
	testCases := []struct {
		name          string
		config        *ToolsConfig
		expectedMode  string
		expectedPath  string
		shouldError   bool
		errorContains string
	}{
		{
			name: "legacy config with tools_upload_flavor only",
			config: &ToolsConfig{
				ToolsUploadFlavor: "linux",
			},
			expectedMode: toolsModeUpload,
			expectedPath: "{{ .Flavor }}.iso",
			shouldError:  false,
		},
		{
			name: "legacy config with tools_upload_flavor and tools_upload_path",
			config: &ToolsConfig{
				ToolsUploadFlavor: "windows",
				ToolsUploadPath:   "custom-tools.iso",
			},
			expectedMode: toolsModeUpload,
			expectedPath: "custom-tools.iso",
			shouldError:  false,
		},
		{
			name: "legacy config with tools_source_path and tools_upload_flavor",
			config: &ToolsConfig{
				ToolsSourcePath:   "/path/to/tools.iso",
				ToolsUploadFlavor: "darwin",
			},
			expectedMode: toolsModeUpload,
			expectedPath: "{{ .Flavor }}.iso",
			shouldError:  false,
		},
		{
			name:         "no tools configuration should not default to upload mode",
			config:       &ToolsConfig{},
			expectedMode: "",
			expectedPath: "{{ .Flavor }}.iso",
			shouldError:  false,
		},
		{
			name: "explicit upload mode with flavor",
			config: &ToolsConfig{
				ToolsMode:         toolsModeUpload,
				ToolsUploadFlavor: "linux",
			},
			expectedMode: toolsModeUpload,
			expectedPath: "{{ .Flavor }}.iso",
			shouldError:  false,
		},
		{
			name: "explicit disable mode should not be affected by legacy config",
			config: &ToolsConfig{
				ToolsMode:         toolsModeDisable,
				ToolsUploadFlavor: "linux", // This should be ignored
			},
			expectedMode: toolsModeDisable,
			expectedPath: "",
			shouldError:  false,
		},
		{
			name: "explicit attach mode should not be affected by backward compatibility",
			config: &ToolsConfig{
				ToolsMode:         toolsModeAttach,
				ToolsUploadFlavor: "linux",
			},
			expectedMode: toolsModeAttach,
			expectedPath: "",
			shouldError:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.config.Prepare(interpolate.NewContext())

			if tc.shouldError {
				if len(errs) == 0 {
					t.Fatalf("Expected an error for %s, but got none", tc.name)
				}
				if tc.errorContains != "" {
					found := false
					for _, err := range errs {
						if strings.Contains(err.Error(), tc.errorContains) {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("Expected error containing '%s', but got: %v", tc.errorContains, errs)
					}
				}
				return
			}

			if len(errs) != 0 {
				t.Fatalf("Should not have received an error for %s: %v", tc.name, errs)
			}

			if tc.config.ToolsMode != tc.expectedMode {
				t.Fatalf("Expected tools_mode to be '%s', got: '%s'", tc.expectedMode, tc.config.ToolsMode)
			}

			if tc.expectedPath != "" && tc.config.ToolsUploadPath != tc.expectedPath {
				t.Fatalf("Expected tools_upload_path to be '%s', got: '%s'", tc.expectedPath, tc.config.ToolsUploadPath)
			}
		})
	}
}

func TestToolsConfigPrepare_MigrationScenarios(t *testing.T) {
	testCases := []struct {
		name        string
		description string
		oldConfig   *ToolsConfig
		newConfig   *ToolsConfig
	}{
		{
			name:        "migrate from upload flavor to attach mode",
			description: "User wants to migrate from upload to attach mode for better performance",
			oldConfig: &ToolsConfig{
				ToolsUploadFlavor: "linux",
			},
			newConfig: &ToolsConfig{
				ToolsMode:         toolsModeAttach,
				ToolsUploadFlavor: "linux",
			},
		},
		{
			name:        "migrate from upload flavor to attach with custom source",
			description: "User wants to use a custom tools ISO with attach mode",
			oldConfig: &ToolsConfig{
				ToolsUploadFlavor: "windows",
			},
			newConfig: &ToolsConfig{
				ToolsMode:       toolsModeAttach,
				ToolsSourcePath: createTempFile(t),
			},
		},
		{
			name:        "migrate from upload to disable",
			description: "User wants to disable tools entirely",
			oldConfig: &ToolsConfig{
				ToolsUploadFlavor: "darwin",
			},
			newConfig: &ToolsConfig{
				ToolsMode: toolsModeDisable,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that old config still works (backward compatibility)
			errs := tc.oldConfig.Prepare(interpolate.NewContext())
			if len(errs) != 0 {
				t.Fatalf("Old config should still work: %v", errs)
			}

			// Test that new config works (migration path)
			errs = tc.newConfig.Prepare(interpolate.NewContext())
			if len(errs) != 0 {
				t.Fatalf("New config should work: %v", errs)
			}

			t.Logf("Migration scenario '%s': %s", tc.name, tc.description)
		})
	}
}

func TestToolsConfigPrepare_ValidationErrorMessages(t *testing.T) {
	testCases := []struct {
		name          string
		config        *ToolsConfig
		expectedError string
	}{
		{
			name: "invalid tools_mode",
			config: &ToolsConfig{
				ToolsMode: "invalid_mode",
			},
			expectedError: "invalid 'tools_mode' specified: invalid_mode; must be one of upload, attach, disable",
		},
		{
			name: "conflicting attach configuration",
			config: &ToolsConfig{
				ToolsMode:         toolsModeAttach,
				ToolsUploadFlavor: "linux",
				ToolsSourcePath:   "/path/to/tools.iso",
			},
			expectedError: "'tools_upload_flavor' and 'tools_source_path' cannot both be specified with 'tools_mode=\"attach\"'",
		},
		{
			name: "attach mode missing source",
			config: &ToolsConfig{
				ToolsMode: toolsModeAttach,
			},
			expectedError: "'tools_source_path' or 'tools_upload_flavor' must be specified when 'tools_mode=\"attach\"'",
		},
		{
			name: "invalid flavor in attach mode",
			config: &ToolsConfig{
				ToolsMode:         toolsModeAttach,
				ToolsUploadFlavor: "invalid",
			},
			expectedError: "invalid 'tools_upload_flavor' specified: invalid; must be one of darwin, linux, windows",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.config.Prepare(interpolate.NewContext())
			if len(errs) == 0 {
				t.Fatalf("Expected an error for %s, but got none", tc.name)
			}

			found := false
			for _, err := range errs {
				if strings.Contains(err.Error(), tc.expectedError) {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected error containing '%s', but got: %v", tc.expectedError, errs)
			}
		})
	}
}
