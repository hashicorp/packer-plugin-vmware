// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestDriverMock_impl(t *testing.T) {
	var _ Driver = new(DriverMock)
}
