// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/directory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceObservabilityDirectoryMetadataAndSchema(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityDirectory()
	var metadata resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &metadata)
	assert.Equal(t, "signalfx_observability_directory", metadata.TypeName)
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, observabilityDirectoryModel{}))
}

func TestObservabilityDirectoryPatchPreservesMembership(t *testing.T) {
	patch := observabilityDirectoryPatch(types.BoolValue(true))
	require.NotNil(t, patch)
	assert.Nil(t, patch.Templates)
	require.NotNil(t, patch.Pinned)
	assert.True(t, *patch.Pinned)

	model := observabilityDirectoryModelFromEntry(&directory.Entry{Path: "team/dashboards", Pinned: true})
	assert.Equal(t, "team/dashboards", model.ID.ValueString())
	assert.True(t, model.Pinned.ValueBool())
}

func TestObservabilityReservedDirectoryPath(t *testing.T) {
	for _, path := range []string{"~templates", "~users", "team/~users", "~observability/homepage"} {
		assert.Equal(t, path, observabilityReservedDirectoryPath(path))
	}
	assert.Empty(t, observabilityReservedDirectoryPath("team/dashboards"))
}

func TestObservabilityDirectoryDeleteSafeguards(t *testing.T) {
	tests := map[string]struct {
		entry      directory.Entry
		wantError  string
		wantDelete bool
	}{
		"empty entry": {
			entry:      directory.Entry{Path: "teams/platform/dashboards"},
			wantDelete: true,
		},
		"template membership": {
			entry:     directory.Entry{Path: "teams/platform/dashboards", Templates: []string{"/v2/template/dashboard-id"}},
			wantError: "occupied or managed by the service",
		},
		"child directory": {
			entry:     directory.Entry{Path: "teams/platform/dashboards", Children: []string{"child"}},
			wantError: "occupied or managed by the service",
		},
		"identity entry": {
			entry:     directory.Entry{Path: "teams/platform/dashboards", Identity: true},
			wantError: "occupied or managed by the service",
		},
		"canonical entry": {
			entry:     directory.Entry{Path: "teams/platform/dashboards", Canonical: true},
			wantError: "occupied or managed by the service",
		},
		"reserved entry": {
			entry:     directory.Entry{Path: "~templates"},
			wantError: "reserved Directory path",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			store := newDirectoryAPIStore()
			store.entries[test.entry.Path] = &test.entry

			mockProvider := fwtest.NewMock(t, store.handlers())
			var providerResponse provider.ConfigureResponse
			mockProvider.Configure(t.Context(), provider.ConfigureRequest{}, &providerResponse)

			managed := NewResourceObservabilityDirectory().(*observabilityDirectoryResource)
			var configureResponse resource.ConfigureResponse
			managed.Configure(t.Context(), resource.ConfigureRequest{ProviderData: providerResponse.ResourceData}, &configureResponse)
			require.False(t, configureResponse.Diagnostics.HasError(), configureResponse.Diagnostics)

			var schemaResponse resource.SchemaResponse
			managed.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
			state := tfsdk.State{Schema: schemaResponse.Schema}
			require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
				ID:     types.StringValue(test.entry.Path),
				Path:   types.StringValue(test.entry.Path),
				Pinned: types.BoolValue(test.entry.Pinned),
			}).HasError())

			response := resource.DeleteResponse{State: state}
			managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)

			if test.wantError == "" {
				require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
			} else {
				require.True(t, response.Diagnostics.HasError())
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), test.wantError)
			}

			store.mu.Lock()
			_, exists := store.entries[test.entry.Path]
			store.mu.Unlock()
			assert.Equal(t, !test.wantDelete, exists)
		})
	}
}

func TestResourceObservabilityDirectoryLifecycleAndGeneratedConfig(t *testing.T) {
	store := newDirectoryAPIStore()

	testresource.UnitTest(
		t,
		testresource.TestCase{
			IsUnitTest: true,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_5_0),
			},
			ProtoV5ProviderFactories: fwtest.NewMockProto5Server(
				t,
				store.handlers(),
				fwtest.WithMockResources(NewResourceObservabilityDirectory),
			),
			Steps: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_observability_directory.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "id", "teams/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "path", "teams/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "true"),
					),
				},
				{
					ResourceName:    "signalfx_observability_directory.test",
					ImportState:     true,
					ImportStateKind: testresource.ImportBlockWithID,
					GenerateConfig:  true,
				},
				{
					ConfigFile: config.StaticFile("testdata/01_observability_directory_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "id", "teams/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "false"),
					),
				},
			},
		},
	)
}
