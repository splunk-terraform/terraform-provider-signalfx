// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwembed

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/assert"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func TestDatasource_Configure(t *testing.T) {
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
			expectDiagnostics: false,
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

			ds := &DatasourceData{}
			req := datasource.ConfigureRequest{
				ProviderData: tc.providerData,
			}
			resp := &datasource.ConfigureResponse{
				Diagnostics: diag.Diagnostics{},
			}

			ds.Configure(context.Background(), req, resp)

			assert.Equal(t, tc.expectDiagnostics, resp.Diagnostics.HasError(), "Expected diagnostics to match")
			assert.Equal(t, tc.expectedMeta, ds.Details(), "Expected meta to match")
		})
	}
}
