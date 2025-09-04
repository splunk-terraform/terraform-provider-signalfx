// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/require"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func TestNewMockProviderFactory_WithOptions(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		options    []func(*MockProvider)
		wantResLen int
		wantDSLen  int
	}

	mockResource := func() resource.Resource { return nil }
	mockDataSource := func() datasource.DataSource { return nil }

	tests := []testCase{
		{
			name:       "no options",
			options:    nil,
			wantResLen: 0,
			wantDSLen:  0,
		},
		{
			name:       "with resources",
			options:    []func(*MockProvider){WithMockResources(mockResource)},
			wantResLen: 1,
			wantDSLen:  0,
		},
		{
			name:       "with datasources",
			options:    []func(*MockProvider){WithMockDataSources(mockDataSource)},
			wantResLen: 0,
			wantDSLen:  1,
		},
		{
			name:       "with both",
			options:    []func(*MockProvider){WithMockResources(mockResource), WithMockDataSources(mockDataSource)},
			wantResLen: 1,
			wantDSLen:  1,
		},
	}

	endpoints := map[string]http.Handler{
		"/test": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			factory := NewMockProto5Server(t, endpoints, tc.options...)
			providerFunc, ok := factory["signalfx"]
			require.True(t, ok, "Provider function should exist in factory")

			providerServer, err := providerFunc()
			require.NoError(t, err)
			require.NotNil(t, providerServer)

			// Type assertion to access resources and datasources
			mockProvider := &MockProvider{}
			for _, opt := range tc.options {
				opt(mockProvider)
			}
			require.Len(t, mockProvider.Resources(t.Context()), tc.wantResLen)
			require.Len(t, mockProvider.DataSources(t.Context()), tc.wantDSLen)
		})
	}
}

func TestMockProvider_Metadata(t *testing.T) {
	t.Parallel()

	mockProvider := &MockProvider{}

	var resp provider.MetadataResponse
	mockProvider.Metadata(t.Context(), provider.MetadataRequest{}, &resp)

	require.Equal(t, "signalfx", resp.TypeName, "TypeName should be 'signalfx'")
	require.Equal(t, "1.0.0", resp.Version, "Version should be '1.0.0'")
}

func TestMockProvider_Schema(t *testing.T) {
	t.Parallel()

	mockProvider := &MockProvider{}

	var resp provider.SchemaResponse
	mockProvider.Schema(t.Context(), provider.SchemaRequest{}, &resp)

	require.NotNil(t, resp.Schema, "Schema should not be nil")
	require.Zero(t, resp.Schema.Attributes, "Schema.Attributes should be empty")
}

func TestMockProvider_Configure_SetsDataValues(t *testing.T) {
	t.Parallel()

	mockMeta := &pmeta.Meta{}
	mockProvider := &MockProvider{
		data: mockMeta,
	}

	var resp provider.ConfigureResponse
	mockProvider.Configure(t.Context(), provider.ConfigureRequest{}, &resp)

	require.Equal(t, mockMeta, resp.ResourceData, "ResourceData should be set to mockMeta")
	require.Equal(t, mockMeta, resp.DataSourceData, "DataSourceData should be set to mockMeta")
	require.Equal(t, mockMeta, resp.EphemeralResourceData, "EphemeralResourceData should be set to mockMeta")
}
