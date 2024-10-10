// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

func TestProviderValidation(t *testing.T) {
	t.Parallel()

	assert.NoError(t, New().InternalValidate(), "Must not error loading provider")
}

func TestProviderHasResource(t *testing.T) {
	t.Parallel()

	p := New()

	expected := []string{
		"signalfx_team",
	}

	for name := range p.ResourcesMap {
		assert.Contains(t, expected, name, "Must have the resource defined as part of provider")
	}

	for _, name := range expected {
		assert.Contains(t, p.ResourcesMap, name, "Must have the expected resource defined in provider")
	}
}

func TestProviderConfiguration(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		details map[string]any
		expect  diag.Diagnostics
	}{
		{
			name:    "no details provided",
			details: make(map[string]any),
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "auth token not set"},
			},
		},
		{
			name: "setting min required fields",
			details: map[string]any{
				"auth_token": "hunter2",
				"api_url":    "api.us.signalfx.com",
			},
			expect: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := New().Configure(
				context.Background(),
				terraform.NewResourceConfigRaw(tc.details),
			)

			assert.Equal(t, tc.expect, actual, "Must match the expected details")
		})
	}
}
