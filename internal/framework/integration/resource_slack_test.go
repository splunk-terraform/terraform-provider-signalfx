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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceSlackMetadataAndSchema(t *testing.T) {
	t.Parallel()
	r := NewResourceSlack()
	metadata := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_slack_integration", metadata.TypeName)
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, resourceSlackModel{}))
}

func TestResourceSlackModel(t *testing.T) {
	t.Parallel()
	model := resourceSlackModel{
		integrationModel: integrationModel{
			ID: types.StringValue("slack-id"), Name: types.StringValue("Slack"), Enabled: types.BoolValue(true),
		},
		WebhookURL: types.StringValue("https://hooks.slack.test/secret"),
	}
	assert.Equal(t, &integration.SlackIntegration{
		Type: "Slack", Name: "Slack", Enabled: true, WebhookUrl: "https://hooks.slack.test/secret",
	}, model.slackIntegration())
	model.updateFromAPI(nil, true)
	model.updateFromAPI(&integration.SlackIntegration{Id: "ignored", Name: "Read", Enabled: false}, false)
	assert.Equal(t, types.StringValue("slack-id"), model.ID)
	assert.Equal(t, types.StringValue("https://hooks.slack.test/secret"), model.WebhookURL)
	model.updateFromAPI(&integration.SlackIntegration{Id: "updated", Name: "Updated", Enabled: true}, true)
	assert.Equal(t, types.StringValue("updated"), model.ID)
}

func TestResourceSlackRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceSlack{}
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

func TestResourceSlackMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := integration.SlackIntegration{Id: "slack-id", Name: "Primary Slack", Enabled: true, Type: "Slack"}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writeSlackResponse(w, current); err != nil {
			t.Errorf("write Slack response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeSlackPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.Type("Slack"), payload.Type)
			assert.Equal(t, "https://hooks.slack.test/primary", payload.WebhookUrl)
			writeCurrent(w)
		}),
		"GET /v2/integration/slack-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/slack-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeSlackPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated Slack", payload.Name)
			assert.Equal(t, "https://hooks.slack.test/updated", payload.WebhookUrl)
			mu.Lock()
			current.Name = payload.Name
			current.Enabled = payload.Enabled
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/slack-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceSlack)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/slack_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_slack_integration.test", "id", "slack-id"),
				testresource.TestCheckResourceAttr("signalfx_slack_integration.test", "webhook_url", "https://hooks.slack.test/primary"),
			)},
			{ConfigFile: config.StaticFile("testdata/slack_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_slack_integration.test", "name", "Updated Slack"),
				testresource.TestCheckResourceAttr("signalfx_slack_integration.test", "enabled", "false"),
				testresource.TestCheckResourceAttr("signalfx_slack_integration.test", "webhook_url", "https://hooks.slack.test/updated"),
			)},
			{ConfigFile: config.StaticFile("testdata/slack_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_slack_integration.test", ImportState: true, ImportStateId: "slack-id", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"webhook_url"}},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceSlackErrorHandling(t *testing.T) {
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
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceSlack)),
				Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/slack_create.tf"), ExpectError: test.error}},
			})
		})
	}
}

func decodeSlackPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.SlackIntegration, bool) {
	t.Helper()
	var payload integration.SlackIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode Slack payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}

func writeSlackResponse(w http.ResponseWriter, details integration.SlackIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{"id": details.Id, "name": details.Name, "enabled": details.Enabled, "type": details.Type})
}
