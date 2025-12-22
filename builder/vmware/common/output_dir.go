// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package common

// OutputDir is an interface type that abstracts the creation and handling
// of the output directory.
type OutputDir interface {
	DirExists() (bool, error)
	ListFiles() ([]string, error)
	MkdirAll() error
	Remove(string) error
	RemoveAll() error
	SetOutputDir(string)
	String() string
}
