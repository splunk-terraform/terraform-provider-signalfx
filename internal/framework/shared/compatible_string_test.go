// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCompatibleIdentifer(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		input  string
		expect string
	}{
		{name: "empty string", input: "", expect: ""},
		{name: "valid string", input: "helper", expect: "helper"},
		{name: "string with spaces", input: "best resource", expect: "best_resource"},
		{name: "uses special characters", input: "free@sprite$", expect: "free_sprite"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, NewCompatibleIdentifer(tc.input), "Must match the expected compatible string")
		})
	}
}
