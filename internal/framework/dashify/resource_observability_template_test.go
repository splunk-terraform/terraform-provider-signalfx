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

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

const observabilityTemplateInitialConfig = `resource "signalfx_observability_template" "test" {
  title        = "Request rate"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = []
  })
}`

const observabilityTemplateUpdatedConfig = `resource "signalfx_observability_template" "test" {
  title        = "Request rate (updated)"
  root_element = "Chart"

  spec = jsonencode({
    "<Chart>" = []
  })

  metadata = {
    imports = ["shared"]
  }
}`

func TestResourceObservabilityTemplateMetadataAndSchema(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityTemplate()
	var metadataResponse resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &metadataResponse)
	assert.Equal(t, "signalfx_observability_template", metadataResponse.TypeName)
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	assert.True(t, schemaResponse.Schema.Attributes["title"].IsRequired())
	rootElement, ok := schemaResponse.Schema.Attributes["root_element"].(schema.StringAttribute)
	require.True(t, ok)
	assert.True(t, rootElement.IsRequired())
	require.Len(t, rootElement.PlanModifiers, 1)
	assert.IsType(t, stringplanmodifier.RequiresReplace(), rootElement.PlanModifiers[0])
	assert.True(t, schemaResponse.Schema.Attributes["spec"].IsRequired())
	metadataAttribute, ok := schemaResponse.Schema.Attributes["metadata"].(schema.SingleNestedAttribute)
	require.True(t, ok)
	assert.True(t, metadataAttribute.IsOptional())
	assert.Empty(t, metadataAttribute.Validators)
	assert.NotContains(t, metadataAttribute.Attributes, "root_element")
	assert.True(t, metadataAttribute.Attributes["imports"].IsOptional())
	datasource, ok := metadataAttribute.Attributes["datasource"].(schema.SingleNestedAttribute)
	require.True(t, ok)
	assert.True(t, datasource.IsOptional())
	assert.True(t, datasource.Attributes["type"].IsOptional())
	assert.True(t, datasource.Attributes["program_text"].IsOptional())
	assert.True(t, datasource.Attributes["slo_id"].IsOptional())
	assert.NotContains(t, schemaResponse.Schema.Attributes, "template_contents")
	assert.NotContains(t, schemaResponse.Schema.Attributes, "imports")
}

func TestResourceObservabilityTemplateImportsValidation(t *testing.T) {
	t.Parallel()

	var schemaResponse resource.SchemaResponse
	NewResourceObservabilityTemplate().Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	metadataAttribute := schemaResponse.Schema.Attributes["metadata"].(schema.SingleNestedAttribute)
	importsAttr := metadataAttribute.Attributes["imports"].(schema.ListAttribute)

	validate := func(values []string) bool {
		list, diags := types.ListValueFrom(t.Context(), types.StringType, values)
		require.False(t, diags.HasError(), diags)
		var response validator.ListResponse
		for _, v := range importsAttr.Validators {
			v.ValidateList(t.Context(), validator.ListRequest{ConfigValue: list}, &response)
		}
		return response.Diagnostics.HasError()
	}

	assert.True(t, validate([]string{"a", "a"}))
	assert.True(t, validate([]string{""}))
	assert.True(t, validate([]string{"a/b"}))
	assert.True(t, validate([]string{"."}))
	assert.True(t, validate([]string{".."}))
	assert.True(t, validate([]string{"/v2/template/a"}))
	assert.False(t, validate([]string{"a", "b"}))
}

func TestResourceObservabilityTemplateRootElementValidation(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityTemplate()
	var schemaResponse resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResponse)
	attribute, ok := schemaResponse.Schema.Attributes["root_element"].(schema.StringAttribute)
	require.True(t, ok)
	require.Len(t, attribute.Validators, 1)

	for name, test := range map[string]struct {
		value       string
		expectError bool
	}{
		"chart":                  {value: "Chart"},
		"dashboard":              {value: "Dashboard"},
		"arbitrary root element": {value: "FutureRootElement"},
		"empty":                  {value: "", expectError: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request := validator.StringRequest{
				Path:           path.Root("root_element"),
				PathExpression: path.MatchRoot("root_element"),
				ConfigValue:    types.StringValue(test.value),
			}
			response := validator.StringResponse{}
			attribute.Validators[0].ValidateString(context.Background(), request, &response)
			assert.Equal(t, test.expectError, response.Diagnostics.HasError())
		})
	}
}

func TestObservabilityTemplateDatasourceValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		root         string
		datasource   observabilityTemplateDatasourceModel
		expectErrors int
	}{
		"chart datasource": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservability)),
				ProgramText: types.StringValue("data('requests').publish()"),
				SLOID:       types.StringNull(),
			},
		},
		"chart program without type": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringNull(),
				ProgramText: types.StringValue("data('requests').publish()"),
				SLOID:       types.StringNull(),
			},
			expectErrors: 1,
		},
		"dashboard type without program": {
			root: "dashboard",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservability)),
				ProgramText: types.StringNull(),
				SLOID:       types.StringNull(),
			},
			expectErrors: 1,
		},
		"SLO datasource": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservabilitySLO)),
				ProgramText: types.StringValue("data('service.level').publish()"),
				SLOID:       types.StringValue("example-slo-id"),
			},
		},
		"SLO datasource without ID": {
			root: "Chart",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringValue(string(template.DatasourceTypeSplunkObservabilitySLO)),
				ProgramText: types.StringValue("data('service.level').publish()"),
				SLOID:       types.StringNull(),
			},
			expectErrors: 1,
		},
		"custom root follows extensible API validation": {
			root: "FutureRootElement",
			datasource: observabilityTemplateDatasourceModel{
				Type:        types.StringNull(),
				ProgramText: types.StringValue("future datasource contents"),
				SLOID:       types.StringNull(),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			response := resource.ValidateConfigResponse{}
			validateObservabilityTemplateDatasource(types.StringValue(test.root), test.datasource, &response)
			assert.Len(t, response.Diagnostics, test.expectErrors)
		})
	}
}

func TestResourceObservabilityTemplateValidatesConfiguredDatasource(t *testing.T) {
	managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, nil)
	config := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, config.Set(t.Context(), observabilityTemplateModel{
		ID:          types.StringUnknown(),
		Title:       types.StringValue("Request rate"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
		Metadata: &observabilityTemplateMetadataModel{
			Imports: types.ListNull(types.StringType),
			Datasource: &observabilityTemplateDatasourceModel{
				Type:        types.StringNull(),
				ProgramText: types.StringValue("data('requests').publish()"),
				SLOID:       types.StringNull(),
			},
		},
	}).HasError())

	var response resource.ValidateConfigResponse
	managed.ValidateConfig(t.Context(), resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: resourceSchema, Raw: config.Raw},
	}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Missing datasource type", response.Diagnostics.Errors()[0].Summary())
}

func TestObservabilityTemplateSpecValidation(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateObservabilityTemplateSpec(`{"<Chart>":[]}`))
	assert.Error(t, validateObservabilityTemplateSpec(`{"<Chart>":`))
	assert.Error(t, validateObservabilityTemplateSpec(`null`))
	assert.Error(t, validateObservabilityTemplateSpec(`[]`))
}

func TestResourceObservabilityTemplateRejectsInvalidConfiguration(t *testing.T) {
	for name, test := range map[string]struct {
		title string
		spec  string
		want  string
	}{
		"blank title": {title: "   ", spec: `{"<Chart>":[]}`, want: "title must contain at least one non-whitespace character"},
		"array spec":  {title: "Example", spec: `[]`, want: "spec must be a JSON object"},
	} {
		t.Run(name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true,
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t, nil, fwtest.WithMockResources(NewResourceObservabilityTemplate),
				),
				Steps: []testresource.TestStep{{
					Config: fmt.Sprintf(`resource "signalfx_observability_template" "test" {
  title        = %q
  root_element = "Chart"
  spec         = %q
}`, test.title, test.spec),
					ExpectError: regexp.MustCompile(test.want),
				}},
			})
		})
	}
}

func TestResourceObservabilityTemplateLifecycleAndGeneratedConfig(t *testing.T) {
	root := template.RootElementChart
	initial := template.Template{
		ID: "template-1", Type: observabilityTemplateRecordType, Title: "Request rate",
		Spec: json.RawMessage(`{"<Chart>":[]}`), Metadata: &template.Metadata{RootElement: &root},
	}
	updated := template.Template{
		ID: "template-1", Type: observabilityTemplateRecordType, Title: "Request rate (updated)",
		Spec: json.RawMessage(`{"<Chart>":[]}`), Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/shared"}},
	}
	managedUpdate := template.Template{
		ID: "template-1", Type: observabilityTemplateRecordType, Title: "Request rate (managed update)",
		Spec: json.RawMessage(`{"<Chart>":[]}`), Metadata: &template.Metadata{RootElement: &root},
	}
	readFixture := initial
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode template create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, observabilityTemplateRecordType, write.Type)
			assert.Equal(t, initial.Title, write.Title)
			assert.JSONEq(t, string(initial.Spec), string(write.Spec))
			assert.Equal(t, initial.Metadata.RootElement, write.Metadata.RootElement)
			assert.Empty(t, write.Metadata.Imports)
			assert.Nil(t, write.Metadata.Datasource)
			readFixture = template.Template{
				ID: "template-1", Type: write.Type, Title: write.Title, Spec: write.Spec,
				Metadata: &template.Metadata{RootElement: write.Metadata.RootElement, Imports: write.Metadata.Imports},
			}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &readFixture}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "template-1", r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &readFixture}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "template-1", r.PathValue("id"))
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode template update request: %v", err)
				http.Error(w, "invalid update request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, observabilityTemplateRecordType, write.Type)
			switch write.Title {
			case managedUpdate.Title:
				assert.Empty(t, write.Metadata.Imports)
			case updated.Title:
				assert.Equal(t, updated.Metadata.Imports, write.Metadata.Imports)
			default:
				t.Errorf("unexpected template update title %q", write.Title)
			}
			assert.JSONEq(t, string(updated.Spec), string(write.Spec))
			assert.Equal(t, updated.Metadata.RootElement, write.Metadata.RootElement)
			assert.Nil(t, write.Metadata.Datasource)
			readFixture = template.Template{
				ID: "template-1", Type: write.Type, Title: write.Title, Spec: write.Spec,
				Metadata: &template.Metadata{RootElement: write.Metadata.RootElement, Imports: write.Metadata.Imports},
			}
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &readFixture}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "template-1", r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}

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
				fwtest.WithMockResources(NewResourceObservabilityTemplate),
			),
			Steps: []testresource.TestStep{
				{
					Config: observabilityTemplateInitialConfig,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_template.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "title", "Request rate"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "root_element", "Chart"),
						testresource.TestCheckNoResourceAttr("signalfx_observability_template.test", "metadata"),
					),
				},
				{
					Config: `resource "signalfx_observability_template" "test" {
  title        = "Request rate (managed update)"
  root_element = "Chart"
  spec         = jsonencode({ "<Chart>" = [] })
}`,
					Check: testresource.TestCheckResourceAttr(
						"signalfx_observability_template.test", "title", managedUpdate.Title,
					),
				},
				{
					PreConfig:       func() { readFixture = initial },
					ResourceName:    "signalfx_observability_template.test",
					ImportState:     true,
					ImportStateKind: testresource.ImportBlockWithID,
					GenerateConfig:  true,
				},
				{
					PreConfig: func() { readFixture = initial },
					Config:    observabilityTemplateUpdatedConfig,
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_template.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "title", "Request rate (updated)"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "root_element", "Chart"),
						testresource.TestCheckResourceAttr("signalfx_observability_template.test", "metadata.imports.0", "shared"),
					),
				},
			},
		},
	)
}

func TestResourceObservabilityTemplateRecreatesAfterRemoteDelete(t *testing.T) {
	createdID := "template-1"
	root := template.RootElementChart
	var readFixture *template.Template
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode template create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			readFixture = &template.Template{
				ID: createdID, Type: write.Type, Title: write.Title, Spec: write.Spec,
				Metadata: &template.Metadata{RootElement: &root},
			}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: readFixture}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if readFixture == nil {
				http.Error(w, "template not found", http.StatusNotFound)
				return
			}
			assert.Equal(t, readFixture.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: readFixture}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			readFixture = nil
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{
			{Config: observabilityTemplateInitialConfig},
			{
				PreConfig: func() {
					createdID = "template-2"
					readFixture = nil
				},
				Config: observabilityTemplateInitialConfig,
				Check: testresource.TestCheckResourceAttr(
					"signalfx_observability_template.test", "id", "template-2",
				),
			},
		},
	})
}

func TestResourceObservabilityTemplateCreateReportsMissingEndpoint(t *testing.T) {
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "template endpoint not found", http.StatusNotFound)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{{
			Config:      observabilityTemplateInitialConfig,
			ExpectError: regexp.MustCompile("The Template API endpoint was not found"),
		}},
	})
}

func TestResourceObservabilityTemplateCreateRejectsMissingResponseID(t *testing.T) {
	root := template.RootElementChart
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &template.Template{
				Title:    "Request rate",
				Spec:     json.RawMessage(`{"<Chart>":[]}`),
				Metadata: &template.Metadata{RootElement: &root},
			}}))
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{{
			Config:      observabilityTemplateInitialConfig,
			ExpectError: regexp.MustCompile("template API returned a template record without an ID"),
		}},
	})
}

func TestResourceObservabilityTemplateUpdateReportsNotFound(t *testing.T) {
	root := template.RootElementChart
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &template.Template{
				ID: "template-1", Type: observabilityTemplateRecordType, Title: "Request rate",
				Spec: json.RawMessage(`{"<Chart>":[]}`), Metadata: &template.Metadata{RootElement: &root},
			}}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &template.Template{
				ID: "template-1", Type: observabilityTemplateRecordType, Title: "Request rate",
				Spec: json.RawMessage(`{"<Chart>":[]}`), Metadata: &template.Metadata{RootElement: &root},
			}}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "template not found", http.StatusNotFound)
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{
			{Config: observabilityTemplateInitialConfig},
			{
				Config:      observabilityTemplateUpdatedConfig,
				ExpectError: regexp.MustCompile("was not found during update"),
			},
		},
	})
}

func TestResourceObservabilityTemplateWriteFailures(t *testing.T) {
	base := observabilityTemplateModel{
		ID:          types.StringValue("template-1"),
		Title:       types.StringValue("Request rate"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
		Metadata: &observabilityTemplateMetadataModel{
			Imports: types.ListNull(types.StringType),
		},
	}

	for name, test := range map[string]struct {
		operation   string
		invalidSpec bool
		handlers    map[string]http.Handler
		wantSummary string
		wantDetail  string
	}{
		"create invalid specification": {
			operation: "create", invalidSpec: true, wantSummary: "Invalid template specification",
		},
		"create server error": {
			operation:   "create",
			wantSummary: "status code 500",
			wantDetail:  "create failed",
			handlers: map[string]http.Handler{
				"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "create failed", http.StatusInternalServerError)
				}),
			},
		},
		"update invalid specification": {
			operation: "update", invalidSpec: true, wantSummary: "Invalid template specification",
		},
		"update server error": {
			operation:   "update",
			wantSummary: "status code 500",
			wantDetail:  "update failed",
			handlers: map[string]http.Handler{
				"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					http.Error(w, "update failed", http.StatusInternalServerError)
				}),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, test.handlers)
			state := tfsdk.State{Schema: resourceSchema}
			require.False(t, state.Set(t.Context(), base).HasError())
			planned := base
			if test.invalidSpec {
				planned.Spec = jsontypes.NewNormalizedValue(`[]`)
			}
			if test.operation == "create" {
				planned.ID = types.StringUnknown()
			}
			plan := tfsdk.Plan{Schema: resourceSchema}
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

func TestResourceObservabilityTemplateReadRejectsMismatchedResponseID(t *testing.T) {
	root := template.RootElementChart
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "owned", r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &template.Template{
				ID:       "unrelated",
				Title:    "Unrelated",
				Spec:     json.RawMessage(`{"<Chart>":[]}`),
				Metadata: &template.Metadata{RootElement: &root},
			}}))
		}),
	}
	managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityTemplateModel{
		ID:          types.StringValue("owned"),
		Title:       types.StringValue("Owned"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}).HasError())

	response := resource.ReadResponse{State: state}
	managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Error reading template", response.Diagnostics.Errors()[0].Summary())
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), `record "unrelated"`)
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), `Template "owned"`)
	assert.Equal(t, state.Raw, response.State.Raw)
}

func TestResourceObservabilityTemplateReadRejectsMissingRecord(t *testing.T) {
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{}))
		}),
	}
	managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityTemplateModel{
		ID:          types.StringValue("owned"),
		Title:       types.StringValue("Owned"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}).HasError())

	response := resource.ReadResponse{State: state}
	managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Error reading template", response.Diagnostics.Errors()[0].Summary())
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "no template record")
	assert.Equal(t, state.Raw, response.State.Raw)
}

func TestResourceObservabilityTemplateReportsErrorEnvelope(t *testing.T) {
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"data":null,"errors":[{"code":"400","message":"invalid template request"}],"includes":[]}`))
		}),
	}
	managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityTemplateModel{
		ID:          types.StringValue("owned"),
		Title:       types.StringValue("Owned"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}).HasError())

	response := resource.ReadResponse{State: state}
	managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Contains(t, response.Diagnostics.Errors()[0].Summary(), "status code 400")
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "invalid template request")
}

func TestResourceObservabilityTemplateUpdateRejectsMissingResponseRecord(t *testing.T) {
	handlers := map[string]http.Handler{
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{}))
		}),
	}
	managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, handlers)
	model := observabilityTemplateModel{
		ID:          types.StringValue("template-1"),
		Title:       types.StringValue("Request rate"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
		Metadata: &observabilityTemplateMetadataModel{
			Imports: types.ListNull(types.StringType),
		},
	}
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), model).HasError())
	plan := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, plan.Set(t.Context(), model).HasError())

	response := resource.UpdateResponse{State: state}
	managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)

	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, "Error updating template", response.Diagnostics.Errors()[0].Summary())
	assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "no template record")
	assert.Equal(t, state.Raw, response.State.Raw)
}

func TestResourceObservabilityTemplateImportedUpdateRequiresMetadata(t *testing.T) {
	root := template.RootElementChart
	imported := template.Template{
		ID:       "imported",
		Type:     observabilityTemplateRecordType,
		Title:    "Imported",
		Spec:     json.RawMessage(`{"<Chart>":[]}`),
		Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/child"}},
	}
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, imported.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &imported}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, imported.ID, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	const importedConfig = `resource "signalfx_observability_template" "test" {
  title        = "Imported"
  root_element = "Chart"
  spec         = jsonencode({ "<Chart>" = [] })
}`
	const changedConfig = `resource "signalfx_observability_template" "test" {
  title        = "Changed"
  root_element = "Chart"
  spec         = jsonencode({ "<Chart>" = [] })
}`
	const computedTitleConfig = `resource "terraform_data" "title" {
  input = "Changed"
}

resource "signalfx_observability_template" "test" {
  title        = terraform_data.title.output
  root_element = "Chart"
  spec         = jsonencode({ "<Chart>" = [] })
}`
	const replacementConfig = `resource "signalfx_observability_template" "test" {
  title        = "Imported"
  root_element = "Dashboard"
  spec         = jsonencode({ "<Dashboard>" = [] })
}`

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{
			{
				Config: `
import {
  to = signalfx_observability_template.test
  id = "imported"
}
` + importedConfig,
			},
			{
				Config:   importedConfig,
				PlanOnly: true,
			},
			{
				Config:      changedConfig,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Cannot safely update imported template"),
			},
			{
				Config:      computedTitleConfig,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Cannot safely update imported template"),
			},
			{
				Config:             replacementConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:  importedConfig,
				Destroy: true,
			},
		},
	})
}

func TestResourceObservabilityTemplateUpdateRetainsImportedMetadataGuard(t *testing.T) {
	managed := NewResourceObservabilityTemplate().(*observabilityTemplateResource)
	var schemaResponse resource.SchemaResponse
	managed.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)

	prior := observabilityTemplateModel{
		ID:          types.StringValue("imported"),
		Title:       types.StringValue("Imported"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}
	planned := prior
	planned.Title = types.StringValue("Changed")

	state := tfsdk.State{Schema: schemaResponse.Schema}
	require.False(t, state.Set(t.Context(), prior).HasError())
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	require.False(t, plan.Set(t.Context(), planned).HasError())

	response := resource.UpdateResponse{State: state}
	managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)
	require.True(t, response.Diagnostics.HasError())
	assert.Equal(t, observabilityTemplateUnsafeUpdateSummary, response.Diagnostics.Errors()[0].Summary())
}

func TestObservabilityTemplateResourceModelMapping(t *testing.T) {
	ctx := context.Background()
	imports, diags := types.ListValueFrom(ctx, types.StringType, []string{"chart"})
	require.False(t, diags.HasError())
	datasource := &observabilityTemplateDatasourceModel{
		Type:        types.StringValue(string(template.DatasourceTypeSplunkObservability)),
		ProgramText: types.StringValue("data('requests').publish()"),
		SLOID:       types.StringNull(),
	}
	model := observabilityTemplateModel{
		Title:       types.StringValue("Example"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
		Metadata: &observabilityTemplateMetadataModel{
			Imports:    imports,
			Datasource: datasource,
		},
	}

	write, diags := observabilityTemplateWrite(ctx, model)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, observabilityTemplateRecordType, write.Type)
	assert.Equal(t, "Example", write.Title)
	assert.JSONEq(t, `{"<Chart>":[]}`, string(write.Spec))
	assert.Equal(t, []string{"/v2/template/chart"}, write.Metadata.Imports)
	require.NotNil(t, write.Metadata.Datasource)
	assert.Equal(t, template.DatasourceTypeSplunkObservability, write.Metadata.Datasource.Type)

	root := template.RootElementChart
	record := &template.Template{
		ID:       "template-id",
		Title:    "Example from API",
		Spec:     json.RawMessage(`{"<Chart>":[{"future":true}]}`),
		Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/chart"}},
	}
	state, err := observabilityTemplateModelFromRecord(model, record)
	require.NoError(t, err)
	assert.Equal(t, "template-id", state.ID.ValueString())
	assert.Equal(t, "Example from API", state.Title.ValueString())
	assert.Equal(t, string(template.RootElementChart), state.RootElement.ValueString())
	assert.JSONEq(t, `{"<Chart>":[{"future":true}]}`, state.Spec.ValueString())
	require.NotNil(t, state.Metadata)
	assert.Equal(t, imports, state.Metadata.Imports)
	assert.Same(t, datasource, state.Metadata.Datasource)
}

func TestObservabilityTemplateModelFromRecordLeavesOptionalMetadataUnsetOnImport(t *testing.T) {
	root := template.RootElement("FutureRootElement")
	record := &template.Template{
		ID:       "template-id",
		Title:    "Imported template",
		Spec:     json.RawMessage(`{"<FutureRootElement>":[]}`),
		Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/child"}},
	}

	state, err := observabilityTemplateModelFromRecord(observabilityTemplateModel{}, record)
	require.NoError(t, err)
	assert.Equal(t, "FutureRootElement", state.RootElement.ValueString())
	assert.Nil(t, state.Metadata)
}

func TestObservabilityTemplateModelFromRecordPreservesUnsetDirectImports(t *testing.T) {
	root := template.RootElementChart
	prior := observabilityTemplateModel{
		Metadata: &observabilityTemplateMetadataModel{
			Imports: types.ListNull(types.StringType),
		},
	}
	record := &template.Template{
		ID:       "template-id",
		Title:    "Template with descendants",
		Spec:     json.RawMessage(`{"<Chart>":[]}`),
		Metadata: &template.Metadata{RootElement: &root, Imports: []string{"/v2/template/direct", "/v2/template/descendant"}},
	}

	state, err := observabilityTemplateModelFromRecord(prior, record)
	require.NoError(t, err)
	require.NotNil(t, state.Metadata)
	assert.True(t, state.Metadata.Imports.IsNull())
}

func TestObservabilityTemplateRejectsIncompleteAPIRecords(t *testing.T) {
	root := template.RootElementChart
	for name, test := range map[string]struct {
		record *template.Template
		want   string
	}{
		"missing metadata":      {record: &template.Template{ID: "example"}, want: "without root element metadata"},
		"invalid specification": {record: &template.Template{ID: "example", Metadata: &template.Metadata{RootElement: &root}, Spec: json.RawMessage(`[]`)}, want: "invalid specification"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := observabilityTemplateModelFromRecord(observabilityTemplateModel{}, test.record)
			require.ErrorContains(t, err, test.want)
		})
	}

	_, err := observabilityTemplateFromResult(nil)
	require.ErrorContains(t, err, "no template record")
	_, err = observabilityTemplateFromResult(&template.Result{})
	require.ErrorContains(t, err, "no template record")
	_, err = observabilityTemplateFromResult(&template.Result{Data: &template.Template{}})
	require.ErrorContains(t, err, "without an ID")
}

func TestResourceObservabilityTemplateDeleteWarnsOnDirectoryMemberships(t *testing.T) {
	root := template.RootElementChart
	record := template.Template{
		ID:               "template-1",
		Type:             observabilityTemplateRecordType,
		Title:            "Request rate",
		Spec:             json.RawMessage(`{"<Chart>":[]}`),
		Metadata:         &template.Metadata{RootElement: &root},
		DirectoryEntries: []string{"/v2/directory/~users/example%40example.com/charts"},
	}
	deleteCalled := false
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityTemplateModel{
		ID:          types.StringValue("template-1"),
		Title:       types.StringValue("Request rate"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}).HasError())

	response := resource.DeleteResponse{State: state}
	managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)

	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	require.NotEmpty(t, response.Diagnostics.Warnings())
	assert.Equal(t, "Deleting Template with Directory memberships", response.Diagnostics.Warnings()[0].Summary())
	assert.Contains(t, response.Diagnostics.Warnings()[0].Detail(), "1 Directory entry")
	assert.NotContains(t, response.Diagnostics.Warnings()[0].Detail(), "/v2/directory/")

	assert.True(t, deleteCalled)
}

func TestResourceObservabilityTemplateDeleteIgnoresMissingRecord(t *testing.T) {
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "template not found", http.StatusNotFound)
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "template not found", http.StatusNotFound)
		}),
	}
	managed, resourceSchema := configuredObservabilityTemplateResourceWithHandlers(t, handlers)
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, state.Set(t.Context(), observabilityTemplateModel{
		ID:          types.StringValue("missing"),
		Title:       types.StringValue("Request rate"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}).HasError())

	response := resource.DeleteResponse{State: state}
	managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)

	assert.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	assert.Empty(t, response.Diagnostics.Warnings())
}

func configuredObservabilityTemplateResourceWithHandlers(t *testing.T, handlers map[string]http.Handler) (*observabilityTemplateResource, schema.Schema) {
	t.Helper()
	mockProvider := fwtest.NewMock(t, handlers)
	var providerResponse provider.ConfigureResponse
	mockProvider.Configure(t.Context(), provider.ConfigureRequest{}, &providerResponse)

	managed := NewResourceObservabilityTemplate().(*observabilityTemplateResource)
	var configureResponse resource.ConfigureResponse
	managed.Configure(t.Context(), resource.ConfigureRequest{ProviderData: providerResponse.ResourceData}, &configureResponse)
	require.False(t, configureResponse.Diagnostics.HasError(), configureResponse.Diagnostics)

	var schemaResponse resource.SchemaResponse
	managed.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	return managed, schemaResponse.Schema
}

func TestObservabilityTemplateWriteWithoutOptionalMetadata(t *testing.T) {
	model := observabilityTemplateModel{
		Title:       types.StringValue("Example"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}

	write, diags := observabilityTemplateWrite(context.Background(), model)
	require.False(t, diags.HasError(), diags)
	require.NotNil(t, write.Metadata.RootElement)
	assert.Equal(t, template.RootElementChart, *write.Metadata.RootElement)
	assert.Empty(t, write.Metadata.Imports)
	assert.Nil(t, write.Metadata.Datasource)
}

func TestObservabilityTemplateWriteRejectsInvalidModels(t *testing.T) {
	invalidImports, diags := types.ListValueFrom(t.Context(), types.StringType, []string{"invalid/id"})
	require.False(t, diags.HasError(), diags)
	base := observabilityTemplateModel{
		Title:       types.StringValue("Example"),
		RootElement: types.StringValue(string(template.RootElementChart)),
		Spec:        jsontypes.NewNormalizedValue(`{"<Chart>":[]}`),
	}

	tests := map[string]struct {
		model observabilityTemplateModel
		want  string
	}{
		"missing root element": {
			model: func() observabilityTemplateModel {
				model := base
				model.RootElement = types.StringNull()
				return model
			}(),
			want: "Missing template root element",
		},
		"invalid specification": {
			model: func() observabilityTemplateModel {
				model := base
				model.Spec = jsontypes.NewNormalizedValue(`[]`)
				return model
			}(),
			want: "Invalid template specification",
		},
		"invalid import": {
			model: func() observabilityTemplateModel {
				model := base
				model.Metadata = &observabilityTemplateMetadataModel{Imports: invalidImports}
				return model
			}(),
			want: "Invalid Template ID",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			write, diags := observabilityTemplateWrite(t.Context(), test.model)
			assert.Nil(t, write)
			require.True(t, diags.HasError())
			assert.Equal(t, test.want, diags.Errors()[0].Summary())
		})
	}
}
