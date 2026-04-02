// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"regexp"
	"strings"
)

const (
	invalidCharacters = `(?<seq>[^\w]+)`
)

var (
	rex = regexp.MustCompile(invalidCharacters)
)

// NewCompatibleIdentifer replaces non compatible characters in a given string
// and replaces them with underscores to make it possible to use as a Terraform identifier.
func NewCompatibleIdentifer(name string) string {
	return strings.Trim(rex.ReplaceAllString(name, "_"), "_")
}
