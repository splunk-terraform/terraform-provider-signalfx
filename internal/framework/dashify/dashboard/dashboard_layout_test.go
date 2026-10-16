// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dashboard

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fwdashify "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestDashboardPositionedPanelsCompactDeletedSlots(t *testing.T) {
	t.Parallel()
	root := template.RootElementDashboard
	record := &template.Template{
		Title: "Positioned panels", Metadata: &template.Metadata{RootElement: &root},
		Spec: json.RawMessage(`{
			"<Dashboard>":[null,{"<Panel>":[{"<$import.left>":[]}]},null,{"<Panel>":[{"<$import.right>":[]}]},null],
			"$import:left":"/v2/template/chart-left",
			"$import:right":"/v2/template/chart-right",
			"layout":{"saved":{"_":{"items":[
				{"id":"_.3","x":"1/2","w":"1/2","h":26},
				{"id":"_.1","x":0,"w":"1/2","h":26}
			]}}}
		}`),
	}
	model, diags := parseDashboardTemplate(record)
	require.Empty(t, diags, diags)
	require.Len(t, model.Container, 2)
	assert.Equal(t, "chart-left", model.Container[0].TemplateID.ValueString())
	assert.Equal(t, "0", model.Container[0].Layout.X.ValueString())
	assert.Equal(t, "chart-right", model.Container[1].TemplateID.ValueString())
	assert.Equal(t, "1/2", model.Container[1].Layout.X.ValueString())
	rebuilt, _, err := buildDashboardSpec(model)
	require.NoError(t, err)
	record.Spec = rebuilt
	roundTrip, diags := parseDashboardTemplate(record)
	require.Empty(t, diags, diags)
	assert.Equal(t, model, roundTrip)
	assert.NotContains(t, string(rebuilt), "null")
}

func TestResourceObservabilityDashboardPositionedImportUpdate(t *testing.T) {
	root := template.RootElementDashboard
	record := &template.Template{
		ID: "positioned-dashboard", Title: "Positioned panels", Type: fwdashify.ObservabilityTemplateRecordType,
		Metadata: &template.Metadata{RootElement: &root},
		Spec: json.RawMessage(`{
			"<Dashboard>":[{"<Panel>":[{"<$import.left>":[]}]},null,{"<Panel>":[{"<$import.right>":[]}]},null],
			"$import:left":"/v2/template/chart-left",
			"$import:right":"/v2/template/chart-right",
			"layout":{"saved":{"_":{"items":[{"id":"_.0","absolute":true,"x":0,"y":0,"w":"1/2","h":26},{"id":"_.2","absolute":true,"x":"1/2","y":26,"w":"1/2","h":26}]}}}
		}`),
	}
	writes := 0
	handlers := map[string]http.Handler{
		"GET /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, record.ID, r.PathValue("id"))
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"PUT /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writes++
			assert.Equal(t, record.ID, r.PathValue("id"))
			var write template.CreateUpdateTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&write); err != nil {
				t.Errorf("decode dashboard update: %v", err)
				http.Error(w, "invalid update", http.StatusBadRequest)
				return
			}
			var spec map[string]any
			assert.NoError(t, json.Unmarshal(write.Spec, &spec))
			assert.Len(t, spec["<Dashboard>"], 2)
			items := spec["layout"].(map[string]any)["saved"].(map[string]any)["_"].(map[string]any)["items"].([]any)
			assert.Equal(t, float64(0), items[0].(map[string]any)["x"], "numeric zero must survive the update")
			assert.Equal(t, float64(0), items[0].(map[string]any)["y"], "numeric zero must survive the update")
			assert.Equal(t, true, items[0].(map[string]any)["absolute"])
			if writes == 1 {
				assert.Equal(t, "25%", items[1].(map[string]any)["x"])
				assert.Equal(t, "1/2", items[1].(map[string]any)["y"])
				assert.Equal(t, true, items[1].(map[string]any)["absolute"])
			} else {
				assert.NotContains(t, items[1].(map[string]any), "x", "omitting x must restore automatic placement")
				assert.NotContains(t, items[1].(map[string]any), "y", "omitting y must restore automatic placement")
				assert.Equal(t, false, items[1].(map[string]any)["absolute"], "false must restore normal row flow")
			}
			assert.Equal(t, "1/2", items[1].(map[string]any)["w"])
			assert.Equal(t, []string{"/v2/template/chart-left", "/v2/template/chart-right"}, write.Metadata.Imports)
			record.Spec, record.Title = write.Spec, write.Title
			assert.NoError(t, json.NewEncoder(w).Encode(template.Result{Data: record}))
		}),
		"DELETE /v2/template/{id}": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	const importedConfig = `variable "right_position" {
  type = object({ x = string, y = string, absolute = bool })
  default = { x = "1/2", y = "26", absolute = true }
}
resource "signalfx_observability_dashboard" "test" {
  title = "Positioned panels"
  container = [
    { template_id = "chart-left", layout = { absolute = true, x = "0", y = "0", width = "1/2", height = "26" } },
    { template_id = "chart-right", layout = { absolute = var.right_position.absolute, x = var.right_position.x, y = var.right_position.y, width = "1/2", height = "26" } },
  ]
}`
	changedPosition := config.Variables{"right_position": config.ObjectVariable(map[string]config.Variable{
		"x": config.StringVariable("25%"), "y": config.StringVariable("1/2"), "absolute": config.BoolVariable(true),
	})}
	automaticPosition := config.Variables{"right_position": config.ObjectVariable(map[string]config.Variable{
		"x": json.RawMessage("null"), "y": json.RawMessage("null"), "absolute": config.BoolVariable(false),
	})}
	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest: true,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_5_0),
		},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceObservabilityDashboard)),
		Steps: []testresource.TestStep{
			{Config: `import {
  to = signalfx_observability_dashboard.test
  id = "positioned-dashboard"
}
` + importedConfig},
			{Config: importedConfig, PlanOnly: true},
			{ResourceName: "signalfx_observability_dashboard.test", ImportState: true, ImportStateKind: testresource.ImportBlockWithID, GenerateConfig: true},
			{Config: importedConfig, ConfigVariables: changedPosition},
			{Config: importedConfig, ConfigVariables: changedPosition, PlanOnly: true},
			{Config: importedConfig, ConfigVariables: automaticPosition},
			{Config: importedConfig, ConfigVariables: automaticPosition, PlanOnly: true},
		},
	})
	assert.Equal(t, 2, writes, "import and no-change plans must not write to the dashboard")
}
