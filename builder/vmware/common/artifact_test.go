// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestLocalArtifact_impl(t *testing.T) {
	var _ packersdk.Artifact = new(artifact)
}
