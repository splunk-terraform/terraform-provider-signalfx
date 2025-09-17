// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestModelEnsureDefaults(t *testing.T) {

	for _, tc := range []struct {
		name     string
		model    *OllyProviderModel
		env      map[string]string
		expected *OllyProviderModel
	}{
		{
			name:  "no env, no defaults set",
			model: &OllyProviderModel{},
			env:   map[string]string{},
			expected: &OllyProviderModel{
				AuthToken:           types.StringNull(),
				APIURL:              types.StringNull(),
				TimeoutSeconds:      types.Int64Value(60),
				RetryMaxAttempts:    types.Int32Value(5),
				RetryWaitMinSeconds: types.Int64Value(1),
				RetryWaitMaxSeconds: types.Int64Value(10),
			},
		},
		{
			name:  "environment variables set, no defaults set",
			model: &OllyProviderModel{},
			env: map[string]string{
				"SFX_AUTH_TOKEN": "test-auth-token",
				"SFX_API_URL":    "https://example.com",
			},
			expected: &OllyProviderModel{
				AuthToken:           types.StringValue("test-auth-token"),
				APIURL:              types.StringValue("https://example.com"),
				TimeoutSeconds:      types.Int64Value(60),
				RetryMaxAttempts:    types.Int32Value(5),
				RetryWaitMinSeconds: types.Int64Value(1),
				RetryWaitMaxSeconds: types.Int64Value(10),
			},
		},
		{
			name: "values are defined",
			model: &OllyProviderModel{
				AuthToken:           types.StringValue("defined-auth-token"),
				APIURL:              types.StringValue("https://example.com"),
				TimeoutSeconds:      types.Int64Value(120),
				RetryMaxAttempts:    types.Int32Value(10),
				RetryWaitMinSeconds: types.Int64Value(2),
				RetryWaitMaxSeconds: types.Int64Value(20),
			},
			env: map[string]string{
				"SFX_AUTH_TOKEN": "test-auth-token",
				"SFX_API_URL":    "https://example.com/v2",
			},
			expected: &OllyProviderModel{
				AuthToken:           types.StringValue("defined-auth-token"),
				APIURL:              types.StringValue("https://example.com"),
				TimeoutSeconds:      types.Int64Value(120),
				RetryMaxAttempts:    types.Int32Value(10),
				RetryWaitMinSeconds: types.Int64Value(2),
				RetryWaitMaxSeconds: types.Int64Value(20),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			tc.model.EnsureDefaults()
			assert.Equal(t, tc.expected, tc.model)
		})
	}
}
