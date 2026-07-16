// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwalert

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func TestResourceDetectorMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewResourceDetector()
	metadata := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_detector", metadata.TypeName)

	response := &resource.SchemaResponse{}
	implementation.Schema(context.Background(), resource.SchemaRequest{}, response)
	require.False(t, response.Diagnostics.HasError())
	assert.Equal(t, int64(1), response.Schema.Version)
	assert.True(t, response.Schema.Attributes["name"].IsRequired())
	assert.True(t, response.Schema.Attributes["label_resolutions"].IsComputed())
	assert.Len(t, response.Schema.Attributes["detector_origin"].(schema.StringAttribute).PlanModifiers, 1)
	assert.Len(t, response.Schema.Attributes["parent_detector_id"].(schema.StringAttribute).PlanModifiers, 1)

	rule := response.Schema.Blocks["rule"].(schema.SetNestedBlock)
	assert.Len(t, rule.Validators, 1)
	assert.IsType(t, schema.ListAttribute{}, rule.NestedObject.Attributes["notifications"])
	assert.IsType(t, schema.SetAttribute{}, rule.NestedObject.Attributes["skip_clear_notification_states"])
	reminder := rule.NestedObject.Blocks["reminder_notification"].(schema.ListNestedBlock)
	assert.Len(t, reminder.Validators, 1)
	assert.IsType(t, schema.SetNestedBlock{}, response.Schema.Blocks["viz_options"])
}

func TestResourceDetectorModelToAPI(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := detectorTestModel(t, false)
	registry := feature.NewRegistry()
	registry.MustRegister(feature.PreviewProviderTags, feature.WithPreviewGlobalAvailable())
	registry.MustRegister(feature.PreviewProviderTeams, feature.WithPreviewGlobalAvailable())
	meta := &pmeta.Meta{Registry: registry, Tags: []string{"provider-tag", "resource-tag"}, Teams: []string{"provider-team", "resource-team"}}

	payload, diagnostics := model.request(ctx, meta)
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, "demo detector", payload.Name)
	assert.Equal(t, int32(30000), *payload.MaxDelay)
	assert.Equal(t, int32(15000), *payload.MinDelay)
	assert.Equal(t, []string{"provider-tag", "resource-tag"}, payload.Tags)
	assert.Equal(t, []string{"provider-team", "resource-team"}, payload.Teams)
	assert.Equal(t, []string{"writer-team"}, payload.AuthorizedWriters.Teams)
	assert.Equal(t, []string{"writer-user"}, payload.AuthorizedWriters.Users)
	require.Len(t, payload.Rules, 1)
	assert.Equal(t, detector.Severity("Critical"), payload.Rules[0].Severity)
	assert.Equal(t, "alerts@example.com", payload.Rules[0].Notifications[0].Value.(*notification.EmailNotification).Email)
	assert.Equal(t, []string{"AUTO_RESOLVED"}, payload.Rules[0].SkipClearNotificationStates)
	require.NotNil(t, payload.Rules[0].ReminderNotification)
	assert.Equal(t, int64(60000), payload.Rules[0].ReminderNotification.IntervalMs)
	require.NotNil(t, payload.VisualizationOptions.Time.Range)
	assert.Equal(t, int64(3600000), *payload.VisualizationOptions.Time.Range)
	require.Len(t, payload.VisualizationOptions.PublishLabelOptions, 1)
	assert.Equal(t, int32(1), *payload.VisualizationOptions.PublishLabelOptions[0].PaletteIndex)

	absolute := detectorTestModel(t, true)
	payload, diagnostics = absolute.request(ctx, nil)
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, "absolute", payload.VisualizationOptions.Time.Type)
	assert.Equal(t, int64(1700000000000), *payload.VisualizationOptions.Time.Start)
	assert.Equal(t, int64(1700003600000), *payload.VisualizationOptions.Time.End)

	validation, diagnostics := model.validationRequest(ctx)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, model.Name.ValueString(), validation.Name)
	assert.Equal(t, model.ProgramText.ValueString(), validation.ProgramText)
	require.Len(t, validation.Rules, 1)
}

func TestResourceDetectorModelRequestFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := detectorTestModel(t, false)
	model.MaxDelay = types.Int64Value(detectorMaximumDelayMilliseconds/1000 + 1)
	_, diagnostics := model.request(ctx, nil)
	require.True(t, diagnostics.HasError())
	assert.Contains(t, diagnostics.Errors()[0].Detail(), "32-bit")

	model = detectorTestModel(t, false)
	var conversionDiagnostics diag.Diagnostics
	var rules []detectorRuleModel
	require.False(t, model.Rules.ElementsAs(ctx, &rules, false).HasError())
	rules[0].Notifications = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("invalid")})
	model.Rules = detectorSetValue(ctx, detectorRuleAttributeTypes, rules, &conversionDiagnostics)
	_, diagnostics = model.request(ctx, nil)
	require.True(t, diagnostics.HasError())
	assert.Contains(t, diagnostics.Errors()[0].Summary(), "Invalid notification")

	model = detectorTestModel(t, false)
	var visualizations []detectorVisualizationModel
	require.False(t, model.VisualizationOptions.ElementsAs(ctx, &visualizations, false).HasError())
	visualizations[0].Color = types.StringValue("not-a-color")
	model.VisualizationOptions = detectorSetValue(ctx, detectorVisualizationAttributeTypes, visualizations, &conversionDiagnostics)
	_, diagnostics = model.request(ctx, nil)
	require.True(t, diagnostics.HasError())
	assert.Contains(t, diagnostics.Errors()[0].Summary(), "Invalid detector visualization color")

	assert.True(t, detectorKnownListOrNull(types.ListUnknown(types.StringType)).IsNull())
	assert.Equal(t, types.Int64Value(3), detectorNestedInt64FromAPI(types.Int64Null(), 3, false))
}

func TestResourceDetectorModelFromAPI(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := detectorTestModel(t, true)
	maxDelay, minDelay := int32(60000), int32(30000)
	start, end, paletteIndex := int64(1700000000123), int64(1700003600999), int32(8)
	resolutions := map[string]any{"CPU": json.Number("1000"), "Memory": float64(2000)}
	emailNotification, err := common.NewNotificationFromString("Email,api@example.com")
	require.NoError(t, err)

	diagnostics := model.updateFromAPI(ctx, &detector.Detector{
		Id: "detector-id", Name: "API detector", Description: "from API", ProgramText: "program", TimeZone: "UTC",
		MaxDelay: &maxDelay, MinDelay: &minDelay, Tags: []string{"api-tag"}, Teams: []string{"api-team"},
		DetectorOrigin: "AutoDetectCustomization", ParentDetectorId: "parent-id",
		AuthorizedWriters: &detector.AuthorizedWriters{Teams: []string{"writer-team"}, Users: []string{"writer-user"}},
		Rules: []*detector.Rule{{
			Severity: detector.Severity("Warning"), DetectLabel: "CPU", Description: "API rule", Notifications: []*notification.Notification{emailNotification},
			Disabled: true, ParameterizedBody: "body", ParameterizedSubject: "subject", RunbookUrl: "runbook", Tip: "tip",
			SkipClearNotificationStates: []string{"STOPPED"}, ReminderNotification: &detector.ReminderNotification{IntervalMs: 1, TimeoutMs: 2, Type: "TIMEOUT"},
		}},
		VisualizationOptions: &detector.Visualization{
			ShowDataMarkers: false, ShowEventLines: true, DisableSampling: true,
			Time:                &detector.Time{Type: "absolute", Start: &start, End: &end},
			PublishLabelOptions: []*detector.PublishLabelOptions{{Label: "CPU", PaletteIndex: &paletteIndex, DisplayName: "CPU API", ValueUnit: "Second", ValuePrefix: "pre", ValueSuffix: "post"}},
		},
		LabelResolutions: &resolutions,
	}, "https://app.example/#/detector/v2/detector-id/edit")
	require.False(t, diagnostics.HasError(), diagnostics)
	assert.Equal(t, types.StringValue("detector-id"), model.ID)
	assert.Equal(t, types.Int64Value(60), model.MaxDelay)
	assert.Equal(t, types.Int64Value(30), model.MinDelay)
	assert.Equal(t, types.Int64Value(1700000000), model.StartTime)
	assert.Equal(t, types.Int64Value(1700003600), model.EndTime)
	assert.True(t, model.TimeRange.IsNull())
	assert.Equal(t, "https://app.example/#/detector/v2/detector-id/edit", model.URL.ValueString())

	var rules []detectorRuleModel
	require.False(t, model.Rules.ElementsAs(ctx, &rules, false).HasError())
	require.Len(t, rules, 1)
	assert.Equal(t, "Email,api@example.com", rules[0].Notifications.Elements()[0].(types.String).ValueString())
	var reminders []detectorReminderNotificationModel
	require.False(t, rules[0].ReminderNotification.ElementsAs(ctx, &reminders, false).HasError())
	assert.Equal(t, int64(2), reminders[0].TimeoutMS.ValueInt64())
	var visualizations []detectorVisualizationModel
	require.False(t, model.VisualizationOptions.ElementsAs(ctx, &visualizations, false).HasError())
	assert.Equal(t, "red", visualizations[0].Color.ValueString())
	var labels map[string]int64
	require.False(t, model.LabelResolutions.ElementsAs(ctx, &labels, false).HasError())
	assert.Equal(t, map[string]int64{"CPU": 1000, "Memory": 2000}, labels)
}

func TestResourceDetectorModelResponseDefaultsAndFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := detectorTestModel(t, false)

	diagnostics := model.updateFromAPI(ctx, nil, "")
	require.True(t, diagnostics.HasError())
	assert.Contains(t, diagnostics.Errors()[0].Detail(), "no resource data")
	diagnostics = model.updateFromAPI(ctx, &detector.Detector{}, "")
	require.True(t, diagnostics.HasError())
	assert.Contains(t, diagnostics.Errors()[0].Detail(), "no resource identifier")

	diagnostics = model.updateFromAPI(ctx, &detector.Detector{Id: "id", Rules: []*detector.Rule{nil}}, "")
	require.True(t, diagnostics.HasError())
	assert.Contains(t, diagnostics.Errors()[0].Detail(), "empty")

	badIndex := int32(99)
	badResolution := map[string]any{"CPU": 1.5}
	diagnostics = model.updateFromAPI(ctx, &detector.Detector{
		Id: "id", Rules: []*detector.Rule{{Severity: "Critical", DetectLabel: "CPU"}},
		VisualizationOptions: &detector.Visualization{PublishLabelOptions: []*detector.PublishLabelOptions{nil, {Label: "CPU", PaletteIndex: &badIndex}}},
		LabelResolutions:     &badResolution,
	}, "")
	assert.True(t, diagnostics.HasError())
	assert.GreaterOrEqual(t, len(diagnostics.Errors()), 3)

	model = resourceDetectorModel{
		ShowDataMarkers: types.BoolUnknown(), ShowEventLines: types.BoolUnknown(), DisableSampling: types.BoolUnknown(),
		TimeRange: types.Int64Unknown(), StartTime: types.Int64Unknown(), EndTime: types.Int64Unknown(),
		AuthorizedWriterTeams: types.SetUnknown(types.StringType), AuthorizedWriterUsers: types.SetUnknown(types.StringType),
		VisualizationOptions: types.SetUnknown(types.ObjectType{AttrTypes: detectorVisualizationAttributeTypes}),
	}
	diagnostics = model.updateFromAPI(ctx, &detector.Detector{Id: "id", Rules: []*detector.Rule{{Severity: "Critical", DetectLabel: "CPU"}}}, "")
	require.False(t, diagnostics.HasError())
	assert.Equal(t, types.Int64Value(3600), model.TimeRange)
	assert.Equal(t, types.BoolValue(true), model.ShowDataMarkers)
	assert.True(t, model.StartTime.IsNull())
	assert.True(t, model.EndTime.IsNull())
	assert.True(t, model.AuthorizedWriterTeams.IsNull() || len(model.AuthorizedWriterTeams.Elements()) == 0)
}

func TestDetectorNumericAndValueHelpers(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		value any
		want  int64
		ok    bool
	}{
		{int(1), 1, true}, {int32(2), 2, true}, {int64(3), 3, true}, {float64(4), 4, true},
		{json.Number("5"), 5, true}, {float64(1.5), 0, false}, {json.Number("bad"), 0, false}, {"6", 0, false},
	} {
		got, ok := detectorNumericInt64(test.value)
		assert.Equal(t, test.ok, ok)
		assert.Equal(t, test.want, got)
	}
	assert.Equal(t, types.Int64Null(), detectorOptionalMillisecondsAsSeconds(nil))
	assert.Equal(t, types.Int64Value(2), detectorOptionalMillisecondsAsSeconds(pointer(int64(2999))))
	assert.Equal(t, types.Int64Value(7), detectorDelayFromAPI(types.Int64Unknown(), pointer(int32(7000))))
	assert.Equal(t, types.Int64Value(9), detectorDelayFromAPI(types.Int64Value(9), nil))
	assert.Equal(t, types.StringValue("api"), detectorResponseString(types.StringValue("current"), "api", "fallback"))
	assert.Equal(t, types.StringValue("current"), detectorResponseString(types.StringValue("current"), "", "fallback"))
	assert.Equal(t, types.StringValue("fallback"), detectorResponseString(types.StringNull(), "", "fallback"))
	assert.Empty(t, detectorURL(context.Background(), nil, nil))
}

func TestDetectorTimeZoneValidator(t *testing.T) {
	t.Parallel()
	implementation := detectorTimeZoneValidator{}
	assert.Equal(t, implementation.Description(context.Background()), implementation.MarkdownDescription(context.Background()))
	for _, test := range []struct {
		value types.String
		error bool
	}{{types.StringNull(), false}, {types.StringUnknown(), false}, {types.StringValue("Australia/Adelaide"), false}, {types.StringValue("Not/AZone"), true}} {
		response := &validator.StringResponse{}
		implementation.ValidateString(context.Background(), validator.StringRequest{ConfigValue: test.value}, response)
		assert.Equal(t, test.error, response.Diagnostics.HasError())
	}
}

func TestResourceDetectorRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceDetector{}
	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(ctx, resource.SchemaRequest{}, schemaResponse)
	plan := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema}
	state := tfsdk.State{Raw: tftypes.NewValue(tftypes.Bool, true), Schema: schemaResponse.Schema}
	createResponse := &resource.CreateResponse{}
	implementation.Create(ctx, resource.CreateRequest{Plan: plan}, createResponse)
	assert.True(t, createResponse.Diagnostics.HasError())
	readResponse := &resource.ReadResponse{}
	implementation.Read(ctx, resource.ReadRequest{State: state}, readResponse)
	assert.True(t, readResponse.Diagnostics.HasError())
	updateResponse := &resource.UpdateResponse{}
	implementation.Update(ctx, resource.UpdateRequest{Plan: plan}, updateResponse)
	assert.True(t, updateResponse.Diagnostics.HasError())
	deleteResponse := &resource.DeleteResponse{}
	implementation.Delete(ctx, resource.DeleteRequest{State: state}, deleteResponse)
	assert.True(t, deleteResponse.Diagnostics.HasError())
}

func TestResourceDetectorMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := detector.Detector{}
	deleted := false
	validateCalls := 0
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		assert.NoError(t, json.NewEncoder(w).Encode(current))
	}
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload detector.ValidateDetectorRequestModel
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.NotEmpty(t, payload.Name)
			assert.NotEmpty(t, payload.ProgramText)
			assert.NotEmpty(t, payload.Rules)
			mu.Lock()
			validateCalls++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
		"POST /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload detector.CreateUpdateDetectorRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.Equal(t, "demo detector", payload.Name)
			assert.Equal(t, int32(30000), *payload.MaxDelay)
			assert.Equal(t, int64(3600000), *payload.VisualizationOptions.Time.Range)
			assert.Equal(t, []string{"provider-tag", "resource-tag"}, payload.Tags)
			assert.Equal(t, []string{"provider-team", "resource-team"}, payload.Teams)
			mu.Lock()
			current = detectorResponseFromRequest(payload)
			mu.Unlock()
			writeCurrent(w)
		}),
		"GET /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload detector.CreateUpdateDetectorRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			assert.Equal(t, "updated detector", payload.Name)
			assert.Equal(t, int64(1700000000000), *payload.VisualizationOptions.Time.Start)
			assert.Equal(t, int64(1700003600000), *payload.VisualizationOptions.Time.End)
			mu.Lock()
			current = detectorResponseFromRequest(payload)
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(
			t,
			endpoints,
			fwtest.WithMockResources(NewResourceDetector),
			fwtest.WithMockProviderMeta(configureDetectorProviderDefaults),
		),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/detector_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_detector.test", "id", "detector-id"),
				testresource.TestCheckResourceAttr("signalfx_detector.test", "max_delay", "30"),
				testresource.TestCheckResourceAttr("signalfx_detector.test", "rule.#", "1"),
				testresource.TestCheckResourceAttr("signalfx_detector.test", "label_resolutions.CPU", "1000"),
			)},
			{ConfigFile: config.StaticFile("testdata/detector_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_detector.test", "name", "updated detector"),
				testresource.TestCheckResourceAttr("signalfx_detector.test", "start_time", "1700000000"),
				testresource.TestCheckResourceAttr("signalfx_detector.test", "end_time", "1700003600"),
			)},
			{ConfigFile: config.StaticFile("testdata/detector_update.tf"), PlanOnly: true},
			{
				ResourceName: "signalfx_detector.test", ImportState: true, ImportStateId: "detector-id", ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"tags", "teams"},
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
	assert.Positive(t, validateCalls)
}

func TestResourceDetectorSchemaValidation(t *testing.T) {
	for name, cfg := range map[string]string{
		"missing rule": `resource "signalfx_detector" "test" {
  name         = "x"
  program_text = "x"
}`,
		"invalid timezone": minimalDetectorConfig(`timezone = "Not/AZone"`),
		"invalid severity": minimalDetectorConfig(`rule {
  severity     = "Severe"
  detect_label = "OTHER"
}`),
		"invalid notification": minimalDetectorConfig(`rule {
  severity      = "Critical"
  detect_label  = "OTHER"
  notifications = ["invalid"]
}`),
		"conflicting time": minimalDetectorConfig(`time_range = 60
start_time = 1`),
		"end without start": minimalDetectorConfig(`end_time = 2`),
		"invalid origin":    minimalDetectorConfig(`detector_origin = "AutoDetect"`),
		"missing parent":    minimalDetectorConfig(`detector_origin = "AutoDetectCustomization"`),
		"invalid delay":     minimalDetectorConfig(`max_delay = 901`),
		"invalid reminder": minimalDetectorConfig(`rule {
  severity     = "Critical"
  detect_label = "OTHER"
  reminder_notification {
    interval_ms = -1
    type        = "TIMEOUT"
  }
}`),
		"invalid color": minimalDetectorConfig(`viz_options {
  label = "CPU"
  color = "green"
}`),
		"invalid unit": minimalDetectorConfig(`viz_options {
  label      = "CPU"
  value_unit = "Meters"
}`),
	} {
		t.Run(name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceDetector)),
				Steps:                    []testresource.TestStep{{Config: cfg, ExpectError: regexp.MustCompile(`(?i)(invalid|must|require|conflict|between|at least)`)}},
			})
		})
	}
}

func TestResourceDetectorValidationAPIError(t *testing.T) {
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"invalid detector","details":["bad program"]}`))
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
		Steps:                    []testresource.TestStep{{Config: minimalDetectorConfig(""), ExpectError: regexp.MustCompile(`Invalid detector program or rules`)}},
	})
}

func TestResourceDetectorCreateErrors(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		error      *regexp.Regexp
	}{
		{name: "API error", body: "failed", status: http.StatusBadGateway, error: regexp.MustCompile(`status code 502`)},
		{name: "nil response", body: `null`, status: http.StatusOK, error: regexp.MustCompile(`no resource identifier`)},
		{name: "missing ID", body: `{}`, status: http.StatusOK, error: regexp.MustCompile(`no resource identifier`)},
		{name: "missing rules", body: `{"id":"detector-id"}`, status: http.StatusOK, error: regexp.MustCompile(`no alert rules`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoints := map[string]http.Handler{
				"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
				"POST /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(test.status)
					_, _ = w.Write([]byte(test.body))
				}),
			}
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
				Steps:                    []testresource.TestStep{{Config: minimalDetectorConfig(""), ExpectError: test.error}},
			})
		})
	}
}

func TestResourceDetectorReadError(t *testing.T) {
	current := detectorResponseFromRequest(minimalDetectorRequest())
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		"POST /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload detector.CreateUpdateDetectorRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			current = detectorResponseFromRequest(payload)
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"GET /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "failed", http.StatusBadGateway)
		}),
		"DELETE /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
		Steps:                    []testresource.TestStep{{Config: minimalDetectorConfig(""), ExpectError: regexp.MustCompile(`status code 502`)}},
	})
}

func TestResourceDetectorUpdateError(t *testing.T) {
	current := detectorResponseFromRequest(minimalDetectorRequest())
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		"POST /v2/detector":          http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { assert.NoError(t, json.NewEncoder(w).Encode(current)) }),
		"GET /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"PUT /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "failed", http.StatusBadGateway)
		}),
		"DELETE /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
		Steps: []testresource.TestStep{
			{Config: minimalDetectorConfig("")},
			{Config: minimalDetectorConfig(`description = "updated"`), ExpectError: regexp.MustCompile(`status code 502`)},
		},
	})
}

func TestResourceDetectorDeleteError(t *testing.T) {
	var mu sync.Mutex
	deleteCalls := 0
	current := detectorResponseFromRequest(minimalDetectorRequest())
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		"POST /v2/detector":          http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { assert.NoError(t, json.NewEncoder(w).Encode(current)) }),
		"GET /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"DELETE /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			deleteCalls++
			if deleteCalls == 1 {
				http.Error(w, "failed", http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
		Steps: []testresource.TestStep{
			{Config: minimalDetectorConfig("")},
			{Config: minimalDetectorConfig(""), Destroy: true, ExpectError: regexp.MustCompile(`status code 502`)},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, deleteCalls)
}

func TestResourceDetectorOverMTSWarning(t *testing.T) {
	current := detectorResponseFromRequest(minimalDetectorRequest())
	current.OverMTSLimit = true
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		"POST /v2/detector":          http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { assert.NoError(t, json.NewEncoder(w).Encode(current)) }),
		"GET /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"DELETE /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
		Steps:                    []testresource.TestStep{{Config: minimalDetectorConfig("")}},
	})
}

func TestResourceDetectorOriginAndParentRequireReplacement(t *testing.T) {
	current := detectorResponseFromRequest(minimalDetectorRequest())
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		"POST /v2/detector": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload detector.CreateUpdateDetectorRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload)) {
				return
			}
			current = detectorResponseFromRequest(payload)
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"GET /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, json.NewEncoder(w).Encode(current))
		}),
		"DELETE /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
		Steps: []testresource.TestStep{
			{Config: minimalDetectorConfig("")},
			{
				Config: minimalDetectorConfig(`detector_origin = "AutoDetectCustomization"
parent_detector_id = "parent-id"`),
				ConfigPlanChecks: testresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("signalfx_detector.test", plancheck.ResourceActionReplace),
				}},
			},
		},
	})
}

func TestResourceDetectorRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := detectorResponseFromRequest(minimalDetectorRequest())
	endpoints := map[string]http.Handler{
		"POST /v2/detector/validate": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
		"POST /v2/detector":          http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { assert.NoError(t, json.NewEncoder(w).Encode(current)) }),
		"GET /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				assert.NoError(t, json.NewEncoder(w).Encode(current))
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/detector/detector-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceDetector)),
		Steps: []testresource.TestStep{
			{Config: minimalDetectorConfig("")},
			{Config: minimalDetectorConfig(""), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func detectorTestModel(t *testing.T, absolute bool) resourceDetectorModel {
	t.Helper()
	ctx := context.Background()
	var diagnostics diag.Diagnostics
	notifications := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("Email,alerts@example.com")})
	states := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("AUTO_RESOLVED")})
	reminders := detectorListValue(ctx, detectorReminderAttributeTypes, []detectorReminderNotificationModel{{
		IntervalMS: types.Int64Value(60000), TimeoutMS: types.Int64Value(300000), Type: types.StringValue("TIMEOUT"),
	}}, &diagnostics)
	rules := detectorSetValue(ctx, detectorRuleAttributeTypes, []detectorRuleModel{{
		Severity: types.StringValue("Critical"), DetectLabel: types.StringValue("CPU"), Description: types.StringValue("CPU is high"),
		Notifications: notifications, Disabled: types.BoolValue(false), ParameterizedBody: types.StringValue("body"),
		ParameterizedSubject: types.StringValue("subject"), RunbookURL: types.StringValue("runbook"), Tip: types.StringValue("tip"),
		SkipClearNotificationStates: states, ReminderNotification: reminders,
	}}, &diagnostics)
	palette := detectorSetValue(ctx, detectorVisualizationAttributeTypes, []detectorVisualizationModel{{
		Label: types.StringValue("CPU"), Color: types.StringValue("blue"), DisplayName: types.StringValue("CPU usage"),
		ValueUnit: types.StringValue("Byte"), ValuePrefix: types.StringValue("pre"), ValueSuffix: types.StringValue("post"),
	}}, &diagnostics)
	require.False(t, diagnostics.HasError(), diagnostics)
	model := resourceDetectorModel{
		ID: types.StringValue("detector-id"), Name: types.StringValue("demo detector"), ProgramText: types.StringValue("program"),
		Description: types.StringValue("description"), Timezone: types.StringValue("UTC"), MaxDelay: types.Int64Value(30), MinDelay: types.Int64Value(15),
		ShowDataMarkers: types.BoolValue(true), ShowEventLines: types.BoolValue(true), DisableSampling: types.BoolValue(false),
		TimeRange: types.Int64Value(3600), StartTime: types.Int64Null(), EndTime: types.Int64Null(),
		Tags: stringSet("resource-tag"), Teams: stringSet("resource-team"), Rules: rules,
		AuthorizedWriterTeams: stringSet("writer-team"), AuthorizedWriterUsers: stringSet("writer-user"), VisualizationOptions: palette,
		LabelResolutions: types.MapUnknown(types.Int64Type), URL: types.StringUnknown(), DetectorOrigin: types.StringValue("Standard"), ParentDetectorID: types.StringValue(""),
	}
	if absolute {
		model.TimeRange = types.Int64Null()
		model.StartTime = types.Int64Value(1700000000)
		model.EndTime = types.Int64Value(1700003600)
	}
	return model
}

func detectorResponseFromRequest(payload detector.CreateUpdateDetectorRequest) detector.Detector {
	resolutions := map[string]any{"CPU": float64(1000)}
	return detector.Detector{
		Id: "detector-id", Name: payload.Name, Description: payload.Description, TimeZone: payload.TimeZone, MaxDelay: payload.MaxDelay, MinDelay: payload.MinDelay,
		ProgramText: payload.ProgramText, Rules: payload.Rules, AuthorizedWriters: payload.AuthorizedWriters, Tags: payload.Tags, Teams: payload.Teams,
		VisualizationOptions: payload.VisualizationOptions, ParentDetectorId: payload.ParentDetectorId, DetectorOrigin: payload.DetectorOrigin,
		LabelResolutions: &resolutions,
	}
}

func minimalDetectorRequest() detector.CreateUpdateDetectorRequest {
	zero := int32(0)
	rangeMilliseconds := int64(3600000)
	return detector.CreateUpdateDetectorRequest{
		Name: "minimal", ProgramText: "program", MaxDelay: &zero, MinDelay: &zero, DetectorOrigin: "Standard",
		Rules:                []*detector.Rule{{Severity: "Critical", DetectLabel: "CPU"}},
		AuthorizedWriters:    &detector.AuthorizedWriters{},
		VisualizationOptions: &detector.Visualization{ShowDataMarkers: true, Time: &detector.Time{Type: "relative", Range: &rangeMilliseconds}},
	}
}

func minimalDetectorConfig(extra string) string {
	return `resource "signalfx_detector" "test" {
  name         = "minimal"
  program_text = "program"
  rule {
    severity     = "Critical"
    detect_label = "CPU"
  }
` + extra + "\n}"
}

func stringSet(values ...string) types.Set {
	items := make([]attr.Value, len(values))
	for index, value := range values {
		items[index] = types.StringValue(value)
	}
	return types.SetValueMust(types.StringType, items)
}

func pointer[T any](value T) *T { return &value }

func configureDetectorProviderDefaults(meta *pmeta.Meta) {
	registry := feature.NewRegistry()
	registry.MustRegister(feature.PreviewProviderTags, feature.WithPreviewGlobalAvailable())
	registry.MustRegister(feature.PreviewProviderTeams, feature.WithPreviewGlobalAvailable())
	meta.Registry = registry
	meta.Tags = []string{"provider-tag"}
	meta.Teams = []string{"provider-team"}
}
