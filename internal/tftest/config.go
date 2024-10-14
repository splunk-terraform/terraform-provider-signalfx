// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import "os"

// LoadConfig is used to read a configuration file from
// disk and used within acceptance tests.
// If there is any issue trying to read the file, this function will panic.
func LoadConfig(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}
