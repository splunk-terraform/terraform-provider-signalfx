// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		"dashboard-a",
		"dashboard-b",
	})
	require.False(t, diags.HasError(), diags)

	patch, diags := observabilityDirectoryPatch(t.Context(), types.BoolValue(true), templates)
	require.False(t, diags.HasError(), diags)
	require.NotNil(t, patch)
	require.NotNil(t, patch.Templates)
	assert.Equal(t, []string{"/v2/template/dashboard-a", "/v2/template/dashboard-b"}, patch.Templates)
	require.NotNil(t, patch.Pinned)
	assert.True(t, *patch.Pinned)

	model, diags := observabilityDirectoryModelFromEntry(t.Context(), "~organization/platform/dashboards", &directory.Entry{
		Path:      "~organization/platform/dashboards",
		Templates: []string{"/v2/template/dashboard-a", "/v2/template/dashboard-b"},
		Pinned:    true,
	})
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "~organization/platform/dashboards", model.ID.ValueString())
	assert.True(t, model.Pinned.ValueBool())
	var actual []string
	require.False(t, model.Templates.ElementsAs(t.Context(), &actual, false).HasError())
	assert.Equal(t, []string{"dashboard-a", "dashboard-b"}, actual)

	empty, diags := types.ListValueFrom(t.Context(), types.StringType, []string{})
	require.False(t, diags.HasError(), diags)
	patch, diags = observabilityDirectoryPatch(t.Context(), types.BoolValue(true), empty)
	require.False(t, diags.HasError(), diags)
	require.NotNil(t, patch.Templates)
	assert.Empty(t, patch.Templates)
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

	assert.True(t, validate([]string{"a", "a"}).HasError())
	assert.False(t, validate([]string{"a", "b"}).HasError())
}

func TestObservabilityReservedDirectoryPath(t *testing.T) {
	for _, path := range []string{"~templates", "~users", "team/~users", "~observability/homepage"} {
		assert.Equal(t, path, observabilityReservedDirectoryPath(path))
	}
	assert.Empty(t, observabilityReservedDirectoryPath("~organization/platform/dashboards"))
}

func TestResourceObservabilityDirectoryRejectsUnoccupiedCreateAndUpdate(t *testing.T) {
	store := newDirectoryAPIStore()
	managed, resourceSchema := configuredObservabilityDirectoryResource(t, store)
	pathValue := types.StringValue("~organization/platform/dashboards")
	emptyTemplates := types.ListValueMust(types.StringType, nil)

	plan := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, plan.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringUnknown(),
		Path:      pathValue,
		Templates: emptyTemplates,
		Pinned:    types.BoolValue(false),
	}).HasError())
	var validation resource.ValidateConfigResponse
	managed.ValidateConfig(t.Context(), resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: resourceSchema, Raw: plan.Raw},
	}, &validation)
	require.False(t, validation.Diagnostics.HasError(), validation.Diagnostics)

	create := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &create)
	require.True(t, create.Diagnostics.HasError())
	assert.Equal(t, "Unoccupied directory", create.Diagnostics.Errors()[0].Summary())

	store.entries[pathValue.ValueString()] = &directory.Entry{Path: pathValue.ValueString(), Pinned: true}
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        pathValue,
		Path:      pathValue,
		Templates: emptyTemplates,
		Pinned:    types.BoolValue(true),
	}).HasError())
	update := resource.UpdateResponse{State: state}
	managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &update)
	require.True(t, update.Diagnostics.HasError())
	assert.Equal(t, "Unoccupied directory", update.Diagnostics.Errors()[0].Summary())
	store.mu.Lock()
	entry := store.entries[pathValue.ValueString()]
	store.mu.Unlock()
	assert.True(t, entry.Pinned)
}

func TestResourceObservabilityDirectoryAllowsUnpinnedParentWithChild(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	store := newDirectoryAPIStore()
	store.entries[directoryPath] = &directory.Entry{
		Path:     directoryPath,
		Pinned:   true,
		Children: []string{"/v2/directory/~organization/platform/dashboards/team"},
	}
	managed, resourceSchema := configuredObservabilityDirectoryResource(t, store)
	planModel := observabilityDirectoryModel{
		ID:        types.StringValue(directoryPath),
		Path:      types.StringValue(directoryPath),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(false),
	}
	plan := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, plan.Set(t.Context(), planModel).HasError())
	var validation resource.ValidateConfigResponse
	managed.ValidateConfig(t.Context(), resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: resourceSchema, Raw: plan.Raw},
	}, &validation)
	require.False(t, validation.Diagnostics.HasError(), validation.Diagnostics)

	stateModel := planModel
	stateModel.Pinned = types.BoolValue(true)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), stateModel).HasError())
	update := resource.UpdateResponse{State: state}
	managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &update)
	require.False(t, update.Diagnostics.HasError(), update.Diagnostics)

	read := resource.ReadResponse{State: update.State}
	managed.Read(t.Context(), resource.ReadRequest{State: update.State}, &read)
	require.False(t, read.Diagnostics.HasError(), read.Diagnostics)
	require.False(t, read.State.Raw.IsNull(), "parent with a child remains occupied")
	var actual observabilityDirectoryModel
	require.False(t, read.State.Get(t.Context(), &actual).HasError())
	assert.False(t, actual.Pinned.ValueBool())
}

func TestResourceObservabilityDirectoryCreateRefusesExistingEntry(t *testing.T) {
	store := newDirectoryAPIStore()
	store.entries["~organization/platform/dashboards"] = &directory.Entry{
		Path:      "~organization/platform/dashboards",
		Templates: []string{"/v2/template/existing"},
		Pinned:    true,
	}
	managed, resourceSchema := configuredObservabilityDirectoryResource(t, store)

	plan := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, plan.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringUnknown(),
		Path:      types.StringValue("~organization/platform/dashboards"),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}).HasError())
	response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Directory already exists", response.Diagnostics.Errors()[0].Summary())
	store.mu.Lock()
	entry := store.entries["~organization/platform/dashboards"]
	store.mu.Unlock()
	assert.Equal(t, []string{"/v2/template/existing"}, entry.Templates)
	assert.True(t, entry.Pinned)
}

func TestResourceObservabilityDirectoryCreateRejectsUnexpectedLookup(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	for name, test := range map[string]struct {
		status int
		entry  *directory.Entry
		want   string
	}{
		"missing entry": {want: "Directory API returned no directory entry"},
		"different path": {
			entry: &directory.Entry{Path: "~organization/platform/other", Pinned: true},
			want:  "Directory API returned a different logical path",
		},
		"server error":         {status: http.StatusInternalServerError},
		"unexpected not found": {status: http.StatusNotFound, want: "HTTP 404"},
	} {
		t.Run(name, func(t *testing.T) {
			handlers := newDirectoryAPIStore().handlers()
			handlers["GET /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.status != 0 {
					http.Error(w, "lookup failed", test.status)
					return
				}
				assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: test.entry}))
			})
			handlers["PATCH /v2/directory/{path...}"] = http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Error("unexpected directory PATCH")
			})
			managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
			plan := tfsdk.Plan{Schema: resourceSchema}
			require.False(t, plan.Set(t.Context(), observabilityDirectoryModel{
				ID:        types.StringUnknown(),
				Path:      types.StringValue(directoryPath),
				Templates: types.ListValueMust(types.StringType, nil),
				Pinned:    types.BoolValue(true),
			}).HasError())

			response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
			managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
			require.True(t, response.Diagnostics.HasError())
			if test.want != "" {
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), test.want)
			}
		})
	}
}

func TestResourceObservabilityDirectoryCreateRejectsResolvedReservedPath(t *testing.T) {
	store := newDirectoryAPIStore()
	managed, resourceSchema := configuredObservabilityDirectoryResource(t, store)

	// A path that was unknown during ValidateConfig is known in the apply plan.
	plan := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, plan.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringUnknown(),
		Path:      types.StringValue("~templates"),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}).HasError())
	response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
	managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Reserved directory path", response.Diagnostics.Errors()[0].Summary())
	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Empty(t, store.entries)
}

func TestResourceObservabilityDirectoryReadRemovesMissingEntry(t *testing.T) {
	store := newDirectoryAPIStore()
	managed, resourceSchema := configuredObservabilityDirectoryResource(t, store)

	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringValue("~organization/platform/dashboards"),
		Path:      types.StringValue("~organization/platform/dashboards"),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(false),
	}).HasError())
	response := resource.ReadResponse{State: state}
	managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)

	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	assert.True(t, response.State.Raw.IsNull(), "missing directory must be removed from Terraform state")
}

func TestResourceObservabilityDirectoryReadReportsUnexpectedNotFound(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := newDirectoryAPIStore().handlers()
	handlers["GET /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "directory endpoint unavailable", http.StatusNotFound)
	})
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringValue(directoryPath),
		Path:      types.StringValue(directoryPath),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}).HasError())
	response := resource.ReadResponse{State: state}
	managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "HTTP 404")
	assert.Equal(t, state.Raw, response.State.Raw)
}

func TestResourceObservabilityDirectoryUpdateReportsUnexpectedNotFound(t *testing.T) {
	store := newDirectoryAPIStore()
	handlers := store.handlers()
	handlers["PATCH /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "directory entry not found", http.StatusNotFound)
	})
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)

	model := observabilityDirectoryModel{
		ID:        types.StringValue("~organization/platform/dashboards"),
		Path:      types.StringValue("~organization/platform/dashboards"),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}
	plan := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, plan.Set(t.Context(), model).HasError())
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), model).HasError())
	response := resource.UpdateResponse{State: state}
	managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Error updating directory", response.Diagnostics.Errors()[0].Summary())
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "HTTP 404")
	assert.NotContains(t, response.Diagnostics.Errors()[0].Detail(), "retry after refreshing")
	assert.Equal(t, state.Raw, response.State.Raw)
}

func TestResourceObservabilityDirectoryDeleteIgnoresMissingEntry(t *testing.T) {
	store := newDirectoryAPIStore()
	managed, resourceSchema := configuredObservabilityDirectoryResource(t, store)

	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringValue("~organization/platform/dashboards"),
		Path:      types.StringValue("~organization/platform/dashboards"),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(false),
	}).HasError())
	response := resource.DeleteResponse{State: state}
	managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)

	assert.False(t, response.Diagnostics.HasError(), response.Diagnostics)
}

func TestResourceObservabilityDirectoryDeleteReportsUnexpectedNotFound(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	for name, failingMethod := range map[string]string{
		"lookup": "GET /v2/directory/{path...}",
		"delete": "DELETE /v2/directory/{path...}",
	} {
		t.Run(name, func(t *testing.T) {
			store := newDirectoryAPIStore()
			store.entries[directoryPath] = &directory.Entry{Path: directoryPath, Pinned: true}
			handlers := store.handlers()
			handlers[failingMethod] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "directory endpoint unavailable", http.StatusNotFound)
			})
			managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
			state := tfsdk.State{Schema: resourceSchema}
			require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
				ID:        types.StringValue(directoryPath),
				Path:      types.StringValue(directoryPath),
				Templates: types.ListValueMust(types.StringType, nil),
				Pinned:    types.BoolValue(true),
			}).HasError())
			response := resource.DeleteResponse{State: state}
			managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)

			require.True(t, response.Diagnostics.HasError())
			assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "HTTP 404")
			store.mu.Lock()
			_, exists := store.entries[directoryPath]
			store.mu.Unlock()
			assert.True(t, exists)
		})
	}
}

func TestResourceObservabilityDirectoryDeleteRejectsDifferentPath(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := newDirectoryAPIStore().handlers()
	handlers["GET /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{
			Path: "~organization/platform/other", Pinned: true,
		}}))
	})
	handlers["DELETE /v2/directory/{path...}"] = http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected directory DELETE")
	})
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringValue(directoryPath),
		Path:      types.StringValue(directoryPath),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}).HasError())

	response := resource.DeleteResponse{State: state}
	managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)
	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Refusing to delete directory", response.Diagnostics.Errors()[0].Summary())
}

func TestResourceObservabilityDirectoryReadRejectsUnsupportedTemplateReference(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := newDirectoryAPIStore().handlers()
	handlers["GET /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{
			Path: directoryPath, Templates: []string{"/v3/templates/other"},
		}}))
	})
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringValue(directoryPath),
		Path:      types.StringValue(directoryPath),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}).HasError())

	response := resource.ReadResponse{State: state}
	managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)
	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Invalid Directory Template reference", response.Diagnostics.Errors()[0].Summary())
	assert.Equal(t, state.Raw, response.State.Raw)
}

func configuredObservabilityDirectoryResource(t *testing.T, store *directoryAPIStore) (*observabilityDirectoryResource, schema.Schema) {
	t.Helper()
	return configuredObservabilityDirectoryResourceWithHandlers(t, store.handlers())
}

func configuredObservabilityDirectoryResourceWithHandlers(t *testing.T, handlers map[string]http.Handler) (*observabilityDirectoryResource, schema.Schema) {
	t.Helper()
	mockProvider := fwtest.NewMock(t, handlers)
	var providerResponse provider.ConfigureResponse
	mockProvider.Configure(t.Context(), provider.ConfigureRequest{}, &providerResponse)

	managed := NewResourceObservabilityDirectory().(*observabilityDirectoryResource)
	var configureResponse resource.ConfigureResponse
	managed.Configure(t.Context(), resource.ConfigureRequest{ProviderData: providerResponse.ResourceData}, &configureResponse)
	require.False(t, configureResponse.Diagnostics.HasError(), configureResponse.Diagnostics)

	var schemaResponse resource.SchemaResponse
	managed.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	return managed, schemaResponse.Schema
}

func TestObservabilityDirectoryDeleteSafeguards(t *testing.T) {
	tests := map[string]struct {
		entry       directory.Entry
		wantError   string
		wantWarning string
		wantDelete  bool
	}{
		"empty entry": {
			entry: directory.Entry{Path: "~organization/platform/dashboards"},
		},
		"pinned empty entry": {
			entry:      directory.Entry{Path: "~organization/platform/dashboards", Pinned: true},
			wantDelete: true,
		},
		"template membership": {
			entry:       directory.Entry{Path: "~organization/platform/dashboards", Templates: []string{"/v2/template/dashboard-id"}},
			wantWarning: "Template(s)",
			wantDelete:  true,
		},
		"child directory": {
			entry:     directory.Entry{Path: "~organization/platform/dashboards", Children: []string{"child"}},
			wantError: "managed by the service or contains child directories",
		},
		"identity entry": {
			entry:     directory.Entry{Path: "~organization/platform/dashboards", Identity: true},
			wantError: "managed by the service or contains child directories",
		},
		"canonical entry": {
			entry:     directory.Entry{Path: "~organization/platform/dashboards", Canonical: true},
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

			managed, resourceSchema := configuredObservabilityDirectoryResource(t, store)
			state := tfsdk.State{Schema: resourceSchema}
			model, diags := observabilityDirectoryModelFromEntry(t.Context(), test.entry.Path, &test.entry)
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
	const initialConfig = `resource "signalfx_observability_directory" "test" {
  path = "~organization/platform/dashboards"
  templates = ["dashboard-a", "dashboard-b"]
  pinned = true
}`
	const updatedConfig = `resource "signalfx_observability_directory" "test" {
  path = "~organization/platform/dashboards"
  templates = ["dashboard-b", "dashboard-c"]
  pinned = false
}`
	const emptyConfig = `resource "signalfx_observability_directory" "test" {
  path      = "~organization/platform/dashboards"
  templates = []
  pinned    = true
}`

	testresource.UnitTest(
		t,
		testresource.TestCase{
			IsUnitTest: true,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.SkipBelow(tfversion.Version1_5_0),
			},
			ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
				t,
				store.handlers(),
				fwtest.WithMockResources(NewResourceObservabilityDirectory),
			),
			Steps: []testresource.TestStep{
				{
					Config: initialConfig,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "id", "~organization/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "path", "~organization/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.0", "dashboard-a"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.1", "dashboard-b"),
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
					Config: updatedConfig,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "id", "~organization/platform/dashboards"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.0", "dashboard-b"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.1", "dashboard-c"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "false"),
					),
				},
				{
					PreConfig: func() {
						store.mu.Lock()
						assert.Equal(t, []string{
							"/v2/template/dashboard-b",
							"/v2/template/dashboard-c",
						}, store.entries["~organization/platform/dashboards"].Templates)
						store.entries["~organization/platform/dashboards"].Templates = []string{
							"/v2/template/dashboard-b",
							"/v2/template/dashboard-c",
							"/v2/template/ui-added",
						}
						store.mu.Unlock()
					},
					Config: updatedConfig,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.0", "dashboard-b"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.1", "dashboard-c"),
					),
				},
				{
					Config: emptyConfig,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "0"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "pinned", "true"),
					),
				},
			},
		},
	)
}

// directoryAPIStore is a minimal in-memory fake of the Directory API used by
// the Directory resource lifecycle tests.
type directoryAPIStore struct {
	mu      sync.Mutex
	entries map[string]*directory.Entry
}

func newDirectoryAPIStore() *directoryAPIStore {
	return &directoryAPIStore{entries: make(map[string]*directory.Entry)}
}

func (s *directoryAPIStore) handlers() map[string]http.Handler {
	return map[string]http.Handler{
		"GET /v2/directory/{path...}":    http.HandlerFunc(s.read),
		"PATCH /v2/directory/{path...}":  http.HandlerFunc(s.patch),
		"DELETE /v2/directory/{path...}": http.HandlerFunc(s.delete),
	}
}

func (s *directoryAPIStore) read(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	entry, ok := s.entries[r.PathValue("path")]
	s.mu.Unlock()
	if !ok {
		// The service synthesizes an unoccupied entry for an absent path.
		entry = &directory.Entry{Path: r.PathValue("path")}
	}
	_ = json.NewEncoder(w).Encode(directory.Result{Data: entry})
}

func (s *directoryAPIStore) patch(w http.ResponseWriter, r *http.Request) {
	var patch directory.PatchDirectoryEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path := r.PathValue("path")
	s.mu.Lock()
	entry, ok := s.entries[path]
	if !ok {
		entry = &directory.Entry{Path: path}
		s.entries[path] = entry
	}
	if patch.Pinned != nil {
		entry.Pinned = *patch.Pinned
	}
	if patch.Templates != nil {
		entry.Templates = patch.Templates
	}
	if !observabilityDirectoryEntryOccupied(entry) {
		delete(s.entries, path)
	}
	s.mu.Unlock()

	_ = json.NewEncoder(w).Encode(directory.Result{Data: entry})
}

func (s *directoryAPIStore) delete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	delete(s.entries, r.PathValue("path"))
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
