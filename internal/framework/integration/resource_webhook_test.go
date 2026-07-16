// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceWebhookMetadataAndSchema(t *testing.T) {
	t.Parallel()
	r := NewResourceWebhook()
	metadata := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_webhook_integration", metadata.TypeName)

	modelWithoutLegacyBlock := struct {
		integrationModel
		URL             types.String `tfsdk:"url"`
		SharedSecret    types.String `tfsdk:"shared_secret"`
		Method          types.String `tfsdk:"method"`
		PayloadTemplate types.String `tfsdk:"payload_template"`
	}{}
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, modelWithoutLegacyBlock))

	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	assert.True(t, resp.Schema.Attributes["url"].IsOptional(), "legacy url is optional")
	assert.True(t, resp.Schema.Attributes["shared_secret"].IsSensitive())
	headerBlock, ok := resp.Schema.Blocks["headers"].(schema.SetNestedBlock)
	require.True(t, ok, "legacy headers must remain set block syntax")
	assert.NotEmpty(t, headerBlock.Description)
	assert.True(t, headerBlock.NestedObject.Attributes["header_key"].IsSensitive())
	assert.True(t, headerBlock.NestedObject.Attributes["header_value"].IsSensitive())
	assert.True(t, headerBlock.Type().Equal(types.SetType{ElemType: webhookHeaderObjectType}))
}

func TestResourceWebhookModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceWebhookModel{
		integrationModel: integrationModel{ID: types.StringValue("webhook-id"), Name: types.StringValue("Webhook"), Enabled: types.BoolValue(true)},
		URL:              types.StringValue("https://webhook.test/primary"),
		SharedSecret:     types.StringValue("primary-secret"),
		Headers:          webhookHeaderSet(t, map[string]string{"z-header": "z-value", "a-header": "a-value"}),
		Method:           types.StringValue("POST"),
		PayloadTemplate:  types.StringValue(`{"message":"{{{message}}}"}`),
	}
	payload, diags := model.webhookIntegration(ctx)
	require.False(t, diags.HasError())
	assert.Equal(t, &integration.WebhookIntegration{
		Type: integration.WEBHOOK, Name: "Webhook", Enabled: true, Url: "https://webhook.test/primary",
		SharedSecret: "primary-secret", Headers: map[string]any{"a-header": "a-value", "z-header": "z-value"},
		Method: "POST", PayloadTemplate: `{"message":"{{{message}}}"}`,
	}, payload)

	assert.False(t, model.updateFromAPI(nil, true).HasError())
	assert.False(t, model.updateFromAPI(&integration.WebhookIntegration{
		Id: "ignored", Name: "Read", Enabled: false,
	}, false).HasError())
	assert.Equal(t, types.StringValue("webhook-id"), model.ID)
	assert.Equal(t, types.StringValue("primary-secret"), model.SharedSecret, "omitted secrets must survive refresh")
	assert.Equal(t, types.StringValue("https://webhook.test/primary"), model.URL, "omitted optional values must preserve null/known state")

	diags = model.updateFromAPI(&integration.WebhookIntegration{
		Id: "updated", Name: "Updated", Enabled: true, Url: "https://webhook.test/updated",
		Headers: map[string]any{"new-header": "new-value"}, Method: "PUT", PayloadTemplate: `{"updated":true}`,
	}, true)
	require.False(t, diags.HasError())
	assert.Equal(t, types.StringValue("updated"), model.ID)
	assert.Equal(t, types.StringValue("https://webhook.test/updated"), model.URL)
	var headers []webhookHeaderModel
	require.False(t, model.Headers.ElementsAs(ctx, &headers, false).HasError())
	require.Len(t, headers, 1)
	assert.Equal(t, "new-header", headers[0].Key.ValueString())
	assert.Equal(t, "new-value", headers[0].Value.ValueString())

	diags = model.updateFromAPI(&integration.WebhookIntegration{Headers: map[string]any{"bad": 42}}, true)
	assert.True(t, diags.HasError())
}

func TestResourceWebhookNullHeaders(t *testing.T) {
	t.Parallel()
	model := resourceWebhookModel{integrationModel: integrationModel{Name: types.StringValue("Webhook"), Enabled: types.BoolValue(true)}, Headers: types.SetNull(webhookHeaderObjectType)}
	payload, diags := model.webhookIntegration(context.Background())
	assert.False(t, diags.HasError())
	assert.Nil(t, payload.Headers)
}

func TestResourceWebhookRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceWebhook{}
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

func TestResourceWebhookMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := integration.WebhookIntegration{
		Id: "webhook-id", Name: "Primary Webhook", Enabled: true, Type: integration.WEBHOOK,
		Url: "https://webhook.test/primary", Headers: map[string]any{"x-primary": "primary-value"},
		Method: "POST", PayloadTemplate: `{"primary":true}`,
	}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("write Webhook response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeWebhookPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.WEBHOOK, payload.Type)
			assert.Equal(t, "primary-secret", payload.SharedSecret)
			assert.Equal(t, map[string]any{"x-primary": "primary-value"}, payload.Headers)
			writeCurrent(w)
		}),
		"GET /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeWebhookPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated Webhook", payload.Name)
			assert.Equal(t, "updated-secret", payload.SharedSecret)
			assert.Equal(t, map[string]any{"x-updated": "updated-value"}, payload.Headers)
			mu.Lock()
			current.Name, current.Enabled, current.Url = payload.Name, payload.Enabled, payload.Url
			current.Headers, current.Method, current.PayloadTemplate = payload.Headers, payload.Method, payload.PayloadTemplate
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceWebhook)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/webhook_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_webhook_integration.test", "id", "webhook-id"),
				testresource.TestCheckResourceAttr("signalfx_webhook_integration.test", "shared_secret", "primary-secret"),
				testresource.TestCheckResourceAttr("signalfx_webhook_integration.test", "headers.#", "1"),
			)},
			{ConfigFile: config.StaticFile("testdata/webhook_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_webhook_integration.test", "name", "Updated Webhook"),
				testresource.TestCheckResourceAttr("signalfx_webhook_integration.test", "enabled", "false"),
				testresource.TestCheckResourceAttr("signalfx_webhook_integration.test", "shared_secret", "updated-secret"),
				testresource.TestCheckResourceAttr("signalfx_webhook_integration.test", "headers.#", "1"),
			)},
			{ConfigFile: config.StaticFile("testdata/webhook_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_webhook_integration.test", ImportState: true, ImportStateId: "webhook-id", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"shared_secret"}},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceWebhookRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := integration.WebhookIntegration{
		Id: "webhook-id", Name: "Primary Webhook", Enabled: true, Type: integration.WEBHOOK,
		Url: "https://webhook.test/primary", Headers: map[string]any{"x-primary": "primary-value"},
		Method: "POST", PayloadTemplate: `{"primary":true}`,
	}
	writeCurrent := func(w http.ResponseWriter) {
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("write Webhook response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"GET /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				writeCurrent(w)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceWebhook)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/webhook_create.tf")},
			{ConfigFile: config.StaticFile("testdata/webhook_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceWebhookErrorHandling(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		error  *regexp.Regexp
	}{
		{name: "administrator guidance", status: http.StatusUnauthorized, error: regexp.MustCompile(adminTokenHelp)},
		{name: "server failure", status: http.StatusBadGateway, error: regexp.MustCompile(`status code 502`)},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoints := map[string]http.Handler{"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "failed", test.status) })}
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceWebhook)),
				Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/webhook_create.tf"), ExpectError: test.error}},
			})
		})
	}
}

func TestResourceWebhookUpdateAdminTokenGuidance(t *testing.T) {
	current := integration.WebhookIntegration{
		Id: "webhook-id", Name: "Primary Webhook", Enabled: true, Type: integration.WEBHOOK,
		Url: "https://webhook.test/primary", Headers: map[string]any{"x-primary": "primary-value"},
		Method: "POST", PayloadTemplate: `{"primary":true}`,
	}
	writeCurrent := func(w http.ResponseWriter) {
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("write Webhook response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration":           http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"GET /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "unauthorized", http.StatusUnauthorized) }),
		"DELETE /v2/integration/webhook-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceWebhook)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/webhook_create.tf")},
			{ConfigFile: config.StaticFile("testdata/webhook_update.tf"), ExpectError: regexp.MustCompile(adminTokenHelp)},
		},
	})
}

func webhookHeaderSet(t *testing.T, headers map[string]string) types.Set {
	t.Helper()
	elements := make([]attr.Value, 0, len(headers))
	for key, value := range headers {
		object, diags := types.ObjectValue(webhookHeaderAttributeTypes, map[string]attr.Value{
			"header_key": types.StringValue(key), "header_value": types.StringValue(value),
		})
		require.False(t, diags.HasError())
		elements = append(elements, object)
	}
	set, diags := types.SetValue(webhookHeaderObjectType, elements)
	require.False(t, diags.HasError())
	return set
}

func decodeWebhookPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.WebhookIntegration, bool) {
	t.Helper()
	var payload integration.WebhookIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode Webhook payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}
