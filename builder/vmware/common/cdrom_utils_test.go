// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestFindNextAvailableCDROMSlot(t *testing.T) {
	tests := []struct {
		name        string
		vmxData     map[string]string
		adapterType string
		expected    string
		expectError bool
	}{
		{
			name:        "empty VMX data - IDE adapter",
			vmxData:     map[string]string{},
			adapterType: "ide",
			expected:    "ide0:0",
			expectError: false,
		},
		{
			name:        "empty VMX data - SATA adapter",
			vmxData:     map[string]string{},
			adapterType: "sata",
			expected:    "sata0:0",
			expectError: false,
		},
		{
			name:        "empty VMX data - SCSI adapter",
			vmxData:     map[string]string{},
			adapterType: "scsi",
			expected:    "scsi0:0",
			expectError: false,
		},
		{
			name: "IDE slot 0 occupied - should return slot 1",
			vmxData: map[string]string{
				"ide0:0.present": "TRUE",
			},
			adapterType: "ide",
			expected:    "ide0:1",
			expectError: false,
		},
		{
			name: "SCSI slots 0-6 occupied - should skip slot 7 and return slot 8",
			vmxData: map[string]string{
				"scsi0:0.present": "TRUE",
				"scsi0:1.present": "TRUE",
				"scsi0:2.present": "TRUE",
				"scsi0:3.present": "TRUE",
				"scsi0:4.present": "TRUE",
				"scsi0:5.present": "TRUE",
				"scsi0:6.present": "TRUE",
			},
			adapterType: "scsi",
			expected:    "scsi0:8",
			expectError: false,
		},
		{
			name: "multiple buses - should use next bus when current is full",
			vmxData: map[string]string{
				"ide0:0.present": "TRUE",
				"ide0:1.present": "TRUE",
			},
			adapterType: "ide",
			expected:    "ide1:0",
			expectError: false,
		},
		{
			name: "mixed adapter types - should only consider specified type",
			vmxData: map[string]string{
				"ide0:0.present":  "TRUE",
				"sata0:0.present": "TRUE",
				"scsi0:0.present": "TRUE",
			},
			adapterType: "ide",
			expected:    "ide0:1",
			expectError: false,
		},
		{
			name:        "empty adapter type - should return error",
			vmxData:     map[string]string{},
			adapterType: "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid adapter type - should return error",
			vmxData:     map[string]string{},
			adapterType: "invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "case insensitive adapter type",
			vmxData:     map[string]string{},
			adapterType: "IDE",
			expected:    "ide0:0",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindNextAvailableCDROMSlot(tt.vmxData, tt.adapterType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAttachCDROMDevice(t *testing.T) {
	tests := []struct {
		name         string
		vmxData      map[string]string
		devicePath   string
		isoPath      string
		adapterType  string
		expectError  bool
		expectedKeys map[string]string
	}{
		{
			name:        "successful IDE attachment",
			vmxData:     map[string]string{},
			devicePath:  "ide0:1",
			isoPath:     "/path/to/tools.iso",
			adapterType: "ide",
			expectError: false,
			expectedKeys: map[string]string{
				"ide0.present":      "TRUE",
				"ide0:1.present":    "TRUE",
				"ide0:1.filename":   "/path/to/tools.iso",
				"ide0:1.devicetype": "cdrom-image",
			},
		},
		{
			name:        "successful SATA attachment",
			vmxData:     map[string]string{},
			devicePath:  "sata0:0",
			isoPath:     "/path/to/tools.iso",
			adapterType: "sata",
			expectError: false,
			expectedKeys: map[string]string{
				"sata0.present":      "TRUE",
				"sata0:0.present":    "TRUE",
				"sata0:0.filename":   "/path/to/tools.iso",
				"sata0:0.devicetype": "cdrom-image",
			},
		},
		{
			name:        "successful SCSI attachment",
			vmxData:     map[string]string{},
			devicePath:  "scsi0:8",
			isoPath:     "/path/to/tools.iso",
			adapterType: "scsi",
			expectError: false,
			expectedKeys: map[string]string{
				"scsi0.present":      "TRUE",
				"scsi0:8.present":    "TRUE",
				"scsi0:8.filename":   "/path/to/tools.iso",
				"scsi0:8.devicetype": "cdrom-image",
			},
		},
		{
			name: "device already in use - should return error",
			vmxData: map[string]string{
				"ide0:1.present": "TRUE",
			},
			devicePath:  "ide0:1",
			isoPath:     "/path/to/tools.iso",
			adapterType: "ide",
			expectError: true,
		},
		{
			name:        "nil vmxData - should return error",
			vmxData:     nil,
			devicePath:  "ide0:1",
			isoPath:     "/path/to/tools.iso",
			adapterType: "ide",
			expectError: true,
		},
		{
			name:        "empty devicePath - should return error",
			vmxData:     map[string]string{},
			devicePath:  "",
			isoPath:     "/path/to/tools.iso",
			adapterType: "ide",
			expectError: true,
		},
		{
			name:        "empty isoPath - should return error",
			vmxData:     map[string]string{},
			devicePath:  "ide0:1",
			isoPath:     "",
			adapterType: "ide",
			expectError: true,
		},
		{
			name:        "empty adapterType - should return error",
			vmxData:     map[string]string{},
			devicePath:  "ide0:1",
			isoPath:     "/path/to/tools.iso",
			adapterType: "",
			expectError: true,
		},
		{
			name:        "invalid device path format - should return error",
			vmxData:     map[string]string{},
			devicePath:  "invalid",
			isoPath:     "/path/to/tools.iso",
			adapterType: "ide",
			expectError: true,
		},
		{
			name:        "mismatched adapter types - should return error",
			vmxData:     map[string]string{},
			devicePath:  "ide0:1",
			isoPath:     "/path/to/tools.iso",
			adapterType: "sata",
			expectError: true,
		},
		{
			name:        "case insensitive adapter type",
			vmxData:     map[string]string{},
			devicePath:  "ide0:1",
			isoPath:     "/path/to/tools.iso",
			adapterType: "IDE",
			expectError: false,
			expectedKeys: map[string]string{
				"ide0.present":      "TRUE",
				"ide0:1.present":    "TRUE",
				"ide0:1.filename":   "/path/to/tools.iso",
				"ide0:1.devicetype": "cdrom-image",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AttachCDROMDevice(tt.vmxData, tt.devicePath, tt.isoPath, tt.adapterType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check that expected keys are present with correct values
			for key, expectedValue := range tt.expectedKeys {
				if actualValue, exists := tt.vmxData[key]; !exists {
					t.Errorf("expected key %s not found in vmxData", key)
				} else if actualValue != expectedValue {
					t.Errorf("key %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestDetachCDROMDevice(t *testing.T) {
	tests := []struct {
		name           string
		vmxData        map[string]string
		devicePath     string
		expectError    bool
		expectedRemain map[string]string
	}{
		{
			name: "successful detachment - removes all device keys",
			vmxData: map[string]string{
				"ide0.present":      "TRUE",
				"ide0:1.present":    "TRUE",
				"ide0:1.filename":   "/path/to/tools.iso",
				"ide0:1.devicetype": "cdrom-image",
				"ide0:0.present":    "TRUE", // should remain
				"sata0:0.present":   "TRUE", // should remain
			},
			devicePath:  "ide0:1",
			expectError: false,
			expectedRemain: map[string]string{
				"ide0.present":    "TRUE",
				"ide0:0.present":  "TRUE",
				"sata0:0.present": "TRUE",
			},
		},
		{
			name:           "detach from empty vmxData - should succeed",
			vmxData:        map[string]string{},
			devicePath:     "ide0:1",
			expectError:    false,
			expectedRemain: map[string]string{},
		},
		{
			name: "detach non-existent device - should succeed",
			vmxData: map[string]string{
				"ide0:0.present": "TRUE",
			},
			devicePath:  "ide0:1",
			expectError: false,
			expectedRemain: map[string]string{
				"ide0:0.present": "TRUE",
			},
		},
		{
			name:        "nil vmxData - should return error",
			vmxData:     nil,
			devicePath:  "ide0:1",
			expectError: true,
		},
		{
			name:        "empty devicePath - should return error",
			vmxData:     map[string]string{},
			devicePath:  "",
			expectError: true,
		},
		{
			name:        "invalid device path format - should return error",
			vmxData:     map[string]string{},
			devicePath:  "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DetachCDROMDevice(tt.vmxData, tt.devicePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check that only expected keys remain
			if len(tt.vmxData) != len(tt.expectedRemain) {
				t.Errorf("expected %d keys to remain, got %d", len(tt.expectedRemain), len(tt.vmxData))
			}

			for key, expectedValue := range tt.expectedRemain {
				if actualValue, exists := tt.vmxData[key]; !exists {
					t.Errorf("expected key %s not found in vmxData", key)
				} else if actualValue != expectedValue {
					t.Errorf("key %s: expected %s, got %s", key, expectedValue, actualValue)
				}
			}

			// Check that device-specific keys were removed
			devicePrefix := tt.devicePath + "."
			for key := range tt.vmxData {
				if len(key) > len(devicePrefix) && key[:len(devicePrefix)] == devicePrefix {
					t.Errorf("device key %s should have been removed", key)
				}
			}
		})
	}
}

// TestCDROMUtilsIntegration tests the utilities working together
func TestCDROMUtilsIntegration(t *testing.T) {
	vmxData := map[string]string{
		"ide0:0.present": "TRUE", // existing installation ISO
	}

	devicePath, err := FindNextAvailableCDROMSlot(vmxData, "ide")
	if err != nil {
		t.Fatalf("FindNextAvailableCDROMSlot failed: %v", err)
	}

	expectedPath := "ide0:1"
	if devicePath != expectedPath {
		t.Errorf("expected device path %s, got %s", expectedPath, devicePath)
	}

	isoPath := "/path/to/vmware-tools.iso"
	err = AttachCDROMDevice(vmxData, devicePath, isoPath, "ide")
	if err != nil {
		t.Fatalf("AttachCDROMDevice failed: %v", err)
	}

	expectedKeys := map[string]string{
		"ide0.present":      "TRUE",
		"ide0:1.present":    "TRUE",
		"ide0:1.filename":   isoPath,
		"ide0:1.devicetype": "cdrom-image",
	}

	for key, expectedValue := range expectedKeys {
		if actualValue, exists := vmxData[key]; !exists {
			t.Errorf("expected key %s not found after attachment", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}

	err = DetachCDROMDevice(vmxData, devicePath)
	if err != nil {
		t.Fatalf("DetachCDROMDevice failed: %v", err)
	}

	expectedRemaining := map[string]string{
		"ide0.present":   "TRUE",
		"ide0:0.present": "TRUE",
	}

	if len(vmxData) != len(expectedRemaining) {
		t.Errorf("expected %d keys after detachment, got %d", len(expectedRemaining), len(vmxData))
	}

	for key, expectedValue := range expectedRemaining {
		if actualValue, exists := vmxData[key]; !exists {
			t.Errorf("expected key %s not found after detachment", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}
}
