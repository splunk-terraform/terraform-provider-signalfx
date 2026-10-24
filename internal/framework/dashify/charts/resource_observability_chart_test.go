// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package charts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

const observabilityChartConfig = `resource "signalfx_observability_chart" "test" {
  title = "Request rate"
  metrics_time_series = {
    title = "Requests"
    program = "data('requests').publish()"
    time_range = "60m"
    visualization = "line"
  }
}`

func TestResourceObservabilityChartMetadataAndSchema(t *testing.T) {
	t.Parallel()
	managed := NewResourceObservabilityChart()
	var metadata resource.MetadataResponse
	managed.Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, &metadata)
	assert.Equal(t, "signalfx_observability_chart", metadata.TypeName)
	var response resource.SchemaResponse
	managed.Schema(t.Context(), resource.SchemaRequest{}, &response)
	require.False(t, response.Schema.ValidateImplementation(t.Context()).HasError())
	assert.Empty(t, response.Schema.Blocks)
	assert.True(t, response.Schema.Attributes["id"].IsComputed())
	assert.True(t, response.Schema.Attributes["metrics_time_series"].IsOptional())
	assert.True(t, response.Schema.Attributes["metrics_time_series"].(schema.SingleNestedAttribute).Attributes["program"].IsRequired())
	for _, chart := range managed.(*observabilityChartResource).engine.Charts() {
		assert.True(t, response.Schema.Attributes[chart.Name].IsOptional(), chart.Name)
	}
	for _, name := range []string{"metrics_time_series", "metrics_single_value", "metrics_list", "metrics_table", "metrics_cluster_map", "text"} {
		assert.Contains(t, response.Schema.Attributes, name)
	}
	assert.Len(t, response.Schema.Attributes, len(managed.(*observabilityChartResource).engine.Charts())+2)
}

func TestResourceObservabilityChartRejectsInvalidPlans(t *testing.T) {
	for _, test := range []struct{ name, config, want string }{
		{"blank title", strings.Replace(observabilityChartConfig, `title = "Request rate"`, `title = "   "`, 1), "title must contain"},
		{"missing chart", `resource "signalfx_observability_chart" "test" { title = "Chart" }`, "exactly one chart type"},

		{"multiple charts", `resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = { program = "A" }
  text = { markdown = "Notes" }
}`, "exactly one chart type"},
		{"list scale needs thresholds", `resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_list = { program = "A", color_by = "Scale" }
}`, "color_scale must be set"},
		{"cluster thresholds need scale", `resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_cluster_map = { program = "A", color_scale = [{ gte = 1 }] }
}`, "color_scale requires color_by"},
		{"cluster conflicting lower bounds", `resource "terraform_data" "unknown" { input = "A" }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_cluster_map = {
    program = terraform_data.unknown.output
    color_by = "Scale"
    color_scale = [{ gt = 1, gte = 1 }]
  }
}`, "conflicts"},
		{"table missing column field", `resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_table = { program = "A", columns = [{ header_name = "Value" }] }
}`, "field"},
		{"missing program", `resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = {}
}`, "program"},
		{"invalid visualization", strings.Replace(observabilityChartConfig, `visualization = "line"`, `visualization = "pie"`, 1), "must be one of"},
		{"invalid duration", strings.Replace(observabilityChartConfig, `time_range = "60m"`, `time_range = "-1h"`, 1), "must"},
		{"known invalid with unknown title and series", `resource "terraform_data" "unknown" { input = "Chart" }
resource "terraform_data" "series" { input = { A = { name = "Requests" } } }
resource "signalfx_observability_chart" "test" {
  title = terraform_data.unknown.output
  metrics_time_series = {
    program = "A"
    visualization = "pie"
    series = terraform_data.series.output
  }
}`, "must be one of"},
		{"known unit conflict with unknown links", `resource "terraform_data" "links" { input = [{ url = "https://example.com" }] }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = {
    program = "A"
    series = { A = { display_unit = { unit = "Second", suffix = "/s" } } }
    links = terraform_data.links.output
  }
}`, "variants conflict"},
		{"known empty unit with unknown links", `resource "terraform_data" "links" { input = [{ url = "https://example.com" }] }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = {
    program = "A"
    series = { A = { display_unit = {} } }
    links = terraform_data.links.output
  }
}`, "must set at least one"},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceObservabilityChart)),
				Steps: []testresource.TestStep{{Config: test.config, PlanOnly: true, ExpectError: regexp.MustCompile(test.want)}},
			})
		})
	}
}

func TestResourceObservabilityChartPlansUnknownStructures(t *testing.T) {
	for _, test := range []struct{ name, config string }{
		{"whole chart", `resource "terraform_data" "chart" { input = { program = "A" } }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = terraform_data.chart.output
}`},
		{"program", `resource "terraform_data" "program" { input = "A" }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = { program = terraform_data.program.output }
}`},
		{"nested map and lists", `resource "terraform_data" "series" { input = { A = { name = "Requests" } } }
resource "terraform_data" "links" { input = [{ url = "https://example.com" }] }
resource "terraform_data" "axes" { input = [{ label = "Requests" }] }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = {
    program = "A"
    series = terraform_data.series.output
    links = terraform_data.links.output
    y_axes = terraform_data.axes.output
  }
}`},
		{"nested object", `resource "terraform_data" "unit" { input = { unit = "Second" } }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  metrics_time_series = {
    program = "A"
    series = { A = { display_unit = terraform_data.unit.output } }
  }
}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceObservabilityChart)),
				Steps: []testresource.TestStep{{Config: test.config, PlanOnly: true, ExpectNonEmptyPlan: true}},
			})
		})
	}
}

func TestResourceObservabilityChartRejectsResolvedInvalidContent(t *testing.T) {
	for _, test := range []struct{ name, content, want string }{
		{"metrics_time_series", `{ program = "A", visualization = "pie" }`, "Invalid chart value"},
		{"text", `null`, "Missing chart content"},
	} {
		t.Run(test.name, func(t *testing.T) {
			writes := 0
			handlers := map[string]http.Handler{"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writes++
				http.Error(w, "invalid content must not reach the API", http.StatusBadRequest)
			})}
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceObservabilityChart)),
				Steps: []testresource.TestStep{{
					Config: fmt.Sprintf(`resource "terraform_data" "chart" { input = %s }
resource "signalfx_observability_chart" "test" {
  title = "Chart"
  %s = terraform_data.chart.output
}`, test.content, test.name),
					ExpectError: regexp.MustCompile(test.want),
				}},
			})
			assert.Zero(t, writes)
		})
	}
}

func TestResourceObservabilityChartImportsAndUpdatesUnmodeledContent(t *testing.T) {
	root := template.RootElementChart
	record := &template.Template{ID: "chart-1", Title: "Imported", Metadata: &template.Metadata{RootElement: &root}, Spec: json.RawMessage(`{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"chart":{"seriesOptions":{"A":{"name":"Requests","aggregateMode":"average"}},"yAxes":[{"label":"Requests","width":80}],"legend":{"aggregateMode":"sum"}},"widget":{}}]}`)}
	writes := 0
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			var write template.CreateUpdateTemplateRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&write)) {
				http.Error(w, "invalid request", 400)
				return
			}
			writes++
			assert.Equal(t, "Changed", write.Title)
			assert.JSONEq(t, `{"<Chart>":[{"<o11y:TimeSeriesChart>":[],"datasource":{"program":"A"},"chart":{"seriesOptions":{"A":{"name":"Requests"}},"yAxes":[{"label":"Requests"}]},"widget":{}}]}`, string(write.Spec))
			record.Title, record.Spec = write.Title, write.Spec
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	const importedConfig = `resource "signalfx_observability_chart" "test" {
  title = "Imported"
  metrics_time_series = {
    program = "A"
    series = { A = { name = "Requests" } }
    y_axes = [{ label = "Requests" }]
  }
}`
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceObservabilityChart)),
		Steps: []testresource.TestStep{
			{Config: importedConfig + `
import {
  to = signalfx_observability_chart.test
  id = "chart-1"
}`, Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "metrics_time_series.series.A.name", "Requests"),
				testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "metrics_time_series.y_axes.0.label", "Requests"),
			)},
			{Config: importedConfig, PlanOnly: true},
			{Config: strings.Replace(importedConfig, "Imported", "Changed", 1)},
			{Config: strings.Replace(importedConfig, "Imported", "Changed", 1), PlanOnly: true},
		},
	})
	assert.Equal(t, 1, writes, "only the title update should write")
}

func TestResourceObservabilityChartAPIResponses(t *testing.T) {
	root := template.RootElementChart
	wrongType := &template.Result{Data: &template.Template{ID: "chart-1", Spec: json.RawMessage(`{"<Chart>":[{"<o11y:TableChart>":[],"chart":{},"datasource":{"program":"A"},"widget":{}}]}`), Metadata: &template.Metadata{RootElement: &root}}}
	for _, test := range []struct {
		name, method string
		status       int
		response     *template.Result
		want         string
		removed      bool
		retainedID   string
	}{
		{"create missing endpoint", "POST", 404, nil, "endpoint was not found", false, ""},
		{"create missing data", "POST", 201, &template.Result{}, "no template record", false, ""},
		{"create malformed document", "POST", 201, &template.Result{Data: &template.Template{ID: "chart-1", Spec: json.RawMessage(`{}`), Metadata: &template.Metadata{RootElement: &root}}}, "exactly one chart", false, "chart-1"},
		{"create invalid ID", "POST", 201, &template.Result{Data: &template.Template{ID: "../chart-1"}}, "for Chart", false, ""},
		{"create wrong chart type", "POST", 201, wrongType, "chart type", false, "chart-1"},
		{"read deleted", "GET", 404, nil, "", true, ""},
		{"read failure", "GET", 500, nil, "", false, ""},
		{"read mismatched ID", "GET", 200, &template.Result{Data: &template.Template{ID: "wrong"}}, "for Chart", false, ""},
		{"update deleted", "PUT", 404, nil, "not found during update", false, ""},
		{"update mismatched ID", "PUT", 200, &template.Result{Data: &template.Template{ID: "wrong"}}, "for Chart", false, ""},
		{"update missing data", "PUT", 200, &template.Result{}, "no template record", false, ""},
		{"update wrong chart type", "PUT", 200, wrongType, "chart type", false, ""},
		{"delete already missing", "DELETE", 404, nil, "", false, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoint := test.method + " /v2/template/{id}"
			if test.method == "POST" {
				endpoint = "POST /v2/template"
			}
			calls := 0
			managed, resourceSchema := configuredObservabilityChartResource(t, map[string]http.Handler{
				endpoint: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if test.method != "POST" {
						assert.Equal(t, "chart-1", r.PathValue("id"))
					}
					if test.status >= 400 {
						http.Error(w, "test API error", test.status)
						return
					}
					w.WriteHeader(test.status)
					assert.NoError(t, json.NewEncoder(w).Encode(test.response))
				}),
			})
			model := observabilityChartTestModel(t)
			state := tfsdk.State{Schema: resourceSchema}
			require.False(t, managed.setState(t.Context(), &state, model).HasError())
			plan := tfsdk.Plan{Schema: resourceSchema}
			plan.Raw = state.Raw
			var details string
			switch test.method {
			case "POST":
				require.False(t, plan.SetAttribute(t.Context(), path.Root("id"), types.StringUnknown()).HasError())
				response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema, Raw: tftypes.NewValue(resourceSchema.Type().TerraformType(t.Context()), nil)}}
				managed.Create(t.Context(), resource.CreateRequest{Plan: plan}, &response)
				require.True(t, response.Diagnostics.HasError())
				details = response.Diagnostics.Errors()[0].Detail()
				if test.retainedID == "" {
					assert.True(t, response.State.Raw.IsNull())
				} else {
					retained, diags := managed.model(t.Context(), response.State)
					require.False(t, diags.HasError(), diags)
					assert.Equal(t, test.retainedID, retained.ID.ValueString())
					assert.True(t, retained.Content.Equal(model.Content), "failed create must preserve planned content")
				}
			case "GET":
				response := resource.ReadResponse{State: state}
				managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)
				assert.Equal(t, test.removed, response.State.Raw.IsNull())
				if !test.removed {
					require.True(t, response.Diagnostics.HasError())
					assert.True(t, response.State.Raw.Equal(state.Raw), "failed read must preserve prior state")
					details = response.Diagnostics.Errors()[0].Detail()
				}
			case "PUT":
				response := resource.UpdateResponse{State: state}
				managed.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &response)
				require.True(t, response.Diagnostics.HasError())
				assert.True(t, response.State.Raw.Equal(state.Raw), "failed update must preserve prior state")
				details = response.Diagnostics.Errors()[0].Detail()
			case "DELETE":
				response := resource.DeleteResponse{State: state}
				managed.Delete(t.Context(), resource.DeleteRequest{State: state}, &response)
				assert.False(t, response.Diagnostics.HasError(), response.Diagnostics)
				assert.Empty(t, response.Diagnostics.Warnings())
			}
			assert.Contains(t, details, test.want)
			assert.Equal(t, 1, calls)
		})
	}
}

func configuredObservabilityChartResource(t *testing.T, handlers map[string]http.Handler) (*observabilityChartResource, schema.Schema) {
	t.Helper()
	mock := fwtest.NewMock(t, handlers)
	var providerResponse provider.ConfigureResponse
	mock.Configure(t.Context(), provider.ConfigureRequest{}, &providerResponse)
	managed := NewResourceObservabilityChart().(*observabilityChartResource)
	var configureResponse resource.ConfigureResponse
	managed.Configure(t.Context(), resource.ConfigureRequest{ProviderData: providerResponse.ResourceData}, &configureResponse)
	require.False(t, configureResponse.Diagnostics.HasError())
	var schemaResponse resource.SchemaResponse
	managed.Schema(t.Context(), resource.SchemaRequest{}, &schemaResponse)
	return managed, schemaResponse.Schema
}

func TestResourceObservabilityChartRecreatesAfterRemoteDelete(t *testing.T) {
	root := template.RootElementChart
	var record *template.Template
	creates := 0
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&write)) {
				http.Error(w, "bad request", 400)
				return
			}
			creates++
			record = &template.Template{ID: fmt.Sprintf("chart-%d", creates), Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &root}}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if record == nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			assert.Equal(t, record.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "chart-2", r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceObservabilityChart)),
		Steps: []testresource.TestStep{
			{Config: observabilityChartConfig},
			{PreConfig: func() { record = nil }, Config: observabilityChartConfig, Check: testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "id", "chart-2")},
		},
	})
	assert.Equal(t, 2, creates)
}

// These fixtures exercise each chart's distinct wire mapping: negation, maps,
// flattened unit unions, string arrays, and metric-free
// text. The HTTP expectations are independent documents, not engine output.
var observabilityChartExamples = []struct {
	name, content, child, checkPath, checkValue string
}{
	{"metrics_time_series", `{
  program = "A"
  headerless = true
  time_range = "60m"
  visualization = "line"
  links = [{ url = "https://example.com" }]
  show_legend = false
  series = { A = { name = "Requests", display_unit = { unit = "Byte" } } }
  y_axes = [{ label = "Rate", minimum = 0.5 }]
  additional_properties = [{ property = "host", enabled = false }]
 }`, `{"<o11y:TimeSeriesChart>":[],"widget":{"headerless":true,"links":[{"url":"https://example.com"}]},"datasource":{"program":"A","time":{"type":"relative","range":3600000,"rangeEnd":0}},"chart":{"chartOptions":{"type":"line"},"legend":{"hide":true},"seriesOptions":{"A":{"name":"Requests","displayUnit":"Byte"}},"yAxes":[{"label":"Rate","min":0.5}],"additionalProperties":[{"property":"host","enabled":false}]}}`, "series.A.display_unit.unit", "Byte"},
	{"metrics_single_value", `{
  program = "A"
  headerless = true
  display_unit = { prefix = "$", suffix = "/s" }
  secondary_visualization = "Sparkline"
  color_scale = [{ gte = 0.5, lt = 100.5, palette_index = 2 }]
 }`, `{"<o11y:SingleValue>":[],"widget":{"headerless":true},"datasource":{"program":"A"},"chart":{"displayUnit":{"prefix":"$","suffix":"/s"},"secondaryVisualization":"Sparkline","colorScale":[{"gte":0.5,"lt":100.5,"paletteIndex":2}]}}`, "display_unit.suffix", "/s"},
	{"metrics_list", `{
  program = "A"
  headerless = true
  color_by = "Scale"
  color_scale = [{ gte = 0.5, palette_index = 2 }]
  published_streams = { A = { name = "Requests", display_unit = { unit = "Second" }, palette_index = 3 } }
  display_fields = [{ property = "host", enabled = true }]
  sort = { by = "value", direction = "desc" }
 }`, `{"<o11y:List>":[],"widget":{"headerless":true},"datasource":{"program":"A"},"chart":{"colorBy":"Scale","colorScale":[{"gte":0.5,"paletteIndex":2}],"publishedStreams":{"A":{"name":"Requests","displayUnit":"Second","paletteIndex":3}},"displayFields":[{"property":"host","enabled":true}],"sort":{"by":"value","direction":"desc"}}}`, "sort.direction", "desc"},
	{"metrics_table", `{
  program = "A"
  headerless = true
  group_by = ["host"]
  disable_sampling = true
  columns = [{ field = "A", header_name = "Requests", enabled = false, display_unit = { suffix = "/s" } }]
 }`, `{"<o11y:TableChart>":[],"widget":{"headerless":true},"datasource":{"program":"A"},"chart":{"groupBy":["host"],"disableSampling":true,"columns":[{"field":"A","headerName":"Requests","enabled":false,"displayUnit":{"suffix":"/s"}}]}}`, "columns.0.display_unit.suffix", "/s"},
	{"metrics_cluster_map", `{
  program = "A"
  headerless = true
  group_by = ["host", "region"]
  color_by = "Scale"
  color_scale = [{ gte = 0.5, lt = 5.5, palette_index = 2 }]
  display_unit = { suffix = "%" }
 }`, `{"<o11y:ClusterMap>":[],"widget":{"headerless":true},"datasource":{"program":"A"},"chart":{"groupBy":["host","region"],"colorBy":"Scale","colorScale":[{"gte":0.5,"lt":5.5,"paletteIndex":2}],"displayUnit":{"suffix":"%"}}}`, "group_by.1", "region"},
	{"text", `{
  headerless = true
  title = "Runbook"
  markdown = "## Instructions\n[Runbook](https://example.com/runbook)"
 }`, `{"<o11y:Text>":[],"widget":{"headerless":true,"title":"Runbook"},"chart":{"markdown":"## Instructions\n[Runbook](https://example.com/runbook)"}}`, "title", "Runbook"},
}

func TestResourceObservabilityChartAllTypesLifecycleAndComputedContent(t *testing.T) {
	for _, example := range observabilityChartExamples {
		t.Run(example.name, func(t *testing.T) {
			root := template.RootElementChart
			var record *template.Template
			writes, deletes := 0, 0
			updatedChild := strings.NewReplacer(`"headerless":true`, `"headerless":false`, `"type":"line"`, `"type":"area"`).Replace(example.child)
			handlers := map[string]http.Handler{
				"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var write template.CreateUpdateTemplateRequest
					if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&write)) {
						http.Error(w, "invalid chart request", http.StatusBadRequest)
						return
					}
					writes++
					assert.JSONEq(t, `{"<Chart>":[`+example.child+`]}`, string(write.Spec))
					assert.Equal(t, "Saved", write.Title)
					assert.Equal(t, fwdashify.ObservabilityTemplateRecordType, write.Type)
					assert.Equal(t, &root, write.Metadata.RootElement)
					assert.Empty(t, write.Metadata.Imports)
					assert.Nil(t, write.Metadata.Datasource)
					record = &template.Template{ID: "chart-1", Title: write.Title, Type: write.Type, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &root}}
					w.WriteHeader(http.StatusCreated)
					assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
				}),
				"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "chart-1", r.PathValue("id"))
					assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
				}),
				"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var write template.CreateUpdateTemplateRequest
					if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&write)) {
						http.Error(w, "invalid chart request", http.StatusBadRequest)
						return
					}
					writes++
					assert.Equal(t, "Updated", write.Title)
					assert.Equal(t, "chart-1", r.PathValue("id"))
					assert.JSONEq(t, `{"<Chart>":[`+updatedChild+`]}`, string(write.Spec))
					record.Title, record.Spec = write.Title, write.Spec
					assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
				}),
				"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "chart-1", r.PathValue("id"))
					deletes++
					w.WriteHeader(http.StatusNoContent)
				}),
			}
			config := fmt.Sprintf(`resource "terraform_data" "title" { input = "Saved" }
resource "terraform_data" "chart" { input = %s }
resource "signalfx_observability_chart" "test" {
  title = terraform_data.title.output
  %s = terraform_data.chart.output
}`, example.content, example.name)
			updated := strings.NewReplacer(`input = "Saved"`, `input = "Updated"`, "headerless = true", "headerless = false", `visualization = "line"`, `visualization = "area"`).Replace(config)
			check := testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "id", "chart-1"),
				testresource.TestCheckResourceAttr("signalfx_observability_chart.test", example.name+"."+example.checkPath, example.checkValue),
			)
			if example.name == "metrics_time_series" {
				check = testresource.ComposeAggregateTestCheckFunc(check,
					testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "metrics_time_series.time_range", "60m"))
			}
			steps := []testresource.TestStep{
				{Config: config, Check: check},
				{Config: config, PlanOnly: true},
				{ResourceName: "signalfx_observability_chart.test", ImportState: true, ImportStateKind: testresource.ImportBlockWithID, GenerateConfig: true},
				{Config: updated, Check: testresource.TestCheckResourceAttr("signalfx_observability_chart.test", example.name+".headerless", "false")},
			}
			expectedWrites := 2
			if example.name == "metrics_time_series" {
				// Equivalent duration spellings stay stable on refresh and can
				// each be explicitly configured without changing the wire value.
				steps = append(steps,
					testresource.TestStep{Config: strings.Replace(updated, `time_range = "60m"`, `time_range = "1h"`, 1), Check: testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "metrics_time_series.time_range", "1h")},
					testresource.TestStep{Config: updated, Check: check},
				)
				expectedWrites += 2
			}
			steps = append(steps, testresource.TestStep{Config: updated, PlanOnly: true})
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true, TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_5_0)},
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceObservabilityChart)),
				Steps:                    steps,
			})
			assert.Equal(t, expectedWrites, writes)
			assert.Equal(t, 1, deletes)
		})
	}
}

func TestResourceObservabilityChartDocumentedExamples(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join("..", "..", "..", "..", "examples", "resources", "observability_chart", "*.tf"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(examples), 6)
	for _, example := range examples {
		t.Run(filepath.Base(example), func(t *testing.T) {
			config, err := os.ReadFile(example)
			require.NoError(t, err)
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest: true, ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceObservabilityChart)),
				// The first example also contains Dashboard placement. Its package
				// imports this one, so keep this test focused on chart declarations.
				Steps: []testresource.TestStep{{Config: strings.SplitN(string(config), `resource "signalfx_observability_dashboard"`, 2)[0], PlanOnly: true, ExpectNonEmptyPlan: true}},
			})
		})
	}
}

func TestResourceObservabilityChartReadsChangedType(t *testing.T) {
	root := template.RootElementChart
	managed, resourceSchema := configuredObservabilityChartResource(t, map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "chart-1", r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &template.Template{
				ID: "chart-1", Title: "Changed in UI", Metadata: &template.Metadata{RootElement: &root},
				Spec: json.RawMessage(`{"<Chart>":[{"<o11y:Text>":[],"widget":{},"chart":{"markdown":"New notes"}}]}`),
			}}))
		}),
	})
	state := tfsdk.State{Schema: resourceSchema}
	require.False(t, managed.setState(t.Context(), &state, observabilityChartTestModel(t)).HasError())
	response := resource.ReadResponse{State: state}
	managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	model, diags := managed.model(t.Context(), response.State)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "text", model.Name)
	assert.Equal(t, "Changed in UI", model.Title.ValueString())
	assert.Equal(t, "New notes", model.Content.Attributes()["markdown"].(types.String).ValueString())
	var old types.Object
	require.False(t, response.State.GetAttribute(t.Context(), path.Root("metrics_time_series"), &old).HasError())
	assert.True(t, old.IsNull(), "an out-of-band chart type change must clear the previous content")
}

func TestResourceObservabilityChartReadsUnlistedVisualization(t *testing.T) {
	root := template.RootElementChart
	deletes := 0
	managed, resourceSchema := configuredObservabilityChartResource(t, map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "chart-1", r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: &template.Template{
				ID: "chart-1", Title: "Changed in UI", Metadata: &template.Metadata{RootElement: &root},
				Spec: json.RawMessage(strings.Replace(observabilityChartFixture, `"type":"line"`, `"type":"pie"`, 1)),
			}}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "chart-1", r.PathValue("id"))
			deletes++
			w.WriteHeader(http.StatusNoContent)
		}),
	})
	for _, imported := range []bool{false, true} {
		state := tfsdk.State{Schema: resourceSchema, Raw: tftypes.NewValue(resourceSchema.Type().TerraformType(t.Context()), nil)}
		if imported {
			require.False(t, state.SetAttribute(t.Context(), path.Root("id"), "chart-1").HasError())
		} else {
			require.False(t, managed.setState(t.Context(), &state, observabilityChartTestModel(t)).HasError())
		}
		response := resource.ReadResponse{State: state}
		managed.Read(t.Context(), resource.ReadRequest{State: state}, &response)
		require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
		require.Len(t, response.Diagnostics.Warnings(), 1)
		assert.Contains(t, response.Diagnostics.Warnings()[0].Detail(), "visualization")
		model, diags := managed.model(t.Context(), response.State)
		require.False(t, diags.HasError(), diags)
		assert.Equal(t, "chart-1", model.ID.ValueString())
		assert.Equal(t, "pie", model.Content.Attributes()["visualization"].(types.String).ValueString())
		deleted := resource.DeleteResponse{State: response.State}
		managed.Delete(t.Context(), resource.DeleteRequest{State: response.State}, &deleted)
		require.False(t, deleted.Diagnostics.HasError(), deleted.Diagnostics)
	}
	assert.Equal(t, 2, deletes)
}

func TestResourceObservabilityChartChangesType(t *testing.T) {
	root := template.RootElementChart
	var record *template.Template
	writes := 0
	handlers := map[string]http.Handler{
		"POST /v2/template": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var write template.CreateUpdateTemplateRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&write)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			writes++
			assert.JSONEq(t, observabilityChartFixture, string(write.Spec))
			record = &template.Template{ID: "chart-1", Title: write.Title, Spec: write.Spec, Metadata: &template.Metadata{RootElement: &root}}
			w.WriteHeader(http.StatusCreated)
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "chart-1", r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "chart-1", r.PathValue("id"))
			var write template.CreateUpdateTemplateRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&write)) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			writes++
			assert.JSONEq(t, `{"<Chart>":[{"<o11y:Text>":[],"widget":{},"chart":{"markdown":"Notes"}}]}`, string(write.Spec))
			record.Title, record.Spec = write.Title, write.Spec
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "chart-1", r.PathValue("id"))
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	const textConfig = `resource "signalfx_observability_chart" "test" {
  title = "Notes"
  text = { markdown = "Notes" }
}`
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true, ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceObservabilityChart)),
		Steps: []testresource.TestStep{
			{Config: observabilityChartConfig},
			{Config: textConfig, Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "id", "chart-1"),
				testresource.TestCheckNoResourceAttr("signalfx_observability_chart.test", "metrics_time_series"),
				testresource.TestCheckResourceAttr("signalfx_observability_chart.test", "text.markdown", "Notes"),
			)},
			{Config: textConfig, PlanOnly: true},
		},
	})
	assert.Equal(t, 2, writes)
}
