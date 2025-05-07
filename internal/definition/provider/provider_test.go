// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
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
		meta    any
		expect  diag.Diagnostics
	}{
		{
			name:    "no details provided",
			details: make(map[string]any),
			meta:    nil,
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
			meta: &pmeta.Meta{
				AuthToken:    "hunter2",
				APIURL:       "api.us.signalfx.com",
				CustomAppURL: "https://app.signalfx.com",
				Tags:         []string{},
				Teams:        []string{},
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
			meta: nil,
			expect: diag.Diagnostics{
				{Severity: diag.Warning, Summary: "no preview with id \"feature-01\" found"},
			},
		},
		{
			name: "Adding Provider tags",
			details: map[string]any{
				"auth_token": "hunter2",
				"api_url":    "api.signalfx.com",
				"tags": []any{
					"brown",
					"bear",
					"battery",
					"staple",
				},
			},
			meta: &pmeta.Meta{
				AuthToken:    "hunter2",
				APIURL:       "api.signalfx.com",
				CustomAppURL: "https://app.signalfx.com",
				Tags: []string{
					"brown",
					"bear",
					"battery",
					"staple",
				},
				Teams: []string{},
			},
			expect: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tftest.CleanEnvVars(t)

			provider := New()
			actual := provider.Configure(
				context.Background(),
				terraform.NewResourceConfigRaw(tc.details),
			)

			meta := provider.Meta()
			if m, ok := meta.(*pmeta.Meta); ok {
				assert.NotNil(t, m.Client, "Must have a valid client")
				// Removing the client from the returned provider since it is hard to compare
				m.Client = nil
			}

			assert.Equal(t, tc.meta, meta, "Must match the expected value")
			assert.Equal(t, tc.expect, actual, "Must match the expected details")
		})
	}
}

func TestProviderTracking(t *testing.T) {
	t.Parallel()

	// Moved to a separate unit test since the tracking results are branch dependant

	rc := terraform.NewResourceConfigRaw(map[string]any{
		"auth_token": "hunter2",
		"api_url":    "api.us.signalfx.com",
		"feature_preview": map[string]any{
			feature.PreviewProviderTags:     true,
			feature.PreviewProviderTracking: true,
		},
	})

	provider := New()
	require.Empty(t, provider.Configure(t.Context(), rc), "Must not return any issues trying to configure provider")

	tags := pmeta.LoadProviderTags(t.Context(), provider.Meta())
	require.Len(t, tags, 3, "Must only have the tags provided from tracking")
	for i, prefix := range []string{"project:", "branch:", "experimental:"} {
		assert.True(t, strings.HasPrefix(tags[i], prefix), "Must have the expected prefix %q with actual %q", prefix, tags[i])
	}
}
