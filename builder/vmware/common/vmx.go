// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ParseVMX parses the keys and values from a VMX file and returns
// them as a Go map.
func ParseVMX(contents string) map[string]string {
	results := make(map[string]string)

	lineRe := regexp.MustCompile(`^(.+?)\s*=\s*"?(.*?)"?\s*$`)

	for _, line := range strings.Split(contents, "\n") {
		matches := lineRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		key := strings.ToLower(matches[1])
		results[key] = matches[2]
	}

	return results
}

// EncodeVMX takes a map and turns it into valid VMX contents.
func EncodeVMX(contents map[string]string) string {
	var buf bytes.Buffer

	i := 0
	keys := make([]string, len(contents))
	for k := range contents {
		keys[i] = k
		i++
	}

	// a list of VMX key fragments that the value must not be quoted
	// fragments are used to cover multiples (i.e. multiple disks)
	// keys are still lowercase at this point, use lower fragments
	noQuotes := []string{
		".virtualssd",
	}

	// a list of VMX key fragments that are case-sensitive
	// fragments are used to cover multiples (i.e. multiple disks)
	caseSensitive := []string{
		".virtualSSD",
	}

	sort.Strings(keys)
	for _, k := range keys {
		pat := "%s = \"%s\"\n"
		// items with no quotes
		for _, q := range noQuotes {
			if strings.Contains(k, q) {
				pat = "%s = %s\n"
				break
			}
		}
		key := k
		// Case-sensitive key fragments.
		for _, c := range caseSensitive {
			key = strings.Replace(key, strings.ToLower(c), c, 1)
		}
		buf.WriteString(fmt.Sprintf(pat, key, contents[k]))
	}

	return buf.String()
}

// WriteVMX takes a path to a VMX file and contents in the form of a
// map and writes it out.
func WriteVMX(path string, data map[string]string) (err error) {
	log.Printf("[INFO] Writing VMX to: %s", path)
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	var buf bytes.Buffer
	buf.WriteString(EncodeVMX(data))
	if _, err = io.Copy(f, &buf); err != nil {
		return
	}

	return
}

// ReadVMX takes a path to a VMX file and reads it into a k/v mapping.
func ReadVMX(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseVMX(string(data)), nil
}
