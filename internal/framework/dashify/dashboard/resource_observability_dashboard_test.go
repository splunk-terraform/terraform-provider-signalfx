// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
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
	for _, attributePath := range []path.Path{
		path.Root("layout"),
		root.AtName("section"),
		root.AtName("group"),
		root.AtName("layout").AtName("x"),
		root.AtName("layout").AtName("absolute"),
		root.AtName("layout").AtName("max_width"),
	} {
		_, diags := resourceSchema.AttributeAtPath(t.Context(), attributePath)
		assert.True(t, diags.HasError(), attributePath.String())
	}

	content, diags := resourceSchema.AttributeAtPath(t.Context(), root.AtName("inline_content"))
	require.False(t, diags.HasError())
	assert.Equal(t, jsontypes.NormalizedType{}, content.(schema.StringAttribute).CustomType)
}

func TestResourceObservabilityDashboardRejectsInvalidPlans(t *testing.T) {
	for _, test := range []struct {
		name   string
		config string
		want   string
	}{
		{
			name: "blank title",
			config: `resource "signalfx_observability_dashboard" "test" {
  title     = "   "
  container = [{ template_id = "chart-1" }]
}`,
			want: "title must contain at least one non-whitespace character",
		},
		{
			name: "missing container content",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{}]
}`,
			want: "Invalid panel content",
		},
		{
			name: "two content sources",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    template_id      = "chart-1"
    inline_content = jsonencode({ "<Chart>" = [] })
  }]
}`,
			want: "Invalid panel content",
		},
		{
			name: "invalid inline content",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    inline_content = jsonencode({ chart = {} })
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
			name: "invalid width array",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  container = [{
    inline_content = jsonencode({ "<Chart>" = [] })
    layout = { width = jsonencode([1, 2]) }
  }]
}`,
			want: "JSON objects and arrays are not supported",
		},
		{
			name: "empty item layout",
			config: `resource "signalfx_observability_dashboard" "test" {
  title     = "Service overview"
  container = [{ template_id = "chart-1", layout = {} }]
}`,
			want: "layout must set at least one size option",
		},
		{
			name: "duplicate pinned control",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  control_bar = {
    pinned_filter = [
      { property = "service.name", variable_name = "service" },
      { property = "service.component", variable_name = "service" },
    ]
  }
}`,
			want: "Duplicate control variable name",
		},
		{
			name: "duplicate defaulted pinned identities",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  control_bar = { pinned_filter = [{ property = "service.name" }, { property = "service.name" }] }
}`,
			want: "Duplicate control variable name",
		},
		{
			name: "explicit and defaulted identity collision",
			config: `resource "signalfx_observability_dashboard" "test" {
  title = "Service overview"
  control_bar = { pinned_filter = [{ property = "service.name" }, { property = "service.component", variable_name = "service.name" }] }
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
	writes := 0
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writes++
			http.Error(w, "invalid computed Template ID must be rejected before writing", http.StatusInternalServerError)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t, handlers, fwtest.WithMockResources(NewResourceObservabilityDashboard),
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
  control_bar = { pinned_filter = [{ property = terraform_data.ids.output[0] }] }
}`,
				ConfigPlanChecks: testresource.ConfigPlanChecks{PostApplyPreRefresh: []plancheck.PlanCheck{
					plancheck.ExpectUnknownValue("signalfx_observability_dashboard.test", tfjsonpath.New("control_bar").AtMapKey("pinned_filter").AtSliceIndex(0).AtMapKey("variable_name")),
				}},
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
				ExpectError: regexp.MustCompile("Invalid panel content"),
			},
			{
				Config: `resource "terraform_data" "id" {
  input = ""
}
resource "signalfx_observability_dashboard" "test" {
  title = "Computed panel content"
  container = [{ template_id = terraform_data.id.output }]
}`,
				ExpectError: regexp.MustCompile("template_id must be non-empty when set"),
			},
		},
	})
	assert.Zero(t, writes)
}

func TestResourceObservabilityDashboardUpdatesModeledImport(t *testing.T) {
	root := template.RootElementDashboard
	imported := template.Template{
		ID: "imported-dashboard", Type: fwdashify.ObservabilityTemplateRecordType,
		Title: "Imported", Spec: json.RawMessage(`{"<Dashboard>":[],"layout":{"saved":{"_":{"items":[]}}}}`),
		Metadata: &template.Metadata{RootElement: &root},
	}
	writes := 0
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, imported.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &imported}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writes++
			assert.Equal(t, imported.ID, r.PathValue("id"))
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode update request: %v", err)
				http.Error(w, "invalid update request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, "Changed", write.Title)
			assert.Equal(t, &root, write.Metadata.RootElement)
			assert.Empty(t, write.Metadata.Imports)
			assert.Nil(t, write.Metadata.Datasource)
			imported.Title = write.Title
			imported.Spec = write.Spec
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
			{Config: changedConfig, Check: testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "title", "Changed")},
			{Config: changedConfig, PlanOnly: true},
			{
				PreConfig: func() {
					assert.Equal(t, 1, writes)
					imported.Spec = json.RawMessage(`{"<Dashboard>":[],"futureField":true}`)
				},
				Config: `resource "signalfx_observability_dashboard" "test" {
  title = "Changed again"
}`,
				PlanOnly: true,
				// The non-refresh plan still sees the old write marker; only the refresh plan revokes it.
				ExpectNonEmptyPlan: true,
				ExpectError:        regexp.MustCompile(observabilityDashboardUnsafeUpdateSummary),
			},
			{Config: changedConfig, Destroy: true},
		},
	})
	assert.Equal(t, 1, writes, "refresh must revoke permission to overwrite newly unmodeled fields")
}

func TestResourceObservabilityDashboardRejectsUnmodeledImportUpdateDuringPlan(t *testing.T) {
	root := template.RootElementDashboard
	record := template.Template{
		ID: "imported-dashboard", Type: fwdashify.ObservabilityTemplateRecordType,
		Title: "Imported", Spec: json.RawMessage(`{"<Dashboard>":[],"futureField":true}`),
		Metadata: &template.Metadata{RootElement: &root},
	}
	const importedConfig = `resource "signalfx_observability_dashboard" "test" {
  title = "Imported"
}`
	for _, test := range []struct {
		name   string
		config string
	}{
		{name: "known title", config: `resource "signalfx_observability_dashboard" "test" {
  title = "Changed"
}`},
		{name: "computed title", config: `resource "terraform_data" "title" {
  input = "Changed"
}
resource "signalfx_observability_dashboard" "test" {
  title = terraform_data.title.output
}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			handlers := map[string]http.Handler{
				"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, record.ID, r.PathValue("id"))
					assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &record}))
				}),
				"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, record.ID, r.PathValue("id"))
					w.WriteHeader(http.StatusNoContent)
				}),
			}
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
					{Config: test.config, PlanOnly: true, ExpectError: regexp.MustCompile(observabilityDashboardUnsafeUpdateSummary)},
				},
			})
		})
	}
}

func TestResourceObservabilityDashboardUpdateRejectsUnknownWriteState(t *testing.T) {
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
					testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.inline_content", "container.0.inline_content"),
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

func TestValidateDashifyTemplate(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		model       dashifyDashboardContainerModel
		wantError   bool
		wantMessage string
	}{
		"empty template id":              {model: dashifyDashboardContainerModel{TemplateID: types.StringValue("")}, wantError: true, wantMessage: "non-empty"},
		"malformed content":              {model: dashifyDashboardContainerModel{InlineContent: jsontypes.NewNormalizedValue(`{"<Chart>":`)}, wantError: true, wantMessage: "valid JSON"},
		"multiple JSON values":           {model: dashifyDashboardContainerModel{InlineContent: jsontypes.NewNormalizedValue(`{"<Chart>":[]} {}`)}, wantError: true, wantMessage: "multiple JSON values"},
		"non-object content":             {model: dashifyDashboardContainerModel{InlineContent: jsontypes.NewNormalizedValue(`[]`)}, wantError: true, wantMessage: "JSON object"},
		"content with multiple elements": {model: dashifyDashboardContainerModel{InlineContent: jsontypes.NewNormalizedValue(`{"<Chart>":[],"<Dashboard>":[]}`)}, wantError: true, wantMessage: "multiple dashboard element keys"},
		"content import element":         {model: dashifyDashboardContainerModel{InlineContent: jsontypes.NewNormalizedValue(`{"<$import.widget0>":[]}`)}, wantError: true, wantMessage: "use template_id"},
		"unknown template id":            {model: dashifyDashboardContainerModel{TemplateID: types.StringUnknown()}},
		"known id with unknown content":  {model: dashifyDashboardContainerModel{TemplateID: types.StringValue("chart-id"), InlineContent: jsontypes.NewNormalizedUnknown()}},
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

func TestDashifyDashboardSpecRoundTrip(t *testing.T) {
	t.Parallel()
	model := observabilityDashboardModel{
		Title: types.StringValue("Service overview"),
		Container: []dashifyDashboardContainerModel{
			{Layout: &dashifyLayoutModel{Width: types.StringValue("1/2"), Height: types.StringValue("20"), MinWidth: types.StringValue("4"), MinHeight: types.StringValue("2")}, TemplateID: types.StringValue("chart-a")},
			{TemplateID: types.StringValue("chart-b")},
		},
	}
	spec, imports, err := buildDashboardSpec(model)
	require.NoError(t, err)
	assert.Equal(t, []string{"/v2/template/chart-a", "/v2/template/chart-b"}, imports)
	var document map[string]any
	require.NoError(t, json.Unmarshal(spec, &document))
	assert.Len(t, document[dashifyDashboardElement], 2)
	assert.Contains(t, document, "$import:widget0")
	assert.Contains(t, document, "$import:widget1")
	layout := document["layout"].(map[string]any)
	saved := layout["saved"].(map[string]any)
	items := saved["_"].(map[string]any)["items"].([]any)
	assert.Equal(t, "_.0", items[0].(map[string]any)["id"])
	assert.Equal(t, "1/2", items[0].(map[string]any)["w"])
	assert.NotContains(t, items[0].(map[string]any), "order")
	assert.NotContains(t, layout, "gap")
	root := template.RootElementDashboard
	parsed, diags := parseDashboardTemplate(&template.Template{ID: "dashboard-id", Title: "Service overview", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.Empty(t, diags, diags)
	assert.Equal(t, model.Title, parsed.Title)
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
					InlineContent: jsontypes.NewNormalizedValue(content),
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
				Type:     fwdashify.ObservabilityTemplateRecordType,
				Title:    "Dashboard",
				Spec:     spec,
				Metadata: &template.Metadata{RootElement: &root},
			})
			require.Empty(t, diags, diags)
			require.Len(t, parsed.Container, 1)
			assert.True(t, parsed.Container[0].TemplateID.IsNull())
			assert.JSONEq(t, content, parsed.Container[0].InlineContent.ValueString())

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
		"<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[{"key":"env","value":"prod"}],"futureContent":{"preserve":true}}],"futureContainer":true,"futureEmpty":{}}],
		"layout":{"gap":{},"saved":{"_":{"items":[{"id":"_.0","order":0,"absolute":true,"x":[1,2],"w":{"value":"1/2","futureEmpty":{}},"minW":2}]}}},
		"$import:widget0":"/v2/template/chart-id",
		"$import:orphan":"/v2/template/orphan-id"
	}`)
	model, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
	require.False(t, diags.HasError(), diags)
	require.Len(t, model.Container, 1)
	require.NotNil(t, model.Container[0].Layout)
	assert.Equal(t, "2", model.Container[0].Layout.MinWidth.ValueString())
	warnings := diags.Warnings()
	require.Len(t, warnings, 1)
	assert.Nil(t, dashboardKnownWriteMarker(diags))
	for _, path := range []string{"container.0.futureContainer", "container.0.futureContent.preserve", "container.0.futureEmpty", "container.0.<$import.widget0>", "$import:orphan", "layout._.0.order", "layout._.0.absolute", "layout._.0.x", "layout._.0.w.value", "layout._.0.w.futureEmpty", "layout.gap"} {
		assert.Contains(t, warnings[0].Detail(), path)
	}
	assert.NotContains(t, warnings[0].Detail(), "layout._.0.minW")
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

func TestDashifyDashboardSpecAllowsEmptyLayoutScaffold(t *testing.T) {
	t.Parallel()
	for _, layout := range []string{`{}`, `{"saved":{}}`, `{"saved":{"_":{}}}`} {
		t.Run(layout, func(t *testing.T) {
			t.Parallel()
			root := template.RootElementDashboard
			spec := json.RawMessage(`{"<Dashboard>":[{"<Panel>":[{"<$import.widget0>":[]}]}],"layout":` + layout + `,"$import:widget0":"/v2/template/chart-id"}`)
			model, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
			require.Empty(t, diags, diags)
			require.Len(t, model.Container, 1)
			assert.Nil(t, model.Container[0].Layout)
		})
	}
}

func TestDashifyDashboardSpecRejectsNestedContainers(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{"<Section>", "<Group>"} {
		t.Run(tag, func(t *testing.T) {
			t.Parallel()
			root := template.RootElementDashboard
			spec := json.RawMessage(`{"title":"Dashboard","<Dashboard>":[{"` + tag + `":[{"<Panel>":[{"<$import.widget0>":[]}]}]}],"$import:widget0":"/v2/template/chart-id"}`)
			_, diags := parseDashboardTemplate(&template.Template{Title: "Dashboard", Spec: spec, Metadata: &template.Metadata{RootElement: &root}})
			require.True(t, diags.HasError(), diags)
			assert.Contains(t, diags.Errors()[0].Detail(), tag)
			assert.Contains(t, diags.Errors()[0].Detail(), "signalfx_observability_template")
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
		"empty advanced width object": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","w":{}}]}}}`,
			want:   "unsupported empty w layout object",
		},
		"empty coordinate object": {
			layout: `{"saved":{"_":{"items":[{"id":"_.0","x":{}}]}}}`,
			want:   "unsupported empty x layout object",
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

func TestDashifyDashboardSpecMatchesLayoutByID(t *testing.T) {
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
				{"id":"_.1","absolute":false,"w":4},
				{"id":"_.0","w":8}
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

	rebuilt, _, err := buildDashboardSpec(model)
	require.NoError(t, err)
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
			assert.Equal(t, fwdashify.ObservabilityTemplateRecordType, write.Type)
			var record *template.Template
			switch {
			case write.Metadata.RootElement != nil && *write.Metadata.RootElement == chartRoot:
				assert.Equal(t, "Request rate", write.Title)
				chart = &template.Template{ID: "chart-1", Type: write.Type, Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &chartRoot}}
				record = chart
			case write.Metadata.RootElement != nil && *write.Metadata.RootElement == dashboardRoot:
				assert.Equal(t, "Service overview", write.Title)
				assert.Contains(t, string(write.Spec), `"controlBar"`)
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
			if write.Title == "Service overview" {
				assert.Contains(t, string(write.Spec), `"variableName":"service.component"`)
			} else {
				assert.Equal(t, "Updated service overview", write.Title)
				assert.NotContains(t, string(write.Spec), `"controlBar"`)
			}
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
				tfversion.SkipBelow(tfversion.Version1_5_0),
			},
			ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
				t,
				handlers,
				fwtest.WithMockResources(NewResourceObservabilityDashboard, fwdashify.NewResourceObservabilityTemplate),
			),
			Steps: []testresource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/00_observability_dashboard.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "title", "Service overview"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.time_range.default", "-PT15M"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.density.default", "60"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.0.variable_name", "service.name"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.0.property", "service.name"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.1.variable_name", "environment"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.1.property", "deployment.environment"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.filter_set.filter.#", "2"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.width", "1/2"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.height", "20"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.min_width", "4"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "container.0.layout.min_height", "2"),
						testresource.TestCheckResourceAttrPair(
							"signalfx_observability_dashboard.test", "container.0.template_id",
							"signalfx_observability_template.chart", "id",
						),
					),
				},
				{
					ResourceName:    "signalfx_observability_dashboard.test",
					ImportState:     true,
					ImportStateKind: testresource.ImportBlockWithID,
					GenerateConfig:  true,
					ImportPlanChecks: testresource.ImportPlanChecks{PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue("signalfx_observability_dashboard.generated", tfjsonpath.New("control_bar").AtMapKey("pinned_filter").AtSliceIndex(0).AtMapKey("variable_name"), knownvalue.StringExact("service.name")),
						plancheck.ExpectKnownValue("signalfx_observability_dashboard.generated", tfjsonpath.New("control_bar").AtMapKey("pinned_filter").AtSliceIndex(1).AtMapKey("variable_name"), knownvalue.StringExact("environment")),
					}},
				},
				{
					ConfigFile:      config.StaticFile("testdata/00_observability_dashboard.tf"),
					ConfigVariables: config.Variables{"filter_property": config.StringVariable("service.component")},
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.0.property", "service.component"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.0.variable_name", "service.component"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "control_bar.pinned_filter.1.variable_name", "environment"),
					),
				},
				{
					ConfigFile: config.StaticFile("testdata/01_observability_dashboard_updated.tf"),
					Check: testresource.ComposeAggregateTestCheckFunc(
						testresource.TestCheckResourceAttrSet("signalfx_observability_dashboard.test", "id"),
						testresource.TestCheckResourceAttr("signalfx_observability_dashboard.test", "title", "Updated service overview"),
						testresource.TestCheckNoResourceAttr("signalfx_observability_dashboard.test", "control_bar.time_range.default"),
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
    inline_content = jsonencode({ "<Chart>" = [] })
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
    inline_content = jsonencode({ "<Chart>" = [] })
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
					ID: "another-dashboard", Type: fwdashify.ObservabilityTemplateRecordType,
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
    inline_content = jsonencode({ "<Chart>" = [] })
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
			record: template.Template{ID: "dashboard-1", Type: fwdashify.ObservabilityTemplateRecordType, Spec: json.RawMessage(`{"<Dashboard>":[]}`)},
			want:   "no Dashboard root element metadata",
		},
		{
			name:   "wrong root",
			record: template.Template{ID: "dashboard-1", Type: fwdashify.ObservabilityTemplateRecordType, Metadata: &template.Metadata{RootElement: &chartRoot}, Spec: json.RawMessage(`{"<Dashboard>":[]}`)},
			want:   "not \"Dashboard\"",
		},
		{
			name:   "missing spec",
			record: template.Template{ID: "dashboard-1", Type: fwdashify.ObservabilityTemplateRecordType, Metadata: &template.Metadata{RootElement: &root}},
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
    inline_content = jsonencode({ "<Chart>" = [] })
  }]
}`,
			ExpectError: regexp.MustCompile("The Template API endpoint was not found"),
		}},
	})
}

func TestResourceObservabilityDashboardUpdateReportsNotFound(t *testing.T) {
	root := template.RootElementDashboard
	record := &template.Template{
		ID: "template-1", Type: fwdashify.ObservabilityTemplateRecordType, Title: "Service overview",
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
    inline_content = jsonencode({ "<Chart>" = [] })
  }]
}`},
			{
				Config: `resource "signalfx_observability_dashboard" "test" {
  title = "Updated overview"
  container = [{
    inline_content = jsonencode({ "<Chart>" = [] })
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
			InlineContent: jsontypes.NewNormalizedValue(content),
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
	assert.Contains(t, parsed.Container[0].InlineContent.ValueString(), "9007199254740993")
}

func TestDashboardInlineContentRejectsNestedImports(t *testing.T) {
	_, err := decodeDashifyInlineContent(`{"<Chart>":[{"<$import.hidden>":[]}]}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot contain import element")
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
