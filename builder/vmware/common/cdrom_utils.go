// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FindNextAvailableCDROMSlot locates the next available CD-ROM device slot for
// the specified adapter type.
func FindNextAvailableCDROMSlot(vmxData map[string]string, adapterType string) (string, error) {
	if adapterType == "" {
		return "", fmt.Errorf("adapter type cannot be empty")
	}

	adapterType = strings.ToLower(adapterType)

	validAdapters := []string{"ide", "sata", "scsi"}
	isValid := false
	for _, valid := range validAdapters {
		if adapterType == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return "", fmt.Errorf("invalid adapter type: %s; must be one of %v", adapterType, validAdapters)
	}

	devicePattern := regexp.MustCompile(fmt.Sprintf(`^%s(\d+):(\d+)\.present$`, adapterType))
	usedSlots := make(map[string]map[int]bool) // bus -> slot -> used

	for key := range vmxData {
		if matches := devicePattern.FindStringSubmatch(key); matches != nil {
			bus := matches[1]
			slot, err := strconv.Atoi(matches[2])
			if err != nil {
				continue
			}

			if usedSlots[bus] == nil {
				usedSlots[bus] = make(map[int]bool)
			}
			usedSlots[bus][slot] = true
		}
	}

	// Find the next available slot.
	for busNum := 0; busNum < 4; busNum++ { // VMware supports up to 4 buses for most adapter types
		busStr := strconv.Itoa(busNum)
		busSlots := usedSlots[busStr]

		// Check slots 0-15 (or 0-6,8-15 for SCSI to skip reserved slot 7).
		maxSlots := 16
		if adapterType == "ide" {
			maxSlots = 2 // IDE typically supports only 2 devices per bus.
		}

		for slot := 0; slot < maxSlots; slot++ {
			// Skip reserved slot 7 for SCSI adapters.
			if adapterType == "scsi" && slot == 7 {
				continue
			}

			if busSlots == nil || !busSlots[slot] {
				return fmt.Sprintf("%s%d:%d", adapterType, busNum, slot), nil
			}
		}
	}

	return "", fmt.Errorf("no available CD-ROM slots found for adapter type %s", adapterType)
}

// AttachCDROMDevice adds CD-ROM device entries to VMX data for the specified
// device path and ISO file.
func AttachCDROMDevice(vmxData map[string]string, devicePath, isoPath, adapterType string) error {
	if vmxData == nil {
		return fmt.Errorf("vmxData cannot be nil")
	}
	if devicePath == "" {
		return fmt.Errorf("devicePath cannot be empty")
	}
	if isoPath == "" {
		return fmt.Errorf("isoPath cannot be empty")
	}
	if adapterType == "" {
		return fmt.Errorf("adapterType cannot be empty")
	}

	adapterType = strings.ToLower(adapterType)
	devicePattern := regexp.MustCompile(`^(ide|sata|scsi)(\d+):(\d+)$`)
	matches := devicePattern.FindStringSubmatch(devicePath)
	if matches == nil {
		return fmt.Errorf("invalid device path format: %s; expected format like 'ide0:1'", devicePath)
	}

	pathAdapterType := matches[1]
	busNum := matches[2]

	if pathAdapterType != adapterType {
		return fmt.Errorf("device path adapter type %s does not match specified adapter type %s", pathAdapterType, adapterType)
	}

	presentKey := fmt.Sprintf("%s.present", devicePath)
	if existing, exists := vmxData[presentKey]; exists && strings.ToLower(existing) == "true" {
		return fmt.Errorf("device %s is already in use", devicePath)
	}

	adapterKey := fmt.Sprintf("%s%s.present", adapterType, busNum)
	vmxData[adapterKey] = "TRUE"

	vmxData[fmt.Sprintf("%s.present", devicePath)] = "TRUE"
	vmxData[fmt.Sprintf("%s.filename", devicePath)] = isoPath
	vmxData[fmt.Sprintf("%s.devicetype", devicePath)] = "cdrom-image"

	return nil
}

// DetachCDROMDevice removes CD-ROM device entries from VMX data for the
// specified device path.
func DetachCDROMDevice(vmxData map[string]string, devicePath string) error {
	if vmxData == nil {
		return fmt.Errorf("vmxData cannot be nil")
	}
	if devicePath == "" {
		return fmt.Errorf("devicePath cannot be empty")
	}

	devicePattern := regexp.MustCompile(`^(ide|sata|scsi)(\d+):(\d+)$`)
	if !devicePattern.MatchString(devicePath) {
		return fmt.Errorf("invalid device path format: %s; expected format like 'ide0:1'", devicePath)
	}

	devicePrefix := devicePath + "."
	keysToDelete := make([]string, 0)

	for key := range vmxData {
		if strings.HasPrefix(key, devicePrefix) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(vmxData, key)
	}

	return nil
}
