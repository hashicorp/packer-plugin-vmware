// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepExport represents a step to export a virtual machines to specific formats.
type StepExport struct {
	Format         string
	SkipExport     bool
	VMName         string
	OVFToolOptions []string
	OutputDir      *string
}

// generateExportArgs creates ovftool arguments for exporting from the hypervisor.
func (s *StepExport) generateExportArgs(exportOutputPath string) ([]string, error) {
	args := []string{
		filepath.Join(exportOutputPath, s.VMName+".vmx"),
		filepath.Join(exportOutputPath, s.VMName+"."+s.Format),
	}
	return append(s.OVFToolOptions, args...), nil
}

// Run executes the export step, converting the virtual machine to the specified format.
func (s *StepExport) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(Driver)

	// Skip export if requested
	if s.SkipExport {
		ui.Say("Skipping export of virtual machine...")
		return multistep.ActionContinue
	}

	// load output path from state. If it doesn't exist, just use the local
	// output directory.
	exportOutputPath, ok := state.Get("export_output_path").(string)
	if !ok || exportOutputPath == "" {
		if *s.OutputDir != "" {
			exportOutputPath = *s.OutputDir
		} else {
			exportOutputPath = s.VMName
		}
	}

	err := os.MkdirAll(exportOutputPath, 0755)
	if err != nil {
		err = fmt.Errorf("error creating export directory: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Exporting virtual machine...")

	var args []string

	ovftool := GetOvfTool()

	args, err = s.generateExportArgs(exportOutputPath)
	if err != nil {
		err = fmt.Errorf("error generating ovftool export args: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Sayf("Executing: %s %s", ovftool, strings.Join(args, " "))

	if err := driver.Export(args); err != nil {
		err = fmt.Errorf("error performing ovftool export: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue

}

// Cleanup performs any necessary cleanup after the export step completes.
func (s *StepExport) Cleanup(state multistep.StateBag) {}
