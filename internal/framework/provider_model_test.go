// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestNewStringFromEnvironment(t *testing.T) {

	for _, tc := range []struct {
		name   string
		env    map[string]string
		expect types.String
	}{
		{
			name:   "missing env var",
			env:    map[string]string{},
			expect: types.StringNull(),
		},
		{
			name:   "empty env var",
			env:    map[string]string{"TEST_ENVVAR": ""},
			expect: types.StringValue(""),
		},
		{
			name:   "set env var",
			env:    map[string]string{"TEST_ENVVAR": "some-value"},
			expect: types.StringValue("some-value"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			actual := NewStringFromEnvironment("TEST_ENVVAR")
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
		})
	}
}
