// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, observabilityDirectoryModel{
		Templates: types.ListNull(types.StringType),
	}))
}

func TestObservabilityDirectoryPatchReplacesMembership(t *testing.T) {
	templates, diags := types.ListValueFrom(t.Context(), types.StringType, []string{
		"/v2/template/dashboard-a",
		"/v2/template/dashboard-b",
	})
	require.False(t, diags.HasError(), diags)

	patch, diags := observabilityDirectoryPatch(t.Context(), types.BoolValue(true), templates)
	require.False(t, diags.HasError(), diags)
	require.NotNil(t, patch)
	require.NotNil(t, patch.Templates)
	assert.Equal(t, []string{"/v2/template/dashboard-a", "/v2/template/dashboard-b"}, *patch.Templates)
	require.NotNil(t, patch.Pinned)
	assert.True(t, *patch.Pinned)

	model, diags := observabilityDirectoryModelFromEntry(t.Context(), &directory.Entry{
		Path:      "team/dashboards",
		Templates: []string{"/v2/template/dashboard-a", "/v2/template/dashboard-b"},
		Pinned:    true,
	})
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "team/dashboards", model.ID.ValueString())
	assert.True(t, model.Pinned.ValueBool())
	var actual []string
	require.False(t, model.Templates.ElementsAs(t.Context(), &actual, false).HasError())
	assert.Equal(t, []string{"/v2/template/dashboard-a", "/v2/template/dashboard-b"}, actual)

	empty, diags := types.ListValueFrom(t.Context(), types.StringType, []string{})
	require.False(t, diags.HasError(), diags)
	patch, diags = observabilityDirectoryPatch(t.Context(), types.BoolValue(false), empty)
	require.False(t, diags.HasError(), diags)
	require.NotNil(t, patch.Templates)
	assert.Empty(t, *patch.Templates)
}

func TestResourceObservabilityDirectoryTemplatesRejectsDuplicates(t *testing.T) {
	t.Parallel()

	var schemaResponse resource.SchemaResponse
	NewResourceObservabilityDirectory().(*observabilityDirectoryResource).Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	templatesAttr := schemaResponse.Schema.Attributes["templates"].(schema.ListAttribute)

	validate := func(values []string) diag.Diagnostics {
		list, diags := types.ListValueFrom(t.Context(), types.StringType, values)
		require.False(t, diags.HasError(), diags)
		var response validator.ListResponse
		for _, v := range templatesAttr.Validators {
			v.ValidateList(t.Context(), validator.ListRequest{ConfigValue: list}, &response)
		}
		return response.Diagnostics
	}

	assert.True(t, validate([]string{"/v2/template/a", "/v2/template/a"}).HasError())
	assert.False(t, validate([]string{"/v2/template/a", "/v2/template/b"}).HasError())
}

func TestObservabilityReservedDirectoryPath(t *testing.T) {
	for _, path := range []string{"~templates", "~users", "team/~users", "~observability/homepage"} {
		assert.Equal(t, path, observabilityReservedDirectoryPath(path))
	}
	assert.Empty(t, observabilityReservedDirectoryPath("team/dashboards"))
}

func TestObservabilityDirectoryDeleteSafeguards(t *testing.T) {
	tests := map[string]struct {
		entry       directory.Entry
		wantError   string
		wantWarning string
		wantDelete  bool
	}{
		"empty entry": {
			entry:      directory.Entry{Path: "teams/platform/dashboards"},
			wantDelete: true,
		},
		"template membership": {
			entry:       directory.Entry{Path: "teams/platform/dashboards", Templates: []string{"/v2/template/dashboard-id"}},
			wantWarning: "Template(s)",
			wantDelete:  true,
		},
		"child directory": {
			entry:     directory.Entry{Path: "teams/platform/dashboards", Children: []string{"child"}},
			wantError: "managed by the service or contains child directories",
		},
		"identity entry": {
			entry:     directory.Entry{Path: "teams/platform/dashboards", Identity: true},
			wantError: "managed by the service or contains child directories",
		},
		"canonical entry": {
			entry:     directory.Entry{Path: "teams/platform/dashboards", Canonical: true},
			wantError: "managed by the service or contains child directories",
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
			model, diags := observabilityDirectoryModelFromEntry(t.Context(), &test.entry)
			require.False(t, diags.HasError(), diags)
			require.False(t, state.Set(t.Context(), model).HasError())

			response := resource.DeleteResponse{State: state}
			managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)

			if test.wantError == "" {
				require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
			} else {
				require.True(t, response.Diagnostics.HasError())
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), test.wantError)
			}
			if test.wantWarning != "" {
				require.NotEmpty(t, response.Diagnostics.Warnings())
				assert.Contains(t, response.Diagnostics.Warnings()[0].Detail(), test.wantWarning)
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
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.0", "/v2/template/dashboard-a"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.1", "/v2/template/dashboard-b"),
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
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.0", "/v2/template/dashboard-b"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.1", "/v2/template/dashboard-c"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "false"),
					),
				},
				{
					PreConfig: func() {
						store.mu.Lock()
						store.entries["teams/platform/dashboards"].Templates = []string{
							"/v2/template/dashboard-b",
							"/v2/template/dashboard-c",
							"/v2/template/ui-added",
						}
						store.mu.Unlock()
					},
					ConfigFile: config.StaticFile("testdata/01_observability_directory_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.0", "/v2/template/dashboard-b"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.1", "/v2/template/dashboard-c"),
					),
				},
				{
					ConfigFile: config.StaticFile("testdata/02_observability_directory_empty.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "0"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "false"),
					),
				},
			},
		},
	)
}
