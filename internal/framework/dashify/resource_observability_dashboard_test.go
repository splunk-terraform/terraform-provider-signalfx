// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceObservabilityDashboardMetadataAndSchema(t *testing.T) {
	t.Parallel()

	r := NewResourceObservabilityDashboard()
	var metadata resource.MetadataResponse
	r.Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &metadata)
	assert.Equal(t, "signalfx_observability_dashboard", metadata.TypeName)
	var schemaResponse resource.SchemaResponse
	r.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	resourceSchema := schemaResponse.Schema
	require.False(t, resourceSchema.ValidateImplementation(t.Context()).HasError())
	assert.NotEmpty(t, resourceSchema.Description)
	assert.Empty(t, resourceSchema.Blocks)

	root := path.Root("container").AtListIndex(0)
	section := root.AtName("section")
	sectionContainer := section.AtName("container").AtListIndex(0)
	group := sectionContainer.AtName("group")
	groupContainer := group.AtName("container").AtListIndex(0)
	for _, test := range []struct {
		path   path.Path
		exists bool
	}{
		{path.Root("control_bar").AtName("pinned_filter").AtListIndex(0).AtName("variable_name"), true},
		{path.Root("control_bar").AtName("filter_set").AtName("filter").AtListIndex(0).AtName("match_missing"), true},
		{path.Root("layout").AtName("defaults").AtName("max_height"), true},
		{path.Root("layout").AtName("defaults").AtName("x"), false},
		{root.AtName("template_content"), true},
		{root.AtName("layout").AtName("order"), true},
		{root.AtName("group").AtName("headerless"), true},
		{section.AtName("collapsible"), true},
		{section.AtName("layout").AtName("gap"), true},
		{sectionContainer.AtName("template_id"), true},
		{sectionContainer.AtName("section"), false},
		{groupContainer.AtName("template_id"), true},
		{groupContainer.AtName("section"), false},
		{groupContainer.AtName("group"), false},
	} {
		_, diags := resourceSchema.AttributeAtPath(t.Context(), test.path)
		assert.Equal(t, test.exists, !diags.HasError(), test.path.String())
	}

	content, diags := resourceSchema.AttributeAtPath(t.Context(), root.AtName("template_content"))
	require.False(t, diags.HasError())
	assert.Equal(t, jsontypes.NormalizedType{}, content.(schema.StringAttribute).CustomType)
	for _, titlePath := range []path.Path{section.AtName("title"), group.AtName("title")} {
		title, diags := resourceSchema.AttributeAtPath(t.Context(), titlePath)
		require.False(t, diags.HasError())
		assert.True(t, title.(schema.StringAttribute).Computed, titlePath.String())
		assert.NotNil(t, title.(schema.StringAttribute).Default, titlePath.String())
	}
}

func TestResourceObservabilityDashboardRejectsInvalidPlans(t *testing.T) {
	for _, test := range []struct {
		name   string
		config string
		want   string
	}{
		{
			name: "missing container content",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{}]
}`,
			want: "Invalid container content",
		},
		{
			name: "two content sources",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_id      = "chart-1"
    template_content = jsonencode({ "<Chart>" = [] })
  }]
}`,
			want: "Invalid template content",
		},
		{
			name: "invalid inline content",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_content = jsonencode({ chart = {} })
  }]
}`,
			want: "no dashboard element key",
		},
		{
			name: "empty container list",
			config: `resource "signalfx_observability_dashboard" "test" {
  title     = "Service overview"
  container = []
}`,
			want: "must contain at least 1 elements",
		},
		{
			name: "empty section",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    section = {}
  }]
}`,
			want: `attribute "container" is required`,
		},
		{
			name: "invalid width array",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_content = jsonencode({ "<Chart>" = [] })
    layout = { width = jsonencode([1, 2]) }
  }]
}`,
			want: "JSON arrays are supported only for x and y",
		},
		{
			name: "duplicate pinned control",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  control_bar = {
    pinned_filter = [
      { variable_name = "service" },
      { variable_name = "service" },
    ]
  }
}`,
			want: "Duplicate control variable name",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true,
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
					t, nil, fwtest.WithMockResources(NewResourceObservabilityDashboard),
				),
				Steps: []testresource.TestStep{{
					Config:      test.config,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(test.want),
				}},
			})
		})
	}
}

func TestResourceObservabilityDashboardPlansWithUnknownValues(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, nil, fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{
			{
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: `resource "terraform_data" "ids" {
  input = ["chart-1"]
}
resource "signalfx_observability_dashboard" "test" {
  title     = "Unknown containers"
  container = [for id in terraform_data.ids.output : { template_id = id }]
}`,
			},
			{
				PlanOnly: true,
				Config: `resource "terraform_data" "title" {
  input = "Computed title"
}
resource "signalfx_observability_dashboard" "test" {
  title     = terraform_data.title.output
  container = [{}]
}`,
				ExpectError: regexp.MustCompile("Invalid container content"),
			},
		},
	})
}

func TestResourceObservabilityDashboardRejectsImportedUpdate(t *testing.T) {
	root := template.RootElementDashboard
	imported := template.Template{
		ID: "imported-dashboard", Type: observabilityTemplateRecordType,
		Title: "Imported", Spec: json.RawMessage(`{"<Dashboard>":[],"layout":{"saved":{"_":{"items":[]}}}}`),
		Metadata: &template.Metadata{RootElement: &root},
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
	const importedConfig = `resource "signalfx_observability_dashboard" "test" {
  title = "Imported"
}`
	const changedConfig = `resource "signalfx_observability_dashboard" "test" {
  title = "Changed"
}`
	const computedTitleConfig = `resource "terraform_data" "title" {
  input = "Changed"
}
resource "signalfx_observability_dashboard" "test" {
  title = terraform_data.title.output
}`
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{
			{Config: `import {
  to = signalfx_observability_dashboard.test
  id = "imported-dashboard"
}
` + importedConfig},
			{Config: importedConfig, PlanOnly: true},
			{Config: changedConfig, PlanOnly: true, ExpectError: regexp.MustCompile(observabilityDashboardUnsafeUpdateSummary)},
			{Config: computedTitleConfig, PlanOnly: true, ExpectError: regexp.MustCompile(observabilityDashboardUnsafeUpdateSummary)},
			{Config: importedConfig, Destroy: true},
		},
	})
}

func TestResourceObservabilityDashboardUpdateRejectsImportedResource(t *testing.T) {
	managed := NewResourceObservabilityDashboard().(*observabilityDashboardResource)
	var schemaResponse resource.SchemaResponse
	managed.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	prior := observabilityDashboardModel{
		ID: types.StringValue("imported-dashboard"), Title: types.StringValue("Imported"),
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
	assert.Equal(t, observabilityDashboardUnsafeUpdateSummary, response.Diagnostics.Errors()[0].Summary())
}

func TestResourceObservabilityDashboardGeneratedConfig(t *testing.T) {
	chartRoot := template.RootElementChart
	dashboardRoot := template.RootElementDashboard
	var chart, dashboard *template.Template
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode create request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, observabilityTemplateRecordType, write.Type)
			var record *template.Template
			switch {
			case write.Metadata.RootElement != nil && *write.Metadata.RootElement == chartRoot:
				assert.Equal(t, "Request rate", write.Title)
				record = &template.Template{ID: "chart-1", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &chartRoot}}
				chart = record
			case write.Metadata.RootElement != nil && *write.Metadata.RootElement == dashboardRoot:
				assert.Equal(t, "Complete layout", write.Title)
				assert.Contains(t, string(write.Spec), `"controlBar"`)
				assert.Equal(t, []string{"/v2/template/chart-1"}, write.Metadata.Imports)
				record = &template.Template{ID: "dashboard-1", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &dashboardRoot, Imports: write.Metadata.Imports}}
				dashboard = record
			default:
				t.Errorf("unexpected root element %v", write.Metadata.RootElement)
				http.Error(w, "unexpected root element", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var record *template.Template
			switch r.PathValue("id") {
			case "chart-1":
				record = chart
			case "dashboard-1":
				record = dashboard
			}
			if record == nil {
				http.Error(w, "template not found", http.StatusNotFound)
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, []string{"chart-1", "dashboard-1"}, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			handlers,
			fwtest.WithMockResources(NewResourceObservabilityDashboard, NewResourceObservabilityTemplate),
		),
		Steps: []testresource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/observability_dashboard_layout.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_observability_dashboard.dashboard_layout", "title", "Complete layout"),
					testresource.TestCheckResourceAttr("signalfx_observability_dashboard.dashboard_layout", "control_bar.time_range.default_variable_value", "-PT15M"),
					testresource.TestCheckResourceAttr("signalfx_observability_dashboard.dashboard_layout", "control_bar.pinned_filter.#", "2"),
					testresource.TestCheckResourceAttr("signalfx_observability_dashboard.dashboard_layout", "container.0.section.container.0.group.container.0.layout.x", `["1/4",8]`),
				),
			},
			{
				ResourceName:    "signalfx_observability_dashboard.dashboard_layout",
				ImportState:     true,
				ImportStateKind: testresource.ImportBlockWithID,
				GenerateConfig:  true,
			},
		},
	})
}

func TestResourceObservabilityDashboardInlineContentLifecycleAndGeneratedConfig(t *testing.T) {
	root := template.RootElementDashboard
	var record *template.Template
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode dashboard create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, "Inline dashboard content", write.Title)
			assert.Equal(t, &root, write.Metadata.RootElement)
			assert.Empty(t, write.Metadata.Imports)
			assert.Contains(t, string(write.Spec), "o11y:SingleValue")
			record = &template.Template{ID: "dashboard-inline", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &root}}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "dashboard-inline", r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "dashboard-inline", r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			handlers,
			fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/observability_dashboard_inline_content.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.inline_content", "id"),
					testresource.TestCheckResourceAttr("signalfx_observability_dashboard.inline_content", "title", "Inline dashboard content"),
					testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.inline_content", "container.0.template_content"),
					testresource.TestCheckNoResourceAttr("signalfx_observability_dashboard.inline_content", "container.0.template_id"),
				),
			},
			{
				ResourceName:    "signalfx_observability_dashboard.inline_content",
				ImportState:     true,
				ImportStateKind: testresource.ImportBlockWithID,
				GenerateConfig:  true,
			},
		},
	})
}

func TestResourceObservabilityDashboardUntitledContainers(t *testing.T) {
	root := template.RootElementDashboard
	var record *template.Template
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode dashboard create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, "Untitled containers", write.Title)
			assert.Equal(t, &root, write.Metadata.RootElement)
			assert.Contains(t, string(write.Spec), "Section")
			assert.Contains(t, string(write.Spec), "Group")
			record = &template.Template{ID: "dashboard-untitled", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &root}}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "dashboard-untitled", r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "dashboard-untitled", r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			handlers,
			fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{{
			Config: `
resource "signalfx_observability_dashboard" "untitled" {
  title = "Untitled containers"

  container = [{
    section = {
      container = [{
        group = {
          container = [{
            template_content = jsonencode({ "<Chart>" = [] })
          }]
        }
      }]
    }
  }]
}
`,
			Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_observability_dashboard.untitled", "container.0.section.title", ""),
				testresource.TestCheckResourceAttr("signalfx_observability_dashboard.untitled", "container.0.section.container.0.group.title", ""),
			),
		}},
	})
}

func TestDashifyLayoutOptionValidators(t *testing.T) {
	t.Parallel()

	block := dashifyLayoutOptionsAttribute()
	for name, test := range map[string]struct {
		attribute   string
		value       float64
		expectError bool
	}{
		"zero gap":       {attribute: "gap", value: 0},
		"negative gap":   {attribute: "gap", value: -1, expectError: true},
		"one pixel step": {attribute: "step", value: 1},
		"zero step":      {attribute: "step", value: 0, expectError: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			attribute, ok := block.Attributes[test.attribute].(schema.Float64Attribute)
			require.True(t, ok)
			require.Len(t, attribute.Validators, 1)
			request := validator.Float64Request{
				Path:           path.Root(test.attribute),
				PathExpression: path.MatchRoot(test.attribute),
				ConfigValue:    types.Float64Value(test.value),
			}
			response := validator.Float64Response{}
			attribute.Validators[0].ValidateFloat64(context.Background(), request, &response)
			assert.Equal(t, test.expectError, response.Diagnostics.HasError())
		})
	}
}

func TestDashifyContainerContentErrorsDescribeOneContainer(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		level dashifyContainerLevel
		want  string
	}{
		"dashboard": {
			level: dashifyDashboardContainerLevel,
			want:  "each dashboard container must set exactly one content source: template_id, template_content, section, or group",
		},
		"section": {
			level: dashifySectionContainerLevel,
			want:  "each container within a section must set exactly one content source: template_id, template_content, or group",
		},
		"group": {
			level: dashifyGroupContainerLevel,
			want:  "each container within a group must set exactly one of template_id or template_content",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, dashifyContainerContentError(test.level))
		})
	}
}

func TestValidateDashifyTemplate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		model       dashifyTemplateModel
		wantError   bool
		wantMessage string
	}{
		"template id": {
			model: dashifyTemplateModel{
				TemplateID: types.StringValue("chart-id"),
				Content:    jsontypes.NewNormalizedNull(),
			},
		},
		"direct content": {
			model: dashifyTemplateModel{
				TemplateID: types.StringNull(),
				Content:    jsontypes.NewNormalizedValue(`{"<o11y:SingleValue>":[],"chart":{}}`),
			},
		},
		"chart wrapped content": {
			model: dashifyTemplateModel{
				TemplateID: types.StringNull(),
				Content:    jsontypes.NewNormalizedValue(`{"<Chart>":[{"<o11y:SingleValue>":[]}]}`),
			},
		},
		"neither": {
			model:       dashifyTemplateModel{TemplateID: types.StringNull(), Content: jsontypes.NewNormalizedNull()},
			wantError:   true,
			wantMessage: "exactly one",
		},
		"both": {
			model: dashifyTemplateModel{
				TemplateID: types.StringValue("chart-id"),
				Content:    jsontypes.NewNormalizedValue(`{"<o11y:SingleValue>":[]}`),
			},
			wantError:   true,
			wantMessage: "exactly one",
		},
		"empty template id": {
			model:       dashifyTemplateModel{TemplateID: types.StringValue(""), Content: jsontypes.NewNormalizedNull()},
			wantError:   true,
			wantMessage: "non-empty",
		},
		"malformed content": {
			model:       dashifyTemplateModel{TemplateID: types.StringNull(), Content: jsontypes.NewNormalizedValue(`{"<Chart>":`)},
			wantError:   true,
			wantMessage: "valid JSON",
		},
		"non-object content": {
			model:       dashifyTemplateModel{TemplateID: types.StringNull(), Content: jsontypes.NewNormalizedValue(`[]`)},
			wantError:   true,
			wantMessage: "JSON object",
		},
		"content without element": {
			model:       dashifyTemplateModel{TemplateID: types.StringNull(), Content: jsontypes.NewNormalizedValue(`{"chart":{}}`)},
			wantError:   true,
			wantMessage: "no dashboard element key",
		},
		"content with multiple elements": {
			model:       dashifyTemplateModel{TemplateID: types.StringNull(), Content: jsontypes.NewNormalizedValue(`{"<Chart>":[],"<Dashboard>":[]}`)},
			wantError:   true,
			wantMessage: "multiple dashboard element keys",
		},
		"content import element": {
			model:       dashifyTemplateModel{TemplateID: types.StringNull(), Content: jsontypes.NewNormalizedValue(`{"<$import.widget0>":[]}`)},
			wantError:   true,
			wantMessage: "use template_id",
		},
		"unknown template id": {
			model: dashifyTemplateModel{TemplateID: types.StringUnknown(), Content: jsontypes.NewNormalizedNull()},
		},
		"known id with unknown content": {
			model: dashifyTemplateModel{TemplateID: types.StringValue("chart-id"), Content: jsontypes.NewNormalizedUnknown()},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var response resource.ValidateConfigResponse
			validateDashifyTemplate(&response, path.Root("container").AtListIndex(0), &test.model)
			assert.Equal(t, test.wantError, response.Diagnostics.HasError(), response.Diagnostics)
			if test.wantMessage != "" {
				require.NotEmpty(t, response.Diagnostics.Errors())
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), test.wantMessage)
			}
		})
	}
}

func TestValidateObservabilityTitle(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		value       types.String
		expectError bool
	}{
		"value":      {value: types.StringValue(" Service health ")},
		"empty":      {value: types.StringValue(""), expectError: true},
		"spaces":     {value: types.StringValue("   "), expectError: true},
		"whitespace": {value: types.StringValue("\t\n"), expectError: true},
		"null":       {value: types.StringNull(), expectError: true},
		"unknown":    {value: types.StringUnknown()},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var response resource.ValidateConfigResponse
			validateObservabilityTitle(&response, path.Root("title"), test.value)
			assert.Equal(t, test.expectError, response.Diagnostics.HasError())
		})
	}
}

func TestValidateDashifyContainersAllowsUntitledSectionsAndGroups(t *testing.T) {
	t.Parallel()

	containers := []dashifyContainer{{
		Section: &dashifySection{
			Title: types.StringNull(),
			Container: []dashifyContainer{{
				Group: &dashifyGroup{
					Title: types.StringValue(""),
					Container: []dashifyContainer{{
						Template: &dashifyTemplateModel{
							TemplateID: types.StringValue("chart-id"),
							Content:    jsontypes.NewNormalizedNull(),
						},
					}},
				},
			}},
		},
	}}

	var response resource.ValidateConfigResponse
	validateDashifyContainers(&response, path.Root("container"), containers, dashifyDashboardContainerLevel)
	assert.False(t, response.Diagnostics.HasError(), response.Diagnostics)
}

func TestValidateDashifyLayoutBlocks(t *testing.T) {
	t.Parallel()

	t.Run("empty item layout", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateDashifyLayout(&resp, path.Root("layout"), &dashifyLayoutModel{})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "at least one")
	})

	t.Run("malformed item value", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateDashifyLayout(&resp, path.Root("layout"), &dashifyLayoutModel{Width: types.StringValue(`{"min":1}`)})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "must contain value")
	})

	t.Run("coordinate array", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateDashifyLayout(&resp, path.Root("layout"), &dashifyLayoutModel{X: types.StringValue(`["1/4",8]`)})
		assert.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	})

	t.Run("empty layout options", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateDashifyLayoutOptions(&resp, path.Root("layout"), &dashifyLayoutOptionsModel{})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "gap, step, defaults")
	})

	t.Run("empty defaults", func(t *testing.T) {
		var resp resource.ValidateConfigResponse
		validateDashifyLayoutOptions(&resp, path.Root("layout"), &dashifyLayoutOptionsModel{Defaults: &dashifyLayoutDefaultsModel{}})
		require.True(t, resp.Diagnostics.HasError())
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "defaults must set")
	})
}

func TestDashifyDashboardSpecRoundTrip(t *testing.T) {
	t.Parallel()

	model := observabilityDashboardModel{
		Title: types.StringValue("Service overview"),
		Layout: &dashifyLayoutOptionsModel{
			Gap:  types.Float64Value(0),
			Step: types.Float64Value(1),
			Defaults: &dashifyLayoutDefaultsModel{
				Absolute:  types.BoolValue(false),
				MinWidth:  types.StringValue("4"),
				MinHeight: types.StringValue("2"),
			},
		},
		Container: []dashifyDashboardContainerModel{
			{
				Layout: &dashifyLayoutModel{
					Absolute:  types.BoolValue(true),
					Width:     types.StringValue(`{"max":"100%","min":4,"value":"1/2"}`),
					Height:    types.StringValue("20"),
					MinWidth:  types.StringValue("4"),
					MaxWidth:  types.StringValue("100%"),
					MinHeight: types.StringValue("2"),
					MaxHeight: types.StringValue("30"),
					X:         types.StringValue(`["1/4",8]`),
					Y:         types.StringValue("12"),
				},
				TemplateID: types.StringValue("chart-a"),
			},
			{
				Section: &dashifySectionModel{
					Title:       types.StringValue("Latency"),
					Collapse:    types.BoolValue(false),
					Collapsible: types.BoolValue(true),
					Layout: &dashifyLayoutOptionsModel{
						Gap:  types.Float64Value(8),
						Step: types.Float64Value(8),
					},
					Container: []dashifySectionContainerModel{
						{
							Group: &dashifyGroupModel{
								Title:      types.StringValue("By service"),
								Headerless: types.BoolValue(true),
								Layout: &dashifyLayoutOptionsModel{
									Defaults: &dashifyLayoutDefaultsModel{Width: types.StringValue("1/2")},
								},
								Container: []dashifyGroupContainerModel{{TemplateID: types.StringValue("chart-b")}},
							},
						},
					},
				},
			},
			{
				Group: &dashifyGroupModel{
					Title:      types.StringValue("Root group"),
					Headerless: types.BoolValue(false),
					Container: []dashifyGroupContainerModel{
						{
							Layout:     &dashifyLayoutModel{Width: types.StringValue("1/2")},
							TemplateID: types.StringValue("chart-c"),
						},
					},
				},
			},
		},
	}

	spec, imports, err := buildDashboardSpec(model)
	require.NoError(t, err)
	assert.Equal(t, []string{"/v2/template/chart-a", "/v2/template/chart-b", "/v2/template/chart-c"}, imports)
	var document map[string]any
	require.NoError(t, json.Unmarshal(spec, &document))
	assert.Len(t, document[dashifyDashboardElement], 3)
	assert.Contains(t, document, "$import:widget0")
	assert.Contains(t, document, "$import:widget1_0_0")
	assert.Contains(t, document, "$import:widget2_0")
	layout := document["layout"].(map[string]any)
	assert.Equal(t, float64(0), layout["gap"])
	assert.Equal(t, float64(1), layout["step"])
	saved := layout["saved"].(map[string]any)
	rootItems := saved["_"].(map[string]any)["items"].([]any)
	assert.NotContains(t, rootItems[0].(map[string]any), "order")
	assert.Equal(t, true, rootItems[0].(map[string]any)["absolute"])
	assert.Equal(t, []any{"1/4", float64(8)}, rootItems[0].(map[string]any)["x"])
	sectionMetadata := saved["_.1"].(map[string]any)["section"].(map[string]any)
	assert.Equal(t, false, sectionMetadata["collapse"])
	assert.Equal(t, true, sectionMetadata["collapsible"])
	groupMetadata := saved["_.1.0"].(map[string]any)["group"].(map[string]any)
	assert.Equal(t, true, groupMetadata["headerless"])
	dashboardChildren := document[dashifyDashboardElement].([]any)
	sectionElement := dashboardChildren[1].(map[string]any)
	assert.Equal(t, float64(8), sectionElement["layout"].(map[string]any)["gap"])

	root := template.RootElementDashboard
	parsed, diags := parseDashboardTemplate(&template.Template{ID: "dashboard-id", Title: "Service overview", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	assert.Equal(t, model.Title, parsed.Title)
	assert.Equal(t, model.Layout, parsed.Layout)
	assert.Equal(t, model.Container, parsed.Container)
}

func TestDashifyDashboardInlineContentRoundTrip(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"direct element": `{"<o11y:SingleValue>":[],"chart":{"color":"blue"},"widget":{"title":"Requests"}}`,
		"chart wrapper":  `{"<Chart>":[{"<o11y:SingleValue>":[],"chart":{},"future":{"preserved":true}}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			model := observabilityDashboardModel{
				Title: types.StringValue("Dashboard"),
				Container: []dashifyDashboardContainerModel{{
					TemplateContent: jsontypes.NewNormalizedValue(content),
				}},
			}

			spec, imports, err := buildDashboardSpec(model)
			require.NoError(t, err)
			assert.Empty(t, imports)

			var document map[string]any
			require.NoError(t, json.Unmarshal(spec, &document))
			assert.NotContains(t, document, "$import:widget0")
			children := document[dashifyDashboardElement].([]any)
			panel := children[0].(map[string]any)[dashifyPanelElement].([]any)
			encodedPanelContent, err := json.Marshal(panel[0])
			require.NoError(t, err)
			assert.JSONEq(t, content, string(encodedPanelContent))

			root := template.RootElementDashboard
			parsed, diags := parseDashboardTemplate(&template.Template{
				Type:     observabilityTemplateRecordType,
				Title:    "Dashboard",
				Spec:     spec,
				Metadata: &template.Metadata{RootElement: &root},
			})
			require.Empty(t, diags, diags)
			require.Len(t, parsed.Container, 1)
			assert.True(t, parsed.Container[0].TemplateID.IsNull())
			assert.JSONEq(t, content, parsed.Container[0].TemplateContent.ValueString())

			rebuilt, rebuiltImports, err := buildDashboardSpec(parsed)
			require.NoError(t, err)
			assert.Empty(t, rebuiltImports)
			assert.JSONEq(t, string(spec), string(rebuilt))
		})
	}
}

func TestDashifyDashboardSpecWarnsAboutUnmodeledFields(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{
		"title":"Dashboard",
		"<Dashboard>":[
			{"<Section>":[
				{"<Group>":[
					{"<Panel>":[{"<$import.widget0>":[],"futureContent":{"preserve":true}}],"futureContainer":true}
				]}
			]}
		],
		"layout":{"saved":{
			"_":{"items":[{"id":"_.0","order":0,"rootMinW":1}]},
			"_.0":{"items":[{"id":"_.0.0","order":0}],"section":{"title":"Section","futureSection":true}},
			"_.0.0":{"items":[{"id":"_.0.0.0","order":0,"minW":2,"futureConstraint":3}],"group":{"title":"Group","futureGroup":true}}
		}},
		"$import:widget0":"/v2/template/chart-id"
	}`)
	model, diags := parseDashboardTemplate(&template.Template{
		Type:     observabilityTemplateRecordType,
		Title:    "Dashboard",
		Spec:     spec,
		Metadata: &template.Metadata{RootElement: &root},
	})
	require.False(t, diags.HasError(), diags)
	require.Len(t, model.Container, 1)
	require.NotNil(t, model.Container[0].Section)
	nestedLayout := model.Container[0].Section.Container[0].Group.Container[0].Layout
	require.NotNil(t, nestedLayout)
	assert.Equal(t, "2", nestedLayout.MinWidth.ValueString())

	warnings := diags.Warnings()
	require.Len(t, warnings, 1)
	for _, path := range []string{
		"container.0.section.container.0.group.container.0.futureContainer",
		"container.0.section.container.0.group.container.0.futureContent.preserve",
		"layout._.0.rootMinW",
		"layout._.0.0.0.futureConstraint",
		"layout.saved._.0.section.futureSection",
		"layout.saved._.0.0.group.futureGroup",
	} {
		assert.Contains(t, warnings[0].Detail(), path)
	}
	assert.NotContains(t, warnings[0].Detail(), "layout._.0.0.0.minW")
}

func TestDashifyDashboardSpecAllowsMissingLayout(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[]}]}],"$import:widget0":"/v2/template/chart-id"}`)
	model, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	require.Len(t, model.Container, 1)
	assert.Nil(t, model.Container[0].Layout)
	assert.Equal(t, "chart-id", model.Container[0].TemplateID.ValueString())
}

func TestDashifyDashboardSpecAllowsUntitledSectionsAndGroups(t *testing.T) {
	t.Parallel()

	for name, layout := range map[string]string{
		"missing metadata": "",
		"missing titles":   `,"layout":{"saved":{"_":{"items":[{"id":"_.0"}]},"_.0":{"items":[{"id":"_.0.0"}],"section":{}},"_.0.0":{"items":[{"id":"_.0.0.0"}],"group":{}}}}`,
		"empty titles":     `,"layout":{"saved":{"_":{"items":[{"id":"_.0"}]},"_.0":{"items":[{"id":"_.0.0"}],"section":{"title":""}},"_.0.0":{"items":[{"id":"_.0.0.0"}],"group":{"title":""}}}}`,
		"null titles":      `,"layout":{"saved":{"_":{"items":[{"id":"_.0"}]},"_.0":{"items":[{"id":"_.0.0"}],"section":{"title":null}},"_.0.0":{"items":[{"id":"_.0.0.0"}],"group":{"title":null}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := template.RootElementDashboard
			spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Section>":[{"<Group>":[{"<Panel>":[{"<$import.widget0>":[]}]}]}]}]` + layout + `,"$import:widget0":"/v2/template/chart-id"}`)
			model, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
			require.Empty(t, diags, diags)
			require.Len(t, model.Container, 1)
			require.NotNil(t, model.Container[0].Section)
			assert.False(t, model.Container[0].Section.Title.IsNull())
			assert.Empty(t, model.Container[0].Section.Title.ValueString())
			require.Len(t, model.Container[0].Section.Container, 1)
			require.NotNil(t, model.Container[0].Section.Container[0].Group)
			assert.False(t, model.Container[0].Section.Container[0].Group.Title.IsNull())
			assert.Empty(t, model.Container[0].Section.Container[0].Group.Title.ValueString())

			rebuilt, _, err := buildDashboardSpec(model)
			require.NoError(t, err)
			reparsed, reparsedDiags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: rebuilt, Metadata: &template.Metadata{RootElement: &root}})
			require.Empty(t, reparsedDiags, reparsedDiags)
			assert.Equal(t, model.Container, reparsed.Container)
		})
	}
}

func TestDashifyDashboardSpecRejectsAmbiguousLayouts(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		layout string
		want   string
	}{
		"layout is not an object": {
			layout: `[]`,
			want:   "layout is []interface {} rather than an object",
		},
		"saved is not an object": {
			layout: `{"saved":[]}`,
			want:   "layout.saved is []interface {} rather than an object",
		},
		"saved list is not an object": {
			layout: `{"saved":{"_":[]}}`,
			want:   `layout.saved["_"] is []interface {} rather than an object`,
		},
		"items is not a list": {
			layout: `{"saved":{"_":{"items":{}}}}`,
			want:   `layout.saved["_"].items is map[string]interface {} rather than a list`,
		},
		"two items place one container": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0"},{"id":"_.0"}]}}}`,
			want:   "both place container 0",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := template.RootElementDashboard
			spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[]}]}],"layout":` + test.layout + `,"$import:widget0":"/v2/template/chart-id"}`)
			_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), test.want)
		})
	}
}

func TestDashifyDashboardSpecRejectsInvalidLayoutValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		layout string
		want   string
	}{
		"absolute is not boolean": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","absolute":"yes"}]}}}`,
			want:   "rather than a boolean",
		},
		"width is coordinate array": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","w":[1,2]}]}}}`,
			want:   "only for x and y",
		},
		"clamped width lacks value": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","w":{"min":1}}]}}}`,
			want:   "must contain value",
		},
		"coordinate contains boolean": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","x":[1,true]}]}}}`,
			want:   "coordinate part 1",
		},
		"gap is not numeric": {
			layout: `{"gap":"8","saved":{"_":{"items":[{"id":"_.0"}]}}}`,
			want:   "rather than a finite number",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := template.RootElementDashboard
			spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[]}]}],"layout":` + test.layout + `,"$import:widget0":"/v2/template/chart-id"}`)
			_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), test.want)
		})
	}
}

func TestDashifyDashboardSpecRejectsInvalidGroupMetadata(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{
		"title":"Dashboard",
		"<Dashboard>":[{"<Group>":[{"<Panel>":[{"<$import.widget0>":[]}]}]}],
		"layout":{"saved":{
			"_":{"items":[{"id":"_.0"}]},
			"_.0":{"items":[{"id":"_.0.0"}],"group":{"title":"Group","headerless":"yes"}}
		}},
		"$import:widget0":"/v2/template/chart-id"
	}`)
	_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.True(t, diags.HasError(), diags)
	assert.Contains(t, diags.Errors()[0].Detail(), "headerless")
	assert.Contains(t, diags.Errors()[0].Detail(), "rather than a boolean")
}

func TestDashifyDashboardSpecPreservesUIOrder(t *testing.T) {
	t.Parallel()

	root := template.RootElementDashboard
	spec := json.RawMessage(`{
		"title":"Dashboard",
		"<Dashboard>":[
			{"<Panel>":[{"<$import.widget0>":[]}]},
			{"<Panel>":[{"<$import.widget1>":[]}]}
		],
		"layout":{"saved":{"_":{
			"at":"2026-09-11T00:00:00Z",
			"parent":"dashboard-id",
			"items":[
				{"id":"_.1","order":0.5,"w":4},
				{"id":"_.0","order":9,"w":8}
			]
		}}},
		"$import:widget0":"/v2/template/chart-a",
		"$import:widget1":"/v2/template/chart-b"
	}`)
	model, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	require.Len(t, model.Container, 2)
	assert.Equal(t, "8", model.Container[0].Layout.Width.ValueString())
	assert.Equal(t, "4", model.Container[1].Layout.Width.ValueString())
	assert.Equal(t, 9.0, model.Container[0].Layout.Order.ValueFloat64())
	assert.Equal(t, 0.5, model.Container[1].Layout.Order.ValueFloat64())

	rebuilt, _, err := buildDashboardSpec(model)
	require.NoError(t, err)
	assert.Contains(t, string(rebuilt), `"order":9`)
	assert.Contains(t, string(rebuilt), `"order":0.5`)
	assert.NotContains(t, string(rebuilt), `"parent"`)
	assert.NotContains(t, string(rebuilt), `"at"`)
}

func TestResourceObservabilityDashboardUnitTest(t *testing.T) {
	chartRoot := template.RootElementChart
	dashboardRoot := template.RootElementDashboard
	var chart, dashboard *template.Template
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode template create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, observabilityTemplateRecordType, write.Type)
			var record *template.Template
			switch {
			case write.Metadata.RootElement != nil && *write.Metadata.RootElement == chartRoot:
				assert.Equal(t, "Request rate", write.Title)
				chart = &template.Template{ID: "chart-1", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &chartRoot}}
				record = chart
			case write.Metadata.RootElement != nil && *write.Metadata.RootElement == dashboardRoot:
				assert.Equal(t, "Service overview", write.Title)
				assert.Equal(t, []string{"/v2/template/chart-1"}, write.Metadata.Imports)
				dashboard = &template.Template{ID: "dashboard-1", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &dashboardRoot, Imports: write.Metadata.Imports}}
				record = dashboard
			default:
				t.Errorf("unexpected root element %v", write.Metadata.RootElement)
				http.Error(w, "unexpected root element", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var record *template.Template
			switch r.PathValue("id") {
			case "chart-1":
				record = chart
			case "dashboard-1":
				record = dashboard
			}
			if record == nil {
				http.Error(w, "template not found", http.StatusNotFound)
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "dashboard-1", r.PathValue("id"))
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode dashboard update request: %v", err)
				http.Error(w, "invalid update request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, "Updated service overview", write.Title)
			assert.Equal(t, []string{"/v2/template/chart-1"}, write.Metadata.Imports)
			dashboard = &template.Template{ID: "dashboard-1", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &dashboardRoot, Imports: write.Metadata.Imports}}
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: dashboard}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, []string{"chart-1", "dashboard-1"}, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(
		t,
		testresource.TestCase{
			IsUnitTest: true,
			TerraformVersionChecks: []tfversion.TerraformVersionCheck{
				tfversion.RequireAbove(tfversion.Version0_12_26),
			},
			ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
				t,
				handlers,
				fwtest.WithMockResources(NewResourceObservabilityDashboard, NewResourceObservabilityTemplate),
			),
			Steps: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_observability_dashboard.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "title", "Service overview"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.width", "6/12"),
						testresource.TestCheckResourceAttrPair(
							"signalfx_observability_dashboard.test", "container.0.template_id",
							"signalfx_observability_template.chart", "id",
						),
					),
				},
				{
					ConfigFile: config.StaticFile("testdata/01_observability_dashboard_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "title", "Updated service overview"),
						testresource.TestCheckNoResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.width"),
						testresource.TestCheckResourceAttrPair(
							"signalfx_observability_dashboard.test", "container.0.template_id",
							"signalfx_observability_template.chart", "id",
						),
					),
				},
			},
		},
	)
}

func TestResourceObservabilityDashboardRecreatesAfterRemoteDelete(t *testing.T) {
	createdID := "template-1"
	var readFixture *template.Template
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode dashboard create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			root := template.RootElementDashboard
			assert.Equal(t, &root, write.Metadata.RootElement)
			readFixture = &template.Template{
				ID: createdID, Type: write.Type, Title: write.Title, Spec: write.Spec,
				Metadata: &template.Metadata{RootElement: &root, Imports: write.Metadata.Imports},
			}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: readFixture}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if readFixture == nil {
				http.Error(w, "dashboard not found", http.StatusNotFound)
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
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{
			{Config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_content = jsonencode({ "<Chart>" = [] })
  }]
}`},
			{
				PreConfig: func() {
					createdID = "template-2"
					readFixture = nil
				},
				Config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_content = jsonencode({ "<Chart>" = [] })
  }]
}`,
				Check: testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "id", "template-2"),
			},
		},
	})
}

func TestResourceObservabilityDashboardReadRejectsMismatchedID(t *testing.T) {
	root := template.RootElementDashboard
	var record *template.Template
	mismatched := false
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode dashboard create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, &root, write.Metadata.RootElement)
			record = &template.Template{ID: "template-1", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &root}}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "template-1", r.PathValue("id"))
			if mismatched {
				assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &template.Template{
					ID: "another-dashboard", Type: observabilityTemplateRecordType,
					Title: "Service overview", Metadata: &template.Metadata{RootElement: &root},
				}}))
				return
			}
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "template-1", r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	const dashboardConfig = `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_content = jsonencode({ "<Chart>" = [] })
  }]
}`
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{
			{Config: dashboardConfig},
			{
				PreConfig:   func() { mismatched = true },
				Config:      dashboardConfig,
				ExpectError: regexp.MustCompile(`(?s)Template API returned record "another-dashboard" for Dashboard.*"template-1"`),
			},
		},
	})
}

func TestObservabilityDashboardRejectsInvalidAPIRecords(t *testing.T) {
	root := template.RootElementDashboard
	chartRoot := template.RootElementChart
	for _, test := range []struct {
		name   string
		record template.Template
		want   string
	}{
		{
			name:   "missing root",
			record: template.Template{ID: "dashboard-1", Type: observabilityTemplateRecordType, Spec: json.RawMessage(`{"<Dashboard>":[]}`)},
			want:   "no Dashboard root element metadata",
		},
		{
			name:   "wrong root",
			record: template.Template{ID: "dashboard-1", Type: observabilityTemplateRecordType, Metadata: &template.Metadata{RootElement: &chartRoot}, Spec: json.RawMessage(`{"<Dashboard>":[]}`)},
			want:   "not \"Dashboard\"",
		},
		{
			name:   "missing spec",
			record: template.Template{ID: "dashboard-1", Type: observabilityTemplateRecordType, Metadata: &template.Metadata{RootElement: &root}},
			want:   "EOF",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, _, diags := observabilityDashboardFromResult(&template.Result{Data: &test.record}, test.record.ID)
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), test.want)
		})
	}
}

func TestResourceObservabilityDashboardCreateReportsMissingEndpoint(t *testing.T) {
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "template endpoint not found", http.StatusNotFound)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{{
			Config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_content = jsonencode({ "<Chart>" = [] })
  }]
}`,
			ExpectError: regexp.MustCompile("The Template API endpoint was not found"),
		}},
	})
}

func TestResourceObservabilityDashboardUpdateReportsNotFound(t *testing.T) {
	root := template.RootElementDashboard
	record := &template.Template{
		ID: "template-1", Type: observabilityTemplateRecordType, Title: "Service overview",
		Spec: json.RawMessage(`{"<Dashboard>":[]}`), Metadata: &template.Metadata{RootElement: &root},
	}
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode dashboard create request: %v", err)
				http.Error(w, "invalid create request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, &root, write.Metadata.RootElement)
			record.Spec = write.Spec
			record.Metadata.Imports = write.Metadata.Imports
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			http.Error(w, "dashboard not found", http.StatusNotFound)
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityDashboard),
		),
		Steps: []testresource.TestStep{
			{Config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_content = jsonencode({ "<Chart>" = [] })
  }]
}`},
			{
				Config: `resource "signalfx_observability_dashboard" "test" {
  title = "Updated overview"
  container = [{
    template_content = jsonencode({ "<Chart>" = [] })
  }]
}`,
				ExpectError: regexp.MustCompile("was not found during update"),
			},
		},
	})
}

func TestDashboardUsesRecordTitle(t *testing.T) {
	model := observabilityDashboardModel{Title: types.StringValue("Dashboard title")}
	spec, _, err := buildDashboardSpec(model)
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(spec, &document))
	document["title"] = "Stale spec title"
	spec, err = json.Marshal(document)
	require.NoError(t, err)
	rootElement := template.RootElementDashboard
	parsed, diags := parseDashboardTemplate(&template.Template{
		Title:    "Dashboard title",
		Spec:     spec,
		Metadata: &template.Metadata{RootElement: &rootElement},
	})
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "Dashboard title", parsed.Title.ValueString())
}

func TestDashboardCompactsNullContainersWithoutLosingLayout(t *testing.T) {
	rootElement := template.RootElementDashboard
	record := &template.Template{
		Title:    "Service overview",
		Metadata: &template.Metadata{RootElement: &rootElement},
		Spec: json.RawMessage(`{
            "<Dashboard>": [
                null,
                {"<Panel>": [{"<Chart>": []}]}
            ],
            "layout": {"saved": {"_": {"items": [
                {"id": "_.0", "w": 2},
                {"id": "_.1", "w": 6}
            ]}}}
        }`),
	}
	model, diags := parseDashboardTemplate(record)
	require.False(t, diags.HasError(), diags)
	require.Len(t, diags.Warnings(), 1)
	assert.Contains(t, diags.Warnings()[0].Detail(), "null element")
	require.Len(t, model.Container, 1)
	require.NotNil(t, model.Container[0].Layout)
	assert.Equal(t, "6", model.Container[0].Layout.Width.ValueString())

	rebuilt, _, err := buildDashboardSpec(model)
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(rebuilt, &document))
	assert.Len(t, document["<Dashboard>"], 1)
	assert.Contains(t, string(rebuilt), `"id":"_.0"`)
	assert.NotContains(t, string(rebuilt), `"id":"_.1"`)
}

func TestDashboardInlineContentPreservesLargeJSONNumbers(t *testing.T) {
	const content = `{"<o11y:SingleValue>":[],"chart":{"sampleCount":9007199254740993}}`
	model := observabilityDashboardModel{
		Title: types.StringValue("Large numbers"),
		Container: []dashifyDashboardContainerModel{{
			TemplateContent: jsontypes.NewNormalizedValue(content),
		}},
	}
	spec, _, err := buildDashboardSpec(model)
	require.NoError(t, err)
	assert.Contains(t, string(spec), "9007199254740993")
	equal, diags := jsontypes.NewNormalizedValue(content).StringSemanticEquals(t.Context(), jsontypes.NewNormalizedValue(`{"<o11y:SingleValue>":[],"chart":{"sampleCount":9007199254740992}}`))
	require.False(t, diags.HasError(), diags)
	assert.False(t, equal)

	rootElement := template.RootElementDashboard
	parsed, diags := parseDashboardTemplate(&template.Template{
		Title:    "Large numbers",
		Spec:     spec,
		Metadata: &template.Metadata{RootElement: &rootElement},
	})
	require.False(t, diags.HasError(), diags)
	require.Len(t, parsed.Container, 1)
	assert.Contains(t, parsed.Container[0].TemplateContent.ValueString(), "9007199254740993")
}

func TestDashboardInlineContentRejectsNestedImports(t *testing.T) {
	_, err := decodeDashifyInlineContent(`{"<Chart>":[{"<$import.hidden>":[]}]}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot contain import element")
}

func TestDashboardWriteWithoutReferencesHasNoImports(t *testing.T) {
	write, err := observabilityDashboardTemplateWrite(observabilityDashboardModel{
		Title: types.StringValue("No references"),
	})
	require.NoError(t, err)
	assert.Empty(t, write.Metadata.Imports)
}

func TestDashboardParsesWrappedTemplateImport(t *testing.T) {
	rootElement := template.RootElementDashboard
	record := &template.Template{
		Title:    "Wrapped import",
		Metadata: &template.Metadata{RootElement: &rootElement},
		Spec: json.RawMessage(`{
            "<Dashboard>": [{"<Panel>": [{"<Chart>": [{"<$import.chart>": []}]}]}],
            "$import:chart": "/v2/template/chart-id",
            "layout": {"saved": {"_": {"items": [{"id": "_.0"}]}}}
        }`),
	}
	model, diags := parseDashboardTemplate(record)
	require.False(t, diags.HasError(), diags)
	require.Empty(t, diags.Warnings())
	require.Len(t, model.Container, 1)
	assert.Equal(t, "chart-id", model.Container[0].TemplateID.ValueString())
}

func TestDashboardRejectsNestedImportWithUnmodeledWrapperFields(t *testing.T) {
	rootElement := template.RootElementDashboard
	record := &template.Template{
		Title:    "Nested import",
		Metadata: &template.Metadata{RootElement: &rootElement},
		Spec: json.RawMessage(`{
            "<Dashboard>": [{"<Panel>": [{
                "<Chart>": [{"<$import.chart>": []}],
                "chart": {"color": "blue"}
            }]}],
            "$import:chart": "/v2/template/chart-id",
            "layout": {"saved": {"_": {"items": [{"id": "_.0"}]}}}
        }`),
	}
	_, diags := parseDashboardTemplate(record)
	require.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "unsupported nested import")
}
