// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
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
		"signalfx_detector",
	}

	for name := range p.ResourcesMap {
		assert.Contains(t, expected, name, "Must have the resource defined as part of provider")
	}

	for _, name := range expected {
		assert.Contains(t, p.ResourcesMap, name, "Must have the expected resource defined in provider")
	}
}

func TestProviderConfiguration(t *testing.T) {

	for _, tc := range []struct {
		name    string
		details map[string]any
		expect  diag.Diagnostics
	}{
		{
			name:    "no details provided",
			details: make(map[string]any),
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "missing auth token or email and password"},
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
		{
			name: "Adding feature previews",
			details: map[string]any{
				"auth_token": "hunter2",
				"api_url":    "api.us.signalfx.com",
				"feature_preview": map[string]any{
					"feature-01": true,
				},
			},
			expect: diag.Diagnostics{
				{Severity: diag.Warning, Summary: "no preview with id \"feature-01\" found"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tftest.CleanEnvVars(t)

			actual := New().Configure(
				context.Background(),
				terraform.NewResourceConfigRaw(tc.details),
			)

			assert.Equal(t, tc.expect, actual, "Must match the expected details")
		})
	}
}
