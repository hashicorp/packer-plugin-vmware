// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// Artifact is the result of running the VMware builder, namely a set
// of files associated with the resulting machine.
type artifact struct {
	builderId string
	id        string
	dir       OutputDir
	f         []string
	config    map[string]string

	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

// BuilderId returns the unique identifier for the builder that created this artifact.
func (a *artifact) BuilderId() string {
	return a.builderId
}

// Files returns a list of file paths associated with this artifact.
func (a *artifact) Files() []string {
	return a.f
}

// Id returns the unique identifier for this artifact instance.
func (a *artifact) Id() string {
	return a.id
}

// String returns a human-readable description of the artifact.
func (a *artifact) String() string {
	return fmt.Sprintf("VM files in directory: %s", a.dir)
}

// State retrieves configuration or state data associated with the artifact by name.
func (a *artifact) State(name string) interface{} {
	if _, ok := a.StateData[name]; ok {
		return a.StateData[name]
	}
	return a.config[name]
}

// Destroy removes all files and directories associated with this artifact.
func (a *artifact) Destroy() error {
	if a.dir != nil {
		return a.dir.RemoveAll()
	}
	return nil
}

// NewArtifact creates a new artifact from the build results and configuration.
func NewArtifact(format string, vmName string, skipExport bool, state multistep.StateBag) (packersdk.Artifact, error) {
	dir := state.Get("dir").(OutputDir)

	files, err := dir.ListFiles()
	if err != nil {
		return nil, err
	}

	builderId := builderId

	config := make(map[string]string)
	config[artifactConfFormat] = format
	config[artifactConfSkipExport] = strconv.FormatBool(skipExport)

	return &artifact{
		builderId: builderId,
		id:        vmName,
		dir:       dir,
		f:         files,
		config:    config,
		StateData: map[string]interface{}{"generated_data": state.Get("generated_data")},
	}, nil
}
