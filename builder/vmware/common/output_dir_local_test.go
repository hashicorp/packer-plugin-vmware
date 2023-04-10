// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestLocalOutputDir_impl(t *testing.T) {
	var _ OutputDir = new(LocalOutputDir)
}
