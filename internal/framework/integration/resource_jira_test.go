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

func TestResourceJiraMetadataAndSchema(t *testing.T) {
	t.Parallel()
	r := NewResourceJira()
	metadata := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_jira_integration", metadata.TypeName)
	assert.NoError(t, fwtest.ResourceSchemaValidate(r, resourceJiraModel{}))

	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	for _, name := range []string{"api_token", "user_email", "username", "password"} {
		attribute, ok := resp.Schema.Attributes[name].(schema.StringAttribute)
		require.True(t, ok)
		assert.True(t, attribute.IsOptional(), name)
		assert.Len(t, attribute.Validators, 1, name+" must preserve legacy credential conflicts")
	}
	assert.True(t, resp.Schema.Attributes["api_token"].IsSensitive())
	assert.True(t, resp.Schema.Attributes["password"].IsSensitive())
	assert.False(t, resp.Schema.Attributes["username"].IsSensitive())
	authMethod := resp.Schema.Attributes["auth_method"].(schema.StringAttribute)
	assert.True(t, authMethod.IsRequired())
	assert.Len(t, authMethod.Validators, 1)
	assert.True(t, resp.Schema.Attributes["assignee_display_name"].IsOptional())
}

func TestResourceJiraModel(t *testing.T) {
	t.Parallel()
	model := resourceJiraModel{
		integrationModel: integrationModel{
			ID: types.StringValue("jira-id"), Name: types.StringValue("Jira"), Enabled: types.BoolValue(true),
		},
		Username: types.StringValue("primary-user"), Password: types.StringValue("primary-password"),
		AuthMethod: types.StringValue(jiraAuthUsernamePassword), BaseURL: types.StringValue("https://primary.atlassian.test"),
		IssueType: types.StringValue("Story"), ProjectKey: types.StringValue("PRIMARY"),
		AssigneeName: types.StringValue("primary-assignee"), AssigneeDisplayName: types.StringValue("Primary Assignee"),
		UserEmail: types.StringValue("ignored@example.test"), APIToken: types.StringValue("ignored-token"),
	}
	assert.Equal(t, &integration.JiraIntegration{
		Type: integration.Type("Jira"), Name: "Jira", Enabled: true, AuthMethod: jiraAuthUsernamePassword,
		Username: "primary-user", Password: "primary-password", BaseURL: "https://primary.atlassian.test",
		IssueType: "Story", ProjectKey: "PRIMARY",
		Assignee: &integration.JiraAssignee{Name: "primary-assignee", DisplayName: "Primary Assignee"},
	}, model.jiraIntegration())

	model.updateFromAPI(nil, true)
	model.updateFromAPI(&integration.JiraIntegration{
		Id: "ignored", Name: "Read", Enabled: false, AuthMethod: jiraAuthUsernamePassword,
		BaseURL: "https://read.atlassian.test", IssueType: "Task", ProjectKey: "READ",
		Assignee: &integration.JiraAssignee{Name: "read-assignee"},
	}, false)
	assert.Equal(t, types.StringValue("jira-id"), model.ID)
	assert.Equal(t, types.StringValue("primary-user"), model.Username, "API-omitted username must survive refresh")
	assert.Equal(t, types.StringValue("primary-password"), model.Password, "API-omitted password must survive refresh")
	assert.True(t, model.UserEmail.IsNull())
	assert.True(t, model.APIToken.IsNull())
	assert.Equal(t, types.StringValue("Primary Assignee"), model.AssigneeDisplayName, "omitted display name must survive refresh")

	model.AuthMethod = types.StringValue(jiraAuthEmailToken)
	model.UserEmail = types.StringValue("updated@example.test")
	model.APIToken = types.StringValue("updated-token")
	emailPayload := model.jiraIntegration()
	assert.Empty(t, emailPayload.Username)
	assert.Empty(t, emailPayload.Password)
	assert.Equal(t, "updated@example.test", emailPayload.UserEmail)
	assert.Equal(t, "updated-token", emailPayload.APIToken)

	model.updateFromAPI(&integration.JiraIntegration{
		Id: "updated", Name: "Updated", Enabled: true, AuthMethod: jiraAuthEmailToken,
		UserEmail: "api@example.test", BaseURL: "https://updated.atlassian.test", IssueType: "Problem", ProjectKey: "UPDATED",
	}, true)
	assert.Equal(t, types.StringValue("updated"), model.ID)
	assert.Equal(t, types.StringValue("api@example.test"), model.UserEmail)
	assert.Equal(t, types.StringValue("updated-token"), model.APIToken, "API-omitted token must survive refresh")
	assert.True(t, model.Username.IsNull())
	assert.True(t, model.Password.IsNull())
	assert.Equal(t, types.StringValue("read-assignee"), model.AssigneeName, "omitted assignee must preserve state")

	model.updateFromAPI(&integration.JiraIntegration{
		Id: "final", Name: "Final", Enabled: true, AuthMethod: jiraAuthEmailToken,
		UserEmail: "final@example.test", APIToken: "api-token", BaseURL: "https://final.atlassian.test",
		IssueType: "Bug", ProjectKey: "FINAL", Assignee: &integration.JiraAssignee{Name: "final-assignee", DisplayName: "Final Assignee"},
	}, true)
	assert.Equal(t, types.StringValue("api-token"), model.APIToken)
	assert.Equal(t, types.StringValue("final-assignee"), model.AssigneeName)
	assert.Equal(t, types.StringValue("Final Assignee"), model.AssigneeDisplayName)
}

func TestResourceJiraRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceJira{}
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

func TestResourceJiraMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := integration.JiraIntegration{
		Id: "jira-id", Name: "Primary Jira", Enabled: true, Type: integration.Type("Jira"),
		AuthMethod: jiraAuthUsernamePassword, Username: "primary-user", BaseURL: "https://primary.atlassian.test",
		IssueType: "Story", ProjectKey: "PRIMARY",
		Assignee: &integration.JiraAssignee{Name: "primary-assignee", DisplayName: "Primary Assignee"},
	}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		if err := writeJiraResponse(w, current); err != nil {
			t.Errorf("write Jira response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeJiraPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, integration.Type("Jira"), payload.Type)
			assert.Equal(t, jiraAuthUsernamePassword, payload.AuthMethod)
			assert.Equal(t, "primary-user", payload.Username)
			assert.Equal(t, "primary-password", payload.Password)
			assert.Empty(t, payload.APIToken)
			writeCurrent(w)
		}),
		"GET /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeJiraPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "Updated Jira", payload.Name)
			assert.False(t, payload.Enabled)
			assert.Equal(t, jiraAuthEmailToken, payload.AuthMethod)
			assert.Equal(t, "updated@example.test", payload.UserEmail)
			assert.Equal(t, "updated-token", payload.APIToken)
			assert.Empty(t, payload.Password)
			mu.Lock()
			current.Name, current.Enabled, current.AuthMethod = payload.Name, payload.Enabled, payload.AuthMethod
			current.Username, current.UserEmail = "", payload.UserEmail
			current.BaseURL, current.IssueType, current.ProjectKey = payload.BaseURL, payload.IssueType, payload.ProjectKey
			current.Assignee = payload.Assignee
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceJira)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/jira_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_jira_integration.test", "id", "jira-id"),
				testresource.TestCheckResourceAttr("signalfx_jira_integration.test", "password", "primary-password"),
			)},
			{ConfigFile: config.StaticFile("testdata/jira_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_jira_integration.test", "name", "Updated Jira"),
				testresource.TestCheckResourceAttr("signalfx_jira_integration.test", "auth_method", jiraAuthEmailToken),
				testresource.TestCheckResourceAttr("signalfx_jira_integration.test", "api_token", "updated-token"),
				testresource.TestCheckNoResourceAttr("signalfx_jira_integration.test", "username"),
			)},
			{ConfigFile: config.StaticFile("testdata/jira_update.tf"), PlanOnly: true},
			{ResourceName: "signalfx_jira_integration.test", ImportState: true, ImportStateId: "jira-id", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"password", "api_token"}},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceJiraValidation(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceJira)),
		Steps: []testresource.TestStep{{
			ConfigFile:  config.StaticFile("testdata/jira_invalid.tf"),
			ExpectError: regexp.MustCompile(`(?s)(OAuth|conflict)`),
		}},
	})
}

func TestResourceJiraRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := integration.JiraIntegration{
		Id: "jira-id", Name: "Primary Jira", Enabled: true, Type: integration.Type("Jira"),
		AuthMethod: jiraAuthUsernamePassword, Username: "primary-user", BaseURL: "https://primary.atlassian.test",
		IssueType: "Story", ProjectKey: "PRIMARY", Assignee: &integration.JiraAssignee{Name: "primary-assignee"},
	}
	writeCurrent := func(w http.ResponseWriter) {
		if err := writeJiraResponse(w, current); err != nil {
			t.Errorf("write Jira response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"GET /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				writeCurrent(w)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceJira)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/jira_create.tf")},
			{ConfigFile: config.StaticFile("testdata/jira_create.tf"), PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceJiraErrorHandling(t *testing.T) {
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
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceJira)),
				Steps:                    []testresource.TestStep{{ConfigFile: config.StaticFile("testdata/jira_create.tf"), ExpectError: test.error}},
			})
		})
	}
}

func TestResourceJiraUpdateAdminTokenGuidance(t *testing.T) {
	current := integration.JiraIntegration{
		Id: "jira-id", Name: "Primary Jira", Enabled: true, Type: integration.Type("Jira"),
		AuthMethod: jiraAuthUsernamePassword, Username: "primary-user", BaseURL: "https://primary.atlassian.test",
		IssueType: "Story", ProjectKey: "PRIMARY", Assignee: &integration.JiraAssignee{Name: "primary-assignee"},
	}
	writeCurrent := func(w http.ResponseWriter) {
		if err := writeJiraResponse(w, current); err != nil {
			t.Errorf("write Jira response: %v", err)
		}
	}
	endpoints := map[string]http.Handler{
		"POST /v2/integration":        http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"GET /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}),
		"DELETE /v2/integration/jira-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceJira)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/jira_create.tf")},
			{ConfigFile: config.StaticFile("testdata/jira_update.tf"), ExpectError: regexp.MustCompile(adminTokenHelp)},
		},
	})
}

func decodeJiraPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (integration.JiraIntegration, bool) {
	t.Helper()
	var payload integration.JiraIntegration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode Jira payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}

func writeJiraResponse(w http.ResponseWriter, details integration.JiraIntegration) error {
	return json.NewEncoder(w).Encode(map[string]any{
		"id": details.Id, "name": details.Name, "enabled": details.Enabled, "type": details.Type,
		"authMethod": details.AuthMethod, "username": details.Username, "userEmail": details.UserEmail,
		"baseUrl": details.BaseURL, "issueType": details.IssueType, "projectKey": details.ProjectKey,
		"assignee": details.Assignee,
	})
}
