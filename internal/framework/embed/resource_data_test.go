// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwembed

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func TestResource_Configure(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		providerData      any
		expectDiagnostics bool
		expectedMeta      *pmeta.Meta
	}{
		{
			name:              "valid provider data",
			providerData:      &pmeta.Meta{},
			expectDiagnostics: false,
			expectedMeta:      &pmeta.Meta{},
		},
		{
			name:              "invalid provider data - wrong type",
			providerData:      "invalid",
			expectDiagnostics: true,
			expectedMeta:      nil,
		},
		{
			name:              "nil provider data",
			providerData:      nil,
			expectDiagnostics: true,
			expectedMeta:      nil,
		},
		{
			name:              "invalid provider data - int type",
			providerData:      42,
			expectDiagnostics: true,
			expectedMeta:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &ResourceData{}
			req := resource.ConfigureRequest{
				ProviderData: tc.providerData,
			}
			resp := &resource.ConfigureResponse{
				Diagnostics: diag.Diagnostics{},
			}

			r.Configure(context.Background(), req, resp)

			assert.Equal(t, tc.expectDiagnostics, resp.Diagnostics.HasError(), "Expected diagnostics to match")
			assert.Equal(t, tc.expectedMeta, r.Details(), "Expected meta to match")
		})
	}
}
