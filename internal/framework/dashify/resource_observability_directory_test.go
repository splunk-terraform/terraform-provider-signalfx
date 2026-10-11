// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	var schemaResponse resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	pathAttribute, ok := schemaResponse.Schema.Attributes["path"].(schema.StringAttribute)
	require.True(t, ok)
	require.Len(t, pathAttribute.PlanModifiers, 1)
	assert.IsType(t, stringplanmodifier.RequiresReplace(), pathAttribute.PlanModifiers[0])
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, observabilityDirectoryModel{
		Templates: types.ListNull(types.StringType),
	}))
}

func TestResourceObservabilityDirectoryPlanRejectsInvalidPath(t *testing.T) {
	for _, test := range []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: "at least 1"},
		{name: "leading slash", path: "/~organization/platform", want: "decoded logical Directory path"},
		{name: "trailing slash", path: "~organization/platform/", want: "decoded logical Directory path"},
		{name: "empty segment", path: "~organization//platform", want: "decoded logical Directory path"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true,
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t, nil, fwtest.WithMockResources(NewResourceObservabilityDirectory),
				),
				Steps: []testresource.TestStep{{
					Config: fmt.Sprintf(`resource "signalfx_observability_directory" "test" {
  path = %q
}`, test.path),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(test.want),
				}},
			})
		})
	}
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

	patch, diags = observabilityDirectoryPatch(
		t.Context(),
		types.BoolNull(),
		types.ListNull(types.StringType),
	)
	require.False(t, diags.HasError(), diags)
	require.NotNil(t, patch.Pinned)
	assert.True(t, *patch.Pinned)
	assert.Nil(t, patch.Templates)

	invalid := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("invalid/id")})
	patch, diags = observabilityDirectoryPatch(t.Context(), types.BoolValue(true), invalid)
	assert.Nil(t, patch)
	require.True(t, diags.HasError())
	assert.Equal(t, "Invalid Template ID", diags.Errors()[0].Summary())
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
	for _, path := range []string{
		"~demo/team", "~local/team", "~signalview/team", "~templates/team",
		"~users", "team/~users", "~observability/homepage",
	} {
		assert.Equal(t, path, observabilityReservedDirectoryPath(path))
	}
	for _, path := range []string{
		"~demo-team", "~local-team", "~signalview-team", "~templates-team",
		"team/~users-extra", "~organization/platform/dashboards",
		// A user's own namespace root is their writable space, not reserved.
		"~users/example@example.com",
	} {
		assert.Empty(t, observabilityReservedDirectoryPath(path))
	}
}

func TestResourceObservabilityDirectoryRejectsReservedPathDuringValidation(t *testing.T) {
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, nil)
	config := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, config.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringUnknown(),
		Path:      types.StringValue("~templates"),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}).HasError())

	var response resource.ValidateConfigResponse
	managed.ValidateConfig(t.Context(), resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: resourceSchema, Raw: config.Raw},
	}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Reserved directory path", response.Diagnostics.Errors()[0].Summary())
}

func TestResourceObservabilityDirectoryRejectsUnoccupiedCreateAndUpdate(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	pathValue := types.StringValue(directoryPath)
	emptyTemplates := types.ListValueMust(types.StringType, nil)
	planModel := observabilityDirectoryModel{
		ID:        types.StringUnknown(),
		Path:      pathValue,
		Templates: emptyTemplates,
		Pinned:    types.BoolValue(false),
	}

	t.Run("create", func(t *testing.T) {
		handlers := map[string]http.Handler{
			"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{Path: directoryPath}}))
			}),
			"PATCH /v2/directory/{path...}": http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("unexpected directory PATCH")
			}),
		}
		managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
		plan := tfsdk.Plan{Schema: resourceSchema}
		require.False(t, plan.Set(t.Context(), planModel).HasError())
		response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
		managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
		require.True(t, response.Diagnostics.HasError())
		assert.Equal(t, "Unoccupied directory", response.Diagnostics.Errors()[0].Summary())
	})

	t.Run("update", func(t *testing.T) {
		handlers := map[string]http.Handler{
			"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{Path: directoryPath, Pinned: true}}))
			}),
			"PATCH /v2/directory/{path...}": http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("unexpected directory PATCH")
			}),
		}
		managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
		plan := tfsdk.Plan{Schema: resourceSchema}
		require.False(t, plan.Set(t.Context(), planModel).HasError())
		state := tfsdk.State{Schema: resourceSchema}
		require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
			ID: pathValue, Path: pathValue, Templates: emptyTemplates, Pinned: types.BoolValue(true),
		}).HasError())
		response := resource.UpdateResponse{State: state}
		managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)
		require.True(t, response.Diagnostics.HasError())
		assert.Equal(t, "Unoccupied directory", response.Diagnostics.Errors()[0].Summary())
	})
}

func TestResourceObservabilityDirectoryPlanRejectsUnoccupiedCreate(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, nil, fwtest.WithMockResources(NewResourceObservabilityDirectory),
		),
		Steps: []testresource.TestStep{{
			Config: `resource "signalfx_observability_directory" "test" {
  path      = "~organization/platform/dashboards"
  templates = []
  pinned    = false
}`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(observabilityDirectoryUnoccupiedSummary),
		}},
	})
}

func TestResourceObservabilityDirectoryPlanAllowsUnpinnedParentImport(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{
				Path:     directoryPath,
				Children: []string{"team"},
			}}))
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityDirectory),
		),
		Steps: []testresource.TestStep{{
			Config: `
import {
  to = signalfx_observability_directory.test
  id = "~organization/platform/dashboards"
}
resource "signalfx_observability_directory" "test" {
  path      = "~organization/platform/dashboards"
  templates = []
  pinned    = false
}`,
			PlanOnly: true,
		}},
	})
}

func TestResourceObservabilityDirectoryAllowsUnpinnedParentWithChild(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	children := []string{"/v2/directory/~organization/platform/dashboards/team"}
	entryFixture := directory.Entry{Path: directoryPath, Pinned: true, Children: children}
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &entryFixture}))
		}),
		"PATCH /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var patch directory.PatchDirectoryEntryRequest
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Errorf("decode directory patch: %v", err)
				http.Error(w, "invalid patch", http.StatusBadRequest)
				return
			}
			if patch.Pinned == nil {
				t.Error("directory patch omitted pinned")
				http.Error(w, "missing pinned", http.StatusBadRequest)
				return
			}
			assert.False(t, *patch.Pinned)
			assert.Empty(t, patch.Templates)
			entryFixture = directory.Entry{Path: directoryPath, Children: children}
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &entryFixture}))
		}),
	}
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
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
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{
				Path: directoryPath, Templates: []string{"/v2/template/existing"}, Pinned: true,
			}}))
		}),
		"PATCH /v2/directory/{path...}": http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("unexpected directory PATCH")
		}),
	}
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
	assert.Equal(t, "Directory already exists", response.Diagnostics.Errors()[0].Summary())
}

func TestResourceObservabilityDirectoryCreateRejectsUnexpectedLookup(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	for name, test := range map[string]struct {
		status      int
		entry       *directory.Entry
		wantSummary string
		wantDetail  string
	}{
		"missing entry": {wantDetail: "Directory API returned no directory entry"},
		"different path": {
			entry:      &directory.Entry{Path: "~organization/platform/other", Pinned: true},
			wantDetail: "Directory API returned a different logical path",
		},
		"server error": {
			status: http.StatusInternalServerError, wantSummary: "status code 500", wantDetail: "lookup failed",
		},
		"unexpected not found": {status: http.StatusNotFound, wantDetail: "HTTP 404"},
	} {
		t.Run(name, func(t *testing.T) {
			handlers := map[string]http.Handler{
				"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if test.status != 0 {
						http.Error(w, "lookup failed", test.status)
						return
					}
					assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: test.entry}))
				}),
				"PATCH /v2/directory/{path...}": http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
					t.Error("unexpected directory PATCH")
				}),
			}
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
			if test.wantSummary != "" {
				assert.Contains(t, response.Diagnostics.Errors()[0].Summary(), test.wantSummary)
			}
			if test.wantDetail != "" {
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), test.wantDetail)
			}
		})
	}
}

func TestResourceObservabilityDirectoryCreateRejectsResolvedReservedPath(t *testing.T) {
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, nil)

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
}

func TestResourceObservabilityDirectoryRejectsIncompleteResponses(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	tests := map[string]struct {
		operation string
		entry     *directory.Entry
		want      string
	}{
		"create without entry": {operation: "create", want: "Error creating directory"},
		"create with different path": {
			operation: "create",
			entry:     &directory.Entry{Path: "~organization/platform/other", Pinned: true},
			want:      "Unexpected Directory path",
		},
		"read without entry":   {operation: "read", want: "Error reading directory"},
		"update without entry": {operation: "update", want: "Error updating directory"},
		"update with different path": {
			operation: "update",
			entry:     &directory.Entry{Path: "~organization/platform/other", Pinned: true},
			want:      "Unexpected Directory path",
		},
		"delete without entry": {operation: "delete", want: "Error checking directory"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			writeResult := func(w http.ResponseWriter, entry *directory.Entry) {
				assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: entry}))
			}
			handlers := map[string]http.Handler{}
			switch test.operation {
			case "create":
				handlers["GET /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeResult(w, &directory.Entry{Path: directoryPath})
				})
				handlers["PATCH /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeResult(w, test.entry)
				})
			case "read", "delete":
				handlers["GET /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeResult(w, test.entry)
				})
			case "update":
				handlers["PATCH /v2/directory/{path...}"] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeResult(w, test.entry)
				})
			}

			managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
			model := observabilityDirectoryModel{
				ID:        types.StringValue(directoryPath),
				Path:      types.StringValue(directoryPath),
				Templates: types.ListValueMust(types.StringType, nil),
				Pinned:    types.BoolValue(true),
			}
			state := tfsdk.State{Schema: resourceSchema}
			require.False(t, state.Set(t.Context(), model).HasError())

			var diagnostics diag.Diagnostics
			switch test.operation {
			case "create":
				model.ID = types.StringUnknown()
				plan := tfsdk.Plan{Schema: resourceSchema}
				require.False(t, plan.Set(t.Context(), model).HasError())
				response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
				managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
				diagnostics = response.Diagnostics
			case "read":
				response := resource.ReadResponse{State: state}
				managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)
				diagnostics = response.Diagnostics
			case "update":
				plan := tfsdk.Plan{Schema: resourceSchema}
				require.False(t, plan.Set(t.Context(), model).HasError())
				response := resource.UpdateResponse{State: state}
				managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)
				diagnostics = response.Diagnostics
			case "delete":
				response := resource.DeleteResponse{State: state}
				managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)
				diagnostics = response.Diagnostics
			}

			require.True(t, diagnostics.HasError())
			assert.Equal(t, test.want, diagnostics.Errors()[0].Summary())
		})
	}
}

func TestResourceObservabilityDirectoryReadRemovesMissingEntry(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{Path: directoryPath}}))
		}),
	}
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)

	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringValue(directoryPath),
		Path:      types.StringValue(directoryPath),
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
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "directory endpoint unavailable", http.StatusNotFound)
		}),
	}
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

func TestResourceObservabilityDirectoryReadRejectsMismatchedResponsePath(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{
				Path:   "~organization/platform/unrelated",
				Pinned: true,
			}}))
		}),
	}
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
	assert.Equal(t, "Unexpected Directory path", response.Diagnostics.Errors()[0].Summary())
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), `"~organization/platform/unrelated"`)
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), `"~organization/platform/dashboards"`)
	assert.Equal(t, state.Raw, response.State.Raw)
}

func TestResourceObservabilityDirectoryReportsErrorEnvelope(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"data":null,"errors":[{"code":"400","message":"invalid directory request"}],"includes":[]}`))
		}),
	}
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
	assert.Contains(t, response.Diagnostics.Errors()[0].Summary(), "status code 400")
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "invalid directory request")
}

func TestResourceObservabilityDirectoryUpdateReportsUnexpectedNotFound(t *testing.T) {
	handlers := map[string]http.Handler{
		"PATCH /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "directory entry not found", http.StatusNotFound)
		}),
	}
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

func TestResourceObservabilityDirectoryWriteFailures(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	base := observabilityDirectoryModel{
		ID:        types.StringValue(directoryPath),
		Path:      types.StringValue(directoryPath),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(true),
	}

	for name, test := range map[string]struct {
		operation   string
		model       observabilityDirectoryModel
		handlers    map[string]http.Handler
		wantSummary string
		wantDetail  string
	}{
		"create patch failure": {
			operation:   "create",
			model:       base,
			wantSummary: "status code 500",
			wantDetail:  "patch failed",
			handlers: map[string]http.Handler{
				"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{Path: directoryPath}}))
				}),
				"PATCH /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "patch failed", http.StatusInternalServerError)
				}),
			},
		},
		"update lookup failure": {
			operation:   "update",
			wantSummary: "status code 500",
			wantDetail:  "lookup failed",
			model: observabilityDirectoryModel{
				ID: base.ID, Path: base.Path, Templates: base.Templates, Pinned: types.BoolValue(false),
			},
			handlers: map[string]http.Handler{
				"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "lookup failed", http.StatusInternalServerError)
				}),
			},
		},
		"update invalid template": {
			operation:   "update",
			wantSummary: "Invalid Template ID",
			model: observabilityDirectoryModel{
				ID:   base.ID,
				Path: base.Path,
				Templates: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("invalid/id"),
				}),
				Pinned: types.BoolValue(true),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, test.handlers)
			state := tfsdk.State{Schema: resourceSchema}
			require.False(t, state.Set(t.Context(), base).HasError())
			plan := tfsdk.Plan{Schema: resourceSchema}
			planned := test.model
			if test.operation == "create" {
				planned.ID = types.StringUnknown()
			}
			require.False(t, plan.Set(t.Context(), planned).HasError())

			var diagnostics diag.Diagnostics
			if test.operation == "create" {
				response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema}}
				managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
				diagnostics = response.Diagnostics
			} else {
				response := resource.UpdateResponse{State: state}
				managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)
				diagnostics = response.Diagnostics
			}

			require.True(t, diagnostics.HasError())
			assert.Contains(t, diagnostics.Errors()[0].Summary(), test.wantSummary)
			if test.wantDetail != "" {
				assert.Contains(t, diagnostics.Errors()[0].Detail(), test.wantDetail)
			}
		})
	}
}

func TestResourceObservabilityDirectoryDeleteIgnoresMissingEntry(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{Path: directoryPath}}))
		}),
		"DELETE /v2/directory/{path...}": http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("unexpected directory DELETE")
		}),
	}
	managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)

	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityDirectoryModel{
		ID:        types.StringValue(directoryPath),
		Path:      types.StringValue(directoryPath),
		Templates: types.ListValueMust(types.StringType, nil),
		Pinned:    types.BoolValue(false),
	}).HasError())
	response := resource.DeleteResponse{State: state}
	managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)

	assert.False(t, response.Diagnostics.HasError(), response.Diagnostics)
}

func TestResourceObservabilityDirectoryDeleteReportsUnexpectedNotFound(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	for name, test := range map[string]struct {
		failingMethod string
		wantSummary   string
	}{
		"lookup": {failingMethod: "GET /v2/directory/{path...}", wantSummary: "Error checking directory"},
		"delete": {failingMethod: "DELETE /v2/directory/{path...}", wantSummary: "Error deleting directory"},
	} {
		t.Run(name, func(t *testing.T) {
			handlers := map[string]http.Handler{
				"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{Path: directoryPath, Pinned: true}}))
				}),
				"DELETE /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}),
			}
			handlers[test.failingMethod] = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
			assert.Equal(t, test.wantSummary, response.Diagnostics.Errors()[0].Summary())
			assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "HTTP 404")
		})
	}
}

func TestResourceObservabilityDirectoryDeleteRejectsDifferentPath(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{
				Path: "~organization/platform/other", Pinned: true,
			}}))
		}),
		"DELETE /v2/directory/{path...}": http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("unexpected directory DELETE")
		}),
	}
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
	assert.Equal(t, "Unexpected Directory path", response.Diagnostics.Errors()[0].Summary())
}

func TestResourceObservabilityDirectoryReadRejectsUnsupportedTemplateReference(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{
				Path: directoryPath, Templates: []string{"/v3/templates/other"},
			}}))
		}),
	}
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
			deleteCalled := false
			handlers := map[string]http.Handler{
				"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/v2/directory/"+test.entry.Path, r.URL.EscapedPath())
					assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &test.entry}))
				}),
				"DELETE /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/v2/directory/"+test.entry.Path, r.URL.EscapedPath())
					deleteCalled = true
					w.WriteHeader(http.StatusNoContent)
				}),
			}

			managed, resourceSchema := configuredObservabilityDirectoryResourceWithHandlers(t, handlers)
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

			assert.Equal(t, test.wantDelete, deleteCalled)
		})
	}
}

func TestResourceObservabilityDirectoryEncodedPathLifecycleAndImport(t *testing.T) {
	for name, test := range map[string]struct {
		path        string
		escapedPath string
		testImport  bool
	}{
		"space": {
			path: "~organization/platform/My Charts", escapedPath: "/v2/directory/~organization/platform/My+Charts",
		},
		"literal plus": {
			path: "~organization/platform/A+B", escapedPath: "/v2/directory/~organization/platform/A%2BB", testImport: true,
		},
		"unicode": {
			path: "~organization/platform/zażółć/東京", escapedPath: "/v2/directory/~organization/platform/za%C5%BC%C3%B3%C5%82%C4%87/%E6%9D%B1%E4%BA%AC",
		},
	} {
		t.Run(name, func(t *testing.T) {
			directoryPath := test.path
			config := fmt.Sprintf(`resource "signalfx_observability_directory" "test" {
  path   = %q
  pinned = true
}`, directoryPath)
			entryFixture := directory.Entry{Path: directoryPath}
			createHandlers := map[string]http.Handler{
				"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, test.escapedPath, r.URL.EscapedPath())
					assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &entryFixture}))
				}),
				"PATCH /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, test.escapedPath, r.URL.EscapedPath())
					entryFixture = directory.Entry{Path: directoryPath, Pinned: true}
					assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &entryFixture}))
				}),
				"DELETE /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, test.escapedPath, r.URL.EscapedPath())
					entryFixture = directory.Entry{Path: directoryPath}
					w.WriteHeader(http.StatusNoContent)
				}),
			}
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true,
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t, createHandlers, fwtest.WithMockResources(NewResourceObservabilityDirectory),
				),
				Steps: []testresource.TestStep{{
					Config: config,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "id", directoryPath),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "path", directoryPath),
					),
				}},
			})

			if !test.testImport {
				return
			}
			importHandlers := map[string]http.Handler{
				"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, test.escapedPath, r.URL.EscapedPath())
					assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &directory.Entry{Path: directoryPath, Pinned: true}}))
				}),
			}
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_5_0),
				},
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t, importHandlers, fwtest.WithMockResources(NewResourceObservabilityDirectory),
				),
				Steps: []testresource.TestStep{{
					Config: fmt.Sprintf(`import {
  to = signalfx_observability_directory.test
  id = %q
}
%s`, directoryPath, config),
					PlanOnly: true,
				}},
			})
		})
	}
}

func TestResourceObservabilityDirectoryLifecycleAndGeneratedConfig(t *testing.T) {
	const directoryPath = "~organization/platform/dashboards"
	const requestPath = "/v2/directory/~organization/platform/dashboards"
	entryFixture := directory.Entry{Path: directoryPath}
	handlers := map[string]http.Handler{
		"GET /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, requestPath, r.URL.EscapedPath())
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &entryFixture}))
		}),
		"PATCH /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, requestPath, r.URL.EscapedPath())
			var patch directory.PatchDirectoryEntryRequest
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Errorf("decode directory patch: %v", err)
				http.Error(w, "invalid patch", http.StatusBadRequest)
				return
			}
			if patch.Pinned == nil {
				t.Error("directory patch omitted pinned")
				http.Error(w, "missing pinned", http.StatusBadRequest)
				return
			}
			entryFixture = directory.Entry{
				Path: directoryPath, Templates: patch.Templates, Pinned: *patch.Pinned,
			}
			assert.NoError(t, json.NewEncoder(w).Encode(directory.Result{Data: &entryFixture}))
		}),
		"DELETE /v2/directory/{path...}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, requestPath, r.URL.EscapedPath())
			entryFixture = directory.Entry{Path: directoryPath}
			w.WriteHeader(http.StatusNoContent)
		}),
	}
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
				handlers,
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
					PreConfig: func() {
						entryFixture = directory.Entry{
							Path: directoryPath, Templates: []string{"/v2/template/dashboard-a", "/v2/template/dashboard-b"}, Pinned: true,
						}
					},
					ResourceName:    "signalfx_observability_directory.test",
					ImportState:     true,
					ImportStateKind: testresource.ImportBlockWithID,
					GenerateConfig:  true,
				},
				{
					PreConfig: func() {
						entryFixture = directory.Entry{
							Path: directoryPath, Templates: []string{"/v2/template/dashboard-a", "/v2/template/dashboard-b"}, Pinned: true,
						}
					},
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
						entryFixture = directory.Entry{
							Path: directoryPath,
							Templates: []string{
								"/v2/template/dashboard-b",
								"/v2/template/dashboard-c",
								"/v2/template/ui-added",
							},
						}
					},
					Config: updatedConfig,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.0", "dashboard-b"),
						testresource.TestCheckResourceAttr("signalfx_observability_directory.test", "templates.1", "dashboard-c"),
					),
				},
				{
					PreConfig: func() {
						assert.Equal(t, []string{"/v2/template/dashboard-b", "/v2/template/dashboard-c"}, entryFixture.Templates)
						entryFixture = directory.Entry{
							Path: directoryPath, Templates: []string{"/v2/template/dashboard-b", "/v2/template/dashboard-c"}, Pinned: false,
						}
					},
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
