// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package common

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	mockWorkstationVerifyVersionFunc func(string) error
	mockGlobFunc                     func(string) ([]string, error)
)

func mockWorkstationVerifyVersion(workstationNoLicenseVersion string) error {
	if mockWorkstationVerifyVersionFunc != nil {
		return mockWorkstationVerifyVersionFunc(workstationNoLicenseVersion)
	}
	return nil
}

func mockGlob(pattern string) ([]string, error) {
	if mockGlobFunc != nil {
		return mockGlobFunc(pattern)
	}
	return nil, nil
}

func testableWorkstationCheckLicense(
	workstationVerifyVersion func(string) error,
	glob func(string) ([]string, error),
) error {
	err := workstationVerifyVersion(workstationNoLicenseVersion)
	if err == nil {
		return nil
	}

	var errVersionRequiresLicense = errors.New("installed version requires a license")
	if err.Error() == errVersionRequiresLicense.Error() {
		matches, globErr := glob("/etc/vmware/license-ws-*")
		if globErr != nil {
			return fmt.Errorf("error finding license: %w", globErr) // Propagate glob errors correctly
		}
		if len(matches) == 0 {
			return errors.New("no license found")
		}

		return nil
	}

	return err
}

func TestWorkstationCheckLicense(t *testing.T) {
	defer func() {
		mockWorkstationVerifyVersionFunc = nil
		mockGlobFunc = nil
	}()

	testCases := []struct {
		name                          string
		mockVerifyVersionResponse     error
		mockGlobResponse              []string
		mockGlobError                 error
		expectedErrorMessageSubstring string
	}{
		{
			name:                          fmt.Sprintf("version greater or equal to %s, skip license check", workstationNoLicenseVersion),
			mockVerifyVersionResponse:     nil,
			mockGlobResponse:              nil,
			mockGlobError:                 nil,
			expectedErrorMessageSubstring: "",
		},
		{
			name:                          fmt.Sprintf("version lower than %s, license found", workstationNoLicenseVersion),
			mockVerifyVersionResponse:     errors.New("installed version requires a license"),
			mockGlobResponse:              []string{"/etc/vmware/license-ws-1234"},
			mockGlobError:                 nil,
			expectedErrorMessageSubstring: "",
		},
		{
			name:                          fmt.Sprintf("version lower than %s, no license found", workstationNoLicenseVersion),
			mockVerifyVersionResponse:     errors.New("installed version requires a license"),
			mockGlobResponse:              []string{},
			mockGlobError:                 nil,
			expectedErrorMessageSubstring: "no license found",
		},
		{
			name:                          "fail to detect version",
			mockVerifyVersionResponse:     fmt.Errorf("failed to detect version"),
			mockGlobResponse:              nil,
			mockGlobError:                 nil,
			expectedErrorMessageSubstring: "failed to detect version",
		},
		{
			name:                          "error finding license file",
			mockVerifyVersionResponse:     errors.New("installed version requires a license"),
			mockGlobResponse:              nil,
			mockGlobError:                 fmt.Errorf("error finding license file"),
			expectedErrorMessageSubstring: "error finding license file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWorkstationVerifyVersionFunc = func(workstationNoLicenseVersion string) error {
				return tc.mockVerifyVersionResponse
			}

			mockGlobFunc = func(pattern string) ([]string, error) {
				return tc.mockGlobResponse, tc.mockGlobError
			}

			err := testableWorkstationCheckLicense(mockWorkstationVerifyVersion, mockGlob)

			if tc.expectedErrorMessageSubstring == "" {
				assert.NoError(t, err, tc.name)
			} else {
				assert.Error(t, err, tc.name)
				assert.Contains(t, err.Error(), tc.expectedErrorMessageSubstring, tc.name)
			}
		})
	}
}
