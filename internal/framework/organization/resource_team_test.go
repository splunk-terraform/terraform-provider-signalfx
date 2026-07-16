// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fworganization

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
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
	"github.com/signalfx/signalfx-go/notification"
	"github.com/signalfx/signalfx-go/team"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceTeamMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewResourceTeam()
	metadata := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_team", metadata.TypeName)
	model := resourceTeamModel{
		Members:               types.SetValueMust(types.StringType, nil),
		NotificationsCritical: notificationStrings(),
		NotificationsDefault:  notificationStrings(),
		NotificationsInfo:     notificationStrings(),
		NotificationsMajor:    notificationStrings(),
		NotificationsMinor:    notificationStrings(),
		NotificationsWarning:  notificationStrings(),
	}
	assert.NoError(t, fwtest.ResourceSchemaValidate(implementation, model))

	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(context.Background(), resource.SchemaRequest{}, schemaResponse)
	for _, name := range []string{
		"notifications_critical", "notifications_default", "notifications_info",
		"notifications_major", "notifications_minor", "notifications_warning",
	} {
		attribute := schemaResponse.Schema.Attributes[name].(schema.ListAttribute)
		assert.Len(t, attribute.Validators, 1)
	}
}

func TestResourceTeamModel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceTeamModel{
		ID:          types.StringValue("team-id"),
		Name:        types.StringValue("Primary Team"),
		Description: types.StringValue("Primary description"),
		Members: types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("member-a"), types.StringValue("member-b"),
		}),
		NotificationsCritical: notificationStrings("Email,critical@example.com"),
		NotificationsDefault:  notificationStrings("Webhook,,secret,https://hooks.example.com/default"),
		NotificationsInfo:     notificationStrings("Slack,slack-id,info-alerts"),
		NotificationsMajor:    notificationStrings("Team,major-team"),
		NotificationsMinor:    notificationStrings("PagerDuty,pagerduty-id"),
		NotificationsWarning:  notificationStrings("VictorOps,victor-id,warning"),
		URL:                   types.StringValue("https://app.signalfx.com/#team/team-id"),
	}

	payload, diagnostics := model.createUpdateRequest(ctx)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, "Primary Team", payload.Name)
	assert.ElementsMatch(t, []string{"member-a", "member-b"}, payload.Members)
	assert.Equal(t, "Email", payload.NotificationLists.Critical[0].Type)
	assert.Equal(t, "Webhook", payload.NotificationLists.Default[0].Type)
	assert.Equal(t, "Slack", payload.NotificationLists.Info[0].Type)
	assert.Equal(t, "Team", payload.NotificationLists.Major[0].Type)
	assert.Equal(t, "PagerDuty", payload.NotificationLists.Minor[0].Type)
	assert.Equal(t, "VictorOps", payload.NotificationLists.Warning[0].Type)

	details := &team.Team{
		Id: "updated-id", Name: "Updated Team", Description: "Updated description", Members: []string{"member-c"},
		NotificationLists: payload.NotificationLists,
	}
	diagnostics = model.updateFromAPI(ctx, details, false, "https://app.example/#team/team-id")
	require.False(t, diagnostics.HasError())
	assert.Equal(t, types.StringValue("team-id"), model.ID)
	assert.Equal(t, types.StringValue("Updated Team"), model.Name)
	assert.Equal(t, types.StringValue("https://app.example/#team/team-id"), model.URL)

	diagnostics = model.updateFromAPI(ctx, &team.Team{Id: "updated-id", Name: "Empty Team"}, true, "https://app.example/#team/updated-id")
	require.False(t, diagnostics.HasError())
	assert.Equal(t, types.StringValue("updated-id"), model.ID)
	assert.Empty(t, model.Members.Elements())
	assert.Empty(t, model.NotificationsCritical.Elements())
	assert.False(t, model.updateFromAPI(ctx, nil, false, "").HasError())
}

func TestResourceTeamModelErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceTeamModel{
		Members:               types.SetUnknown(types.StringType),
		NotificationsCritical: types.ListUnknown(types.StringType),
	}
	_, diagnostics := model.createUpdateRequest(ctx)
	assert.True(t, diagnostics.HasError())

	diagnostics = model.updateFromAPI(ctx, &team.Team{
		NotificationLists: team.NotificationLists{Default: []*notification.Notification{{Type: "Unknown", Value: struct{}{}}}},
	}, false, "")
	assert.True(t, diagnostics.HasError())
}

func TestResourceTeamRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceTeam{}
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

func TestResourceTeamMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := team.Team{}
	deleted := false

	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := json.NewEncoder(w).Encode(current); err != nil {
			t.Errorf("write team response: %v", err)
		}
	}
	updateCurrent := func(payload team.CreateUpdateTeamRequest) {
		mu.Lock()
		defer mu.Unlock()
		current = team.Team{
			Id: "team-id", Name: payload.Name, Description: payload.Description,
			Members: payload.Members, NotificationLists: payload.NotificationLists,
		}
	}

	endpoints := map[string]http.Handler{
		"POST /v2/team": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeTeamPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Primary Team", payload.Name)
			assert.ElementsMatch(t, []string{"member-a", "member-b"}, payload.Members)
			assert.Equal(t, "Email", payload.NotificationLists.Critical[0].Type)
			assert.Equal(t, "Webhook", payload.NotificationLists.Default[0].Type)
			assert.Equal(t, "Slack", payload.NotificationLists.Info[0].Type)
			assert.Equal(t, "Team", payload.NotificationLists.Major[0].Type)
			assert.Equal(t, "PagerDuty", payload.NotificationLists.Minor[0].Type)
			assert.Equal(t, "VictorOps", payload.NotificationLists.Warning[0].Type)
			updateCurrent(payload)
			writeCurrent(w)
		}),
		"GET /v2/team/team-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/team/team-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeTeamPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated Team", payload.Name)
			assert.Equal(t, []string{"member-c"}, payload.Members)
			assert.Equal(t, "Email", payload.NotificationLists.Warning[0].Type)
			updateCurrent(payload)
			writeCurrent(w)
		}),
		"DELETE /v2/team/team-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceTeam)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/team_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_team.test", "id", "team-id"),
				testresource.TestCheckResourceAttr("signalfx_team.test", "name", "Primary Team"),
				testresource.TestCheckResourceAttr("signalfx_team.test", "members.#", "2"),
				testresource.TestCheckResourceAttr("signalfx_team.test", "notifications_critical.0", "Email,critical@example.com"),
				testresource.TestCheckResourceAttrWith("signalfx_team.test", "url", func(value string) error {
					if !strings.HasSuffix(value, "#/team/team-id") {
						return fmt.Errorf("expected team application URL, got %q", value)
					}
					return nil
				}),
			)},
			{ConfigFile: config.StaticFile("testdata/team_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_team.test", "name", "Updated Team"),
				testresource.TestCheckResourceAttr("signalfx_team.test", "members.#", "1"),
				testresource.TestCheckResourceAttr("signalfx_team.test", "notifications_warning.0", "Email,warning@example.com"),
			)},
			{ConfigFile: config.StaticFile("testdata/team_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_team.test", ImportState: true, ImportStateId: "team-id", ImportStateVerify: true},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceTeamRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := team.Team{Id: "team-id", Name: "Primary Team"}
	endpoints := map[string]http.Handler{
		"POST /v2/team": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(current) }),
		"GET /v2/team/team-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				_ = json.NewEncoder(w).Encode(current)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/team/team-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceTeam)),
		Steps: []testresource.TestStep{
			{Config: `resource "signalfx_team" "test" { name = "Primary Team" }`},
			{Config: `resource "signalfx_team" "test" { name = "Primary Team" }`, PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceTeamErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		config    string
		endpoints map[string]http.Handler
		error     *regexp.Regexp
	}{
		{
			name: "API error", config: `resource "signalfx_team" "test" { name = "Failed Team" }`,
			endpoints: map[string]http.Handler{"POST /v2/team": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "failed", http.StatusBadGateway)
			})},
			error: regexp.MustCompile(`status code 502`),
		},
		{
			name: "missing API identifier", config: `resource "signalfx_team" "test" { name = "Empty Team" }`,
			endpoints: map[string]http.Handler{"POST /v2/team": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			})},
			error: regexp.MustCompile(`returned no resource identifier`),
		},
		{name: "missing name", config: `resource "signalfx_team" "test" {}`, error: regexp.MustCompile(`argument "name" is required`)},
		{
			name: "invalid notification", config: `resource "signalfx_team" "test" {
  name                  = "Invalid"
  notifications_default = ["invalid"]
}`,
			error: regexp.MustCompile(`Invalid notification destination`),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, test.endpoints, fwtest.WithMockResources(NewResourceTeam)),
				Steps:                    []testresource.TestStep{{Config: test.config, ExpectError: test.error}},
			})
		})
	}
}

func notificationStrings(values ...string) types.List {
	elements := make([]attr.Value, len(values))
	for index, value := range values {
		elements[index] = types.StringValue(value)
	}
	return types.ListValueMust(types.StringType, elements)
}

func decodeTeamPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (team.CreateUpdateTeamRequest, bool) {
	t.Helper()
	var payload team.CreateUpdateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode team payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}
