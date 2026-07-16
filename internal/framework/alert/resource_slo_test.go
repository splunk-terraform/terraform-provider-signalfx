// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/signalfx/signalfx-go/slo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceSLOMetadataAndSchema(t *testing.T) {
	t.Parallel()
	metadata := &frameworkresource.MetadataResponse{}
	NewResourceSLO().Metadata(context.Background(), frameworkresource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_slo", metadata.TypeName)

	response := &frameworkresource.SchemaResponse{}
	NewResourceSLO().Schema(context.Background(), frameworkresource.SchemaRequest{}, response)
	require.False(t, response.Diagnostics.HasError())
	assert.Equal(t, int64(1), response.Schema.Version)
	assert.True(t, response.Schema.Attributes["id"].IsComputed())
	input, ok := response.Schema.Blocks["input"].(schema.ListNestedBlock)
	require.True(t, ok)
	target, ok := response.Schema.Blocks["target"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.NotEmpty(t, input.Validators)
	alertRule, ok := target.NestedObject.Blocks["alert_rule"].(schema.ListNestedBlock)
	require.True(t, ok)
	rule, ok := alertRule.NestedObject.Blocks["rule"].(schema.ListNestedBlock)
	require.True(t, ok)
	_, ok = rule.NestedObject.Blocks["parameters"].(schema.ListNestedBlock)
	assert.True(t, ok)
}

func TestResourceSLOModelRequestAndResponse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := testSLOModel(t, ctx)
	payload, diagnostics := model.request(ctx)
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, "checkout availability", payload.Name)
	require.Len(t, payload.Targets, 1)
	assert.Equal(t, "30d", payload.Targets[0].CompliancePeriod)
	require.Len(t, payload.Targets[0].SloAlertRules, 3)
	require.NotNil(t, payload.Targets[0].SloAlertRules[0].BreachSloAlertRule)
	require.NotNil(t, payload.Targets[0].SloAlertRules[1].BurnRateSloAlertRule)
	require.NotNil(t, payload.Targets[0].SloAlertRules[2].ErrorBudgetLeftSloAlertRule)
	require.Len(t, payload.Targets[0].SloAlertRules[0].BreachSloAlertRule.Rules[0].Notifications, 1)

	payload.Id = "slo-1"
	payload.Targets[0].SloAlertRules[0], payload.Targets[0].SloAlertRules[2] =
		payload.Targets[0].SloAlertRules[2], payload.Targets[0].SloAlertRules[0]
	diagnostics = model.updateFromAPI(ctx, payload)
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, "slo-1", model.ID.ValueString())
	targets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
	require.Len(t, targets, 1)
	alertRules := detectorListElements[sloAlertRuleModel](ctx, targets[0].AlertRules, &diagnostics)
	require.Len(t, alertRules, 3)
	assert.Equal(t, slo.BreachRule, alertRules[0].Type.ValueString(), "configured order must survive API reordering")
	assert.Equal(t, slo.BurnRateRule, alertRules[1].Type.ValueString())
	assert.Equal(t, slo.ErrorBudgetLeftRule, alertRules[2].Type.ValueString())
}

func TestResourceSLOCalendarAndResponseFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := testSLOModel(t, ctx)
	var diagnostics = model.updateFromAPI(ctx, nil)
	assert.True(t, diagnostics.HasError())

	for name, details := range map[string]*slo.SloObject{
		"missing id":      {BaseSlo: slo.BaseSlo{Type: slo.RequestBased}},
		"missing input":   {BaseSlo: slo.BaseSlo{Id: "x", Type: slo.RequestBased}, RequestBasedSlo: &slo.RequestBasedSlo{}},
		"missing target":  {BaseSlo: slo.BaseSlo{Id: "x", Type: slo.RequestBased}, RequestBasedSlo: &slo.RequestBasedSlo{Inputs: &slo.RequestBasedSloInput{}}},
		"unknown input":   {BaseSlo: slo.BaseSlo{Id: "x", Type: "WindowsBased"}},
		"extra targets":   testSLOResponseWithTargetCount(2),
		"unknown target":  testSLOResponseWithTarget(slo.SloTarget{BaseSloTarget: slo.BaseSloTarget{Type: "Unknown"}}),
		"missing rolling": testSLOResponseWithTarget(slo.SloTarget{BaseSloTarget: slo.BaseSloTarget{Type: slo.RollingWindowTarget}}),
	} {
		t.Run(name, func(t *testing.T) {
			copy := model
			assert.True(t, copy.updateFromAPI(ctx, details).HasError())
		})
	}

	calendar := testSLOModel(t, ctx)
	targets := detectorListElements[sloTargetModel](ctx, calendar.Target, &diagnostics)
	targets[0].Type = types.StringValue(slo.CalendarWindowTarget)
	targets[0].CompliancePeriod = types.StringNull()
	targets[0].CycleType = types.StringValue("week")
	targets[0].CycleStart = types.StringValue("sunday")
	calendar.Target = detectorListValue(ctx, sloTargetAttributeTypes, targets, &diagnostics)
	payload, requestDiagnostics := calendar.request(ctx)
	require.False(t, requestDiagnostics.HasError(), requestDiagnostics)
	assert.Equal(t, "week", payload.Targets[0].CycleType)
}

func TestResourceSLOLocalValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for name, mutate := range map[string]func(*resourceSLOModel){
		"missing input": func(model *resourceSLOModel) {
			model.Input = types.ListNull(types.ObjectType{AttrTypes: sloInputAttributeTypes})
		},
		"missing target": func(model *resourceSLOModel) {
			model.Target = types.ListNull(types.ObjectType{AttrTypes: sloTargetAttributeTypes})
		},
		"calendar needs cycle": func(model *resourceSLOModel) {
			var diagnostics = modelDiagnostics()
			targets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
			targets[0].Type = types.StringValue(slo.CalendarWindowTarget)
			targets[0].CompliancePeriod = types.StringNull()
			targets[0].CycleType = types.StringNull()
			model.Target = detectorListValue(ctx, sloTargetAttributeTypes, targets, &diagnostics)
		},
		"missing breach": func(model *resourceSLOModel) {
			var diagnostics = modelDiagnostics()
			targets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
			rules := detectorListElements[sloAlertRuleModel](ctx, targets[0].AlertRules, &diagnostics)
			targets[0].AlertRules = detectorListValue(ctx, sloAlertRuleAttributeTypes, rules[1:2], &diagnostics)
			model.Target = detectorListValue(ctx, sloTargetAttributeTypes, targets, &diagnostics)
		},
		"duplicate rule type": func(model *resourceSLOModel) {
			var diagnostics = modelDiagnostics()
			targets := detectorListElements[sloTargetModel](ctx, model.Target, &diagnostics)
			rules := detectorListElements[sloAlertRuleModel](ctx, targets[0].AlertRules, &diagnostics)
			rules[1].Type = types.StringValue(slo.BreachRule)
			targets[0].AlertRules = detectorListValue(ctx, sloAlertRuleAttributeTypes, rules, &diagnostics)
			model.Target = detectorListValue(ctx, sloTargetAttributeTypes, targets, &diagnostics)
		},
	} {
		t.Run(name, func(t *testing.T) {
			model := testSLOModel(t, ctx)
			mutate(&model)
			assert.True(t, model.validate(ctx).HasError())
		})
	}
}

func TestResourceSLOMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	var current *slo.SloObject
	var validationNames []string
	handlers := map[string]http.Handler{
		"POST /v2/slo/validate": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload slo.SloObject
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			mu.Lock()
			validationNames = append(validationNames, payload.Name)
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
		"POST /v2/slo": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload slo.SloObject
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			payload.Id = "slo-1"
			mu.Lock()
			current = &payload
			mu.Unlock()
			assert.NoError(t, json.NewEncoder(w).Encode(&payload))
		}),
		"GET /v2/slo/slo-1": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"PUT /v2/slo/slo-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload slo.SloObject
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			payload.Id = "slo-1"
			mu.Lock()
			current = &payload
			mu.Unlock()
			assert.NoError(t, json.NewEncoder(w).Encode(&payload))
		}),
		"DELETE /v2/slo/slo-1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		IsUnitTest:               true,
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.RequireAbove(tfversion.Version1_0_0)},
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceSLO)),
		Steps: []testresource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/slo_create.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_slo.test", "id", "slo-1"),
					testresource.TestCheckResourceAttr("signalfx_slo.test", "target.0.alert_rule.0.type", slo.BreachRule),
					testresource.TestCheckResourceAttr("signalfx_slo.test", "target.0.alert_rule.1.type", slo.BurnRateRule),
				),
			},
			{ResourceName: "signalfx_slo.test", ImportState: true, ImportStateVerify: true},
			{
				ConfigFile: config.StaticFile("testdata/slo_update.tf"),
				Check: testresource.ComposeAggregateTestCheckFunc(
					testresource.TestCheckResourceAttr("signalfx_slo.test", "description", "Updated checkout request success"),
					testresource.TestCheckResourceAttr("signalfx_slo.test", "target.0.slo", "99"),
					testresource.TestCheckResourceAttr("signalfx_slo.test", "target.0.alert_rule.1.type", slo.ErrorBudgetLeftRule),
				),
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, validationNames, "checkout availability")
	assert.True(t, containsString(validationNames, "terraform-validation-slo-1"))
	assert.NotNil(t, current)
	assert.Equal(t, "slo-1", current.Id)
}

func TestResourceSLOValidationAndCreateErrors(t *testing.T) {
	t.Parallel()
	for name, handlers := range map[string]map[string]http.Handler{
		"validation": {
			"POST /v2/slo/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "invalid signalflow", http.StatusBadRequest)
			}),
		},
		"create": {
			"POST /v2/slo/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
			"POST /v2/slo": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "create failed", http.StatusInternalServerError)
			}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, handlers, fwtest.WithMockResources(NewResourceSLO)),
				Steps: []testresource.TestStep{{
					ConfigFile:  config.StaticFile("testdata/slo_create.tf"),
					ExpectError: assertErrorRegexp(name),
				}},
			})
		})
	}
}

func testSLOModel(t *testing.T, ctx context.Context) resourceSLOModel {
	t.Helper()
	var diagnostics = modelDiagnostics()
	parameters := func(value sloParametersModel) types.List {
		return detectorListValue(ctx, sloParametersAttributeTypes, []sloParametersModel{value}, &diagnostics)
	}
	rule := func(severity, notification string, values sloParametersModel) sloRuleModel {
		notifications := types.ListNull(types.StringType)
		if notification != "" {
			notifications = types.ListValueMust(types.StringType, []attr.Value{types.StringValue(notification)})
		}
		return sloRuleModel{
			Severity: types.StringValue(severity), Description: types.StringValue(""), Notifications: notifications,
			Disabled: types.BoolValue(false), ParameterizedBody: types.StringValue(""), ParameterizedSubject: types.StringValue(""),
			RunbookURL: types.StringValue(""), Tip: types.StringValue(""),
			SkipClearNotificationStates: types.SetValueMust(types.StringType, nil), Parameters: parameters(values),
			ReminderNotification: types.ListNull(types.ObjectType{AttrTypes: detectorReminderAttributeTypes}),
		}
	}
	breach := rule("Critical", "Email,alerts@example.com", sloParametersModel{
		FireLasting: types.StringValue("5m"), PercentOfLasting: types.Float64Value(100),
	})
	burn := rule("Warning", "", sloParametersModel{
		ShortWindow1: types.StringValue("5m"), LongWindow1: types.StringValue("1h"),
		ShortWindow2: types.StringValue("30m"), LongWindow2: types.StringValue("6h"),
		BurnRateThreshold1: types.Float64Value(14.4), BurnRateThreshold2: types.Float64Value(6),
	})
	budget := rule("Major", "", sloParametersModel{
		FireLasting: types.StringValue("5m"), PercentOfLasting: types.Float64Value(100),
		PercentErrorBudgetLeft: types.Float64Value(12),
	})
	alertRules := []sloAlertRuleModel{
		{Type: types.StringValue(slo.BreachRule), Rules: detectorListValue(ctx, sloRuleAttributeTypes, []sloRuleModel{breach}, &diagnostics)},
		{Type: types.StringValue(slo.BurnRateRule), Rules: detectorListValue(ctx, sloRuleAttributeTypes, []sloRuleModel{burn}, &diagnostics)},
		{Type: types.StringValue(slo.ErrorBudgetLeftRule), Rules: detectorListValue(ctx, sloRuleAttributeTypes, []sloRuleModel{budget}, &diagnostics)},
	}
	target := sloTargetModel{
		Type: types.StringValue(slo.RollingWindowTarget), SLO: types.Float64Value(98),
		CompliancePeriod: types.StringValue("30d"), CycleType: types.StringNull(), CycleStart: types.StringNull(),
		AlertRules: detectorListValue(ctx, sloAlertRuleAttributeTypes, alertRules, &diagnostics),
	}
	model := resourceSLOModel{
		ID: types.StringNull(), Name: types.StringValue("checkout availability"),
		Description: types.StringValue("description"), Type: types.StringValue(slo.RequestBased),
		Input: detectorListValue(ctx, sloInputAttributeTypes, []sloInputModel{{
			ProgramText:     types.StringValue("G = data('good')\nT = data('total')"),
			GoodEventsLabel: types.StringValue("G"), TotalEventsLabel: types.StringValue("T"),
		}}, &diagnostics),
		Target: detectorListValue(ctx, sloTargetAttributeTypes, []sloTargetModel{target}, &diagnostics),
	}
	require.False(t, diagnostics.HasError(), diagnostics)
	return model
}

func testSLOResponseWithTargetCount(count int) *slo.SloObject {
	targets := make([]slo.SloTarget, count)
	return &slo.SloObject{
		BaseSlo:         slo.BaseSlo{Id: "x", Type: slo.RequestBased, Targets: targets},
		RequestBasedSlo: &slo.RequestBasedSlo{Inputs: &slo.RequestBasedSloInput{}},
	}
}

func testSLOResponseWithTarget(target slo.SloTarget) *slo.SloObject {
	return &slo.SloObject{
		BaseSlo:         slo.BaseSlo{Id: "x", Type: slo.RequestBased, Targets: []slo.SloTarget{target}},
		RequestBasedSlo: &slo.RequestBasedSlo{Inputs: &slo.RequestBasedSloInput{}},
	}
}

func containsString(values []string, substring string) bool {
	for _, value := range values {
		if strings.Contains(value, substring) {
			return true
		}
	}
	return false
}

func modelDiagnostics() diag.Diagnostics {
	return nil
}

func assertErrorRegexp(name string) *regexp.Regexp {
	if name == "validation" {
		return regexp.MustCompile("Invalid SLO configuration")
	}
	return regexp.MustCompile("create failed|status code 500")
}
