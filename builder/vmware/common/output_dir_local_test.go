// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestLocalOutputDir_impl(t *testing.T) {
	var _ OutputDir = new(LocalOutputDir)
}
