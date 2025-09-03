// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// Test that empty config works (no tools configuration)
func TestToolsConfigPrepare_Empty(t *testing.T) {
	c := &ToolsConfig{}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) > 0 {
		t.Fatalf("Empty config should not error: %v", errs)
	}

	// Empty config should not set any defaults
	if c.ToolsMode != "" {
		t.Fatalf("Empty config should not set tools_mode, got: %s", c.ToolsMode)
	}
	if c.ToolsUploadPath != "" {
		t.Fatalf("Empty config should not set tools_upload_path, got: %s", c.ToolsUploadPath)
	}
}

// Test valid configurations for each mode
func TestToolsConfigPrepare_ValidConfigurations(t *testing.T) {
	testCases := []struct {
		name   string
		config *ToolsConfig
	}{
		{
			name: "upload mode with flavor",
			config: &ToolsConfig{
				ToolsMode:         toolsModeUpload,
				ToolsUploadFlavor: "linux",
			},
		},
		{
			name: "upload mode with flavor and custom path",
			config: &ToolsConfig{
				ToolsMode:         toolsModeUpload,
				ToolsUploadFlavor: "windows",
				ToolsUploadPath:   "custom-tools.iso",
			},
		},
		{
			name: "upload mode with custom source path",
			config: &ToolsConfig{
				ToolsMode:       toolsModeUpload,
				ToolsSourcePath: "testdata/tools.iso",
			},
		},
		{
			name: "upload mode with custom source path and upload path",
			config: &ToolsConfig{
				ToolsMode:       toolsModeUpload,
				ToolsSourcePath: "testdata/tools.iso",
				ToolsUploadPath: "/tmp/custom-tools.iso",
			},
		},
		{
			name: "attach mode with source path",
			config: &ToolsConfig{
				ToolsMode:       toolsModeAttach,
				ToolsSourcePath: "testdata/tools.iso",
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
				t.Fatalf("Valid config should not error: %v", errs)
			}
		})
	}
}

// Test that upload mode sets default upload path
func TestToolsConfigPrepare_UploadModeDefaults(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode:         toolsModeUpload,
		ToolsUploadFlavor: "linux",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 0 {
		t.Fatalf("Should not error: %v", errs)
	}

	expectedPath := "{{ .Flavor }}.iso"
	if c.ToolsUploadPath != expectedPath {
		t.Fatalf("Expected upload path '%s', got '%s'", expectedPath, c.ToolsUploadPath)
	}
}

// Test required field validation
func TestToolsConfigPrepare_RequiredFields(t *testing.T) {
	testCases := []struct {
		name          string
		config        *ToolsConfig
		expectedError string
	}{
		{
			name: "upload mode missing both flavor and source path",
			config: &ToolsConfig{
				ToolsMode: toolsModeUpload,
			},
			expectedError: "'tools_mode=\"upload\"' requires either 'tools_upload_flavor' or 'tools_source_path'",
		},
		{
			name: "attach mode missing source path",
			config: &ToolsConfig{
				ToolsMode: toolsModeAttach,
			},
			expectedError: "'tools_source_path' is required when 'tools_mode=\"attach\"'",
		},
		{
			name: "tools config without explicit mode",
			config: &ToolsConfig{
				ToolsUploadFlavor: "linux",
			},
			expectedError: "'tools_mode' must be explicitly specified",
		},
		{
			name: "source path without explicit mode",
			config: &ToolsConfig{
				ToolsSourcePath: "/path/to/tools.iso",
			},
			expectedError: "'tools_mode' must be explicitly specified",
		},
		{
			name: "upload path without explicit mode",
			config: &ToolsConfig{
				ToolsUploadPath: "custom-tools.iso",
			},
			expectedError: "'tools_mode' must be explicitly specified when using any tools configuration",
		},
		{
			name: "upload path with attach mode",
			config: &ToolsConfig{
				ToolsMode:       toolsModeAttach,
				ToolsSourcePath: "testdata/tools.iso",
				ToolsUploadPath: "custom-tools.iso",
			},
			expectedError: "'tools_upload_path' can only be used with 'tools_mode=\"upload\"', not 'tools_mode=\"attach\"'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.config.Prepare(interpolate.NewContext())
			if len(errs) != 1 {
				t.Fatalf("Expected exactly one error, got %d: %v", len(errs), errs)
			}

			if !strings.Contains(errs[0].Error(), tc.expectedError) {
				t.Fatalf("Expected error containing '%s', got: %s", tc.expectedError, errs[0].Error())
			}
		})
	}
}

// Test mutual exclusivity validation
func TestToolsConfigPrepare_MutualExclusivity(t *testing.T) {
	c := &ToolsConfig{
		ToolsMode:         toolsModeUpload,
		ToolsUploadFlavor: "linux",
		ToolsSourcePath:   "testdata/tools.iso",
	}

	errs := c.Prepare(interpolate.NewContext())
	if len(errs) != 1 {
		t.Fatalf("Expected exactly one error, got %d: %v", len(errs), errs)
	}

	expectedError := "'tools_upload_flavor' and 'tools_source_path' cannot both be specified"
	if !strings.Contains(errs[0].Error(), expectedError) {
		t.Fatalf("Expected error about mutual exclusivity, got: %s", errs[0].Error())
	}
}

// Test invalid values
func TestToolsConfigPrepare_InvalidValues(t *testing.T) {
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
			expectedError: "invalid 'tools_mode' specified: invalid_mode",
		},
		{
			name: "invalid tools_upload_flavor",
			config: &ToolsConfig{
				ToolsMode:         toolsModeUpload,
				ToolsUploadFlavor: "invalid_flavor",
			},
			expectedError: "invalid 'tools_upload_flavor' specified: invalid_flavor",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.config.Prepare(interpolate.NewContext())
			if len(errs) != 1 {
				t.Fatalf("Expected exactly one error, got %d: %v", len(errs), errs)
			}

			if !strings.Contains(errs[0].Error(), tc.expectedError) {
				t.Fatalf("Expected error containing '%s', got: %s", tc.expectedError, errs[0].Error())
			}
		})
	}
}

// Test file existence validation for attach mode
func TestToolsConfigPrepare_AttachModeFileValidation(t *testing.T) {
	t.Run("nonexistent source path", func(t *testing.T) {
		c := &ToolsConfig{
			ToolsMode:       toolsModeAttach,
			ToolsSourcePath: "/nonexistent/path/tools.iso",
		}

		errs := c.Prepare(interpolate.NewContext())
		if len(errs) != 1 {
			t.Fatalf("Expected exactly one error, got %d: %v", len(errs), errs)
		}

		expectedError := "'tools_source_path' does not exist"
		if !strings.Contains(errs[0].Error(), expectedError) {
			t.Fatalf("Expected error about nonexistent file, got: %s", errs[0].Error())
		}
	})

	t.Run("existing source path", func(t *testing.T) {
		c := &ToolsConfig{
			ToolsMode:       toolsModeAttach,
			ToolsSourcePath: "testdata/tools.iso",
		}

		errs := c.Prepare(interpolate.NewContext())
		if len(errs) != 0 {
			t.Fatalf("Should not error for existing file: %v", errs)
		}
	})
}

// Test valid flavors
func TestToolsConfigPrepare_ValidFlavors(t *testing.T) {
	validFlavors := []string{"darwin", "linux", "windows"}

	for _, flavor := range validFlavors {
		t.Run("flavor_"+flavor, func(t *testing.T) {
			c := &ToolsConfig{
				ToolsMode:         toolsModeUpload,
				ToolsUploadFlavor: flavor,
			}

			errs := c.Prepare(interpolate.NewContext())
			if len(errs) != 0 {
				t.Fatalf("Should not error for valid flavor %s: %v", flavor, errs)
			}
		})
	}
}

// Test disable mode
func TestToolsConfigPrepare_DisableMode(t *testing.T) {
	t.Run("basic disable mode", func(t *testing.T) {
		c := &ToolsConfig{
			ToolsMode: toolsModeDisable,
		}

		errs := c.Prepare(interpolate.NewContext())
		if len(errs) != 0 {
			t.Fatalf("Disable mode should not error: %v", errs)
		}
	})

	t.Run("disable mode with tools_upload_path should not error", func(t *testing.T) {
		c := &ToolsConfig{
			ToolsMode:       toolsModeDisable,
			ToolsUploadPath: "/tmp/tools.iso",
		}

		errs := c.Prepare(interpolate.NewContext())
		if len(errs) != 0 {
			t.Fatalf("Disable mode with tools_upload_path should not error (should just log warning): %v", errs)
		}
	})

	t.Run("disable mode with all tools config should not error", func(t *testing.T) {
		c := &ToolsConfig{
			ToolsMode:         toolsModeDisable,
			ToolsUploadPath:   "/tmp/tools.iso",
			ToolsUploadFlavor: "linux",
			ToolsSourcePath:   "testdata/tools.iso",
		}

		errs := c.Prepare(interpolate.NewContext())
		if len(errs) != 0 {
			t.Fatalf("Disable mode with all tools config should not error (should just log warnings): %v", errs)
		}
	})
}

// TestToolsConfigPrepare_ModeSpecificFields validates that configuration fields
// can only be used with their appropriate tools_mode values.
func TestToolsConfigPrepare_ModeSpecificFields(t *testing.T) {
	testCases := []struct {
		name          string
		config        *ToolsConfig
		expectedError string
	}{
		{
			name: "upload_flavor with attach mode",
			config: &ToolsConfig{
				ToolsMode:         toolsModeAttach,
				ToolsUploadFlavor: "linux",
			},
			expectedError: "'tools_upload_flavor' can only be used with 'tools_mode=\"upload\"', not 'tools_mode=\"attach\"'",
		},

		{
			name: "upload_path with attach mode",
			config: &ToolsConfig{
				ToolsMode:       toolsModeAttach,
				ToolsUploadPath: "/path/to/upload",
			},
			expectedError: "'tools_upload_path' can only be used with 'tools_mode=\"upload\"', not 'tools_mode=\"attach\"'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.config.Prepare(interpolate.NewContext())
			if len(errs) == 0 {
				t.Fatalf("Expected at least one error, got none")
			}

			// Check that the expected error is present in the error list
			found := false
			for _, err := range errs {
				if strings.Contains(err.Error(), tc.expectedError) {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected error containing '%s', got errors: %v", tc.expectedError, errs)
			}
		})
	}
}
