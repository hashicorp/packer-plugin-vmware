// © Broadcom. All Rights Reserved.
// The term "Broadcom" refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"os"
	"path/filepath"
)

// LocalOutputDir is an OutputDir implementation where the directory
// is on the local machine.
type LocalOutputDir struct {
	dir string
}

// DirExists checks if the output directory exists on the local filesystem.
func (d *LocalOutputDir) DirExists() (bool, error) {
	_, err := os.Stat(d.dir)
	return err == nil, nil
}

// ListFiles returns a list of all files in the output directory.
func (d *LocalOutputDir) ListFiles() ([]string, error) {
	files := make([]string, 0, 10)

	visit := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	}

	return files, filepath.Walk(d.dir, visit)
}

// MkdirAll creates the output directory and any necessary parent directories.
func (d *LocalOutputDir) MkdirAll() error {
	return os.MkdirAll(d.dir, 0755)
}

// Remove deletes the specified file from the output directory.
func (d *LocalOutputDir) Remove(path string) error {
	return os.Remove(path)
}

// RemoveAll deletes the entire output directory and all its contents.
func (d *LocalOutputDir) RemoveAll() error {
	return os.RemoveAll(d.dir)
}

// SetOutputDir sets the path for the output directory.
func (d *LocalOutputDir) SetOutputDir(path string) {
	d.dir = path
}

// String returns the path of the output directory as a string.
func (d *LocalOutputDir) String() string {
	return d.dir
}
