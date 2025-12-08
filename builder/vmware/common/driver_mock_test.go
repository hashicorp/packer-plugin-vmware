// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestDriverMock_impl(t *testing.T) {
	var _ Driver = new(DriverMock)
}
