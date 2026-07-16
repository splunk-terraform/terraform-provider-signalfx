// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fworganization

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
	"github.com/signalfx/signalfx-go/notification"
	"github.com/signalfx/signalfx-go/orgtoken"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwtest"
)

func TestResourceOrgTokenMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewResourceOrgToken()
	metadata := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_org_token", metadata.TypeName)

	model := struct {
		ID            types.String `tfsdk:"id"`
		Name          types.String `tfsdk:"name"`
		Description   types.String `tfsdk:"description"`
		AuthScopes    types.List   `tfsdk:"auth_scopes"`
		Disabled      types.Bool   `tfsdk:"disabled"`
		Notifications types.List   `tfsdk:"notifications"`
		Secret        types.String `tfsdk:"secret"`
		ExpiresAt     types.Int64  `tfsdk:"expires_at"`
	}{AuthScopes: types.ListNull(types.StringType), Notifications: types.ListNull(types.StringType)}
	assert.NoError(t, fwtest.ResourceSchemaValidate(implementation, model))

	schemaResponse := &resource.SchemaResponse{}
	implementation.Schema(context.Background(), resource.SchemaRequest{}, schemaResponse)
	hostBlock := schemaResponse.Schema.Blocks["host_or_usage_limits"].(schema.SingleNestedBlock)
	dpmBlock := schemaResponse.Schema.Blocks["dpm_limits"].(schema.SingleNestedBlock)
	assert.Len(t, hostBlock.Validators, 1)
	assert.Len(t, hostBlock.Attributes, 8)
	assert.Len(t, dpmBlock.Validators, 1)
	assert.Len(t, dpmBlock.Attributes, 2)
	assert.Len(t, schemaResponse.Schema.Attributes["notifications"].(schema.ListAttribute).Validators, 1)
	assert.True(t, schemaResponse.Schema.Attributes["secret"].IsSensitive())
}

func TestResourceOrgTokenModelHostLimits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceOrgTokenModel{
		ID:            types.StringValue("existing-token"),
		Name:          types.StringValue("Primary Token"),
		Description:   types.StringValue("Primary description"),
		AuthScopes:    stringList("INGEST", "API"),
		Disabled:      types.BoolValue(false),
		Notifications: notificationStrings("Email,alerts@example.com"),
		Secret:        types.StringValue("existing-secret"),
		HostOrUsageLimits: &orgTokenHostOrUsageLimitsModel{
			HostLimit:                           types.Int64Value(100),
			HostNotificationThreshold:           types.Int64Value(90),
			ContainerLimit:                      types.Int64Value(-1),
			ContainerNotificationThreshold:      types.Int64Value(-1),
			CustomMetricsLimit:                  types.Int64Value(2000),
			CustomMetricsNotificationThreshold:  types.Int64Value(1800),
			HighResMetricsLimit:                 types.Int64Value(-1),
			HighResMetricsNotificationThreshold: types.Int64Value(-1),
		},
	}

	payload, diagnostics := model.createUpdateRequest(ctx)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, "Primary Token", payload.Name)
	assert.Equal(t, []string{"INGEST", "API"}, payload.AuthScopes)
	assert.Equal(t, int64(100), *payload.Limits.CategoryQuota.HostThreshold)
	assert.Nil(t, payload.Limits.CategoryQuota.ContainerThreshold)
	assert.Equal(t, int64(1800), *payload.Limits.CategoryNotificationThreshold.CustomMetricThreshold)
	require.Len(t, payload.Notifications, 1)
	assert.Equal(t, "Email", payload.Notifications[0].Type)

	apiName := "api-normalized-token"
	diagnostics = model.updateFromAPI(ctx, &orgtoken.Token{
		Name: apiName, AuthScopes: []string{"RUM"}, Description: "API description",
		Secret: "", Disabled: true, Expiry: 12345,
	}, true, true)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, types.StringValue(apiName), model.ID)
	assert.Equal(t, types.StringValue("Primary Token"), model.Name)
	assert.Equal(t, []string{"INGEST", "API"}, listStrings(t, model.AuthScopes))
	assert.Equal(t, types.StringValue("existing-secret"), model.Secret)
	assert.Equal(t, types.Int64Value(12345), model.ExpiresAt)
	assert.NotNil(t, model.HostOrUsageLimits)
}

func TestResourceOrgTokenModelDPMLimitsAndRefresh(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceOrgTokenModel{
		Name:          types.StringNull(),
		AuthScopes:    types.ListNull(types.StringType),
		Disabled:      types.BoolNull(),
		Notifications: types.ListNull(types.StringType),
		DPMLimits: &orgTokenDPMLimitsModel{
			DPMLimit: types.Int32Value(5000), DPMNotificationThreshold: types.Int32Value(-1),
		},
	}
	payload, diagnostics := model.createUpdateRequest(ctx)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, int32(5000), *payload.Limits.DpmQuota)
	assert.Nil(t, payload.Limits.DpmNotificationThreshold)

	quota := int32(7000)
	threshold := int32(6500)
	diagnostics = model.updateFromAPI(ctx, &orgtoken.Token{
		Name: "imported-token", AuthScopes: []string{"RUM", "API"}, Disabled: true,
		Limits: &orgtoken.Limit{DpmQuota: &quota, DpmNotificationThreshold: &threshold},
		Expiry: 98765,
	}, false, false)
	require.False(t, diagnostics.HasError())
	assert.Equal(t, types.StringValue("imported-token"), model.Name)
	assert.Equal(t, []string{"API", "RUM"}, listStrings(t, model.AuthScopes))
	assert.Nil(t, model.HostOrUsageLimits)
	require.NotNil(t, model.DPMLimits)
	assert.Equal(t, int32(7000), model.DPMLimits.DPMLimit.ValueInt32())
	assert.Equal(t, int32(6500), model.DPMLimits.DPMNotificationThreshold.ValueInt32())
	assert.Equal(t, types.Int64Value(98765), model.ExpiresAt)

	hostLimit := int64(25)
	diagnostics = model.updateFromAPI(ctx, &orgtoken.Token{Limits: &orgtoken.Limit{
		CategoryQuota: &orgtoken.UsageLimits{HostThreshold: &hostLimit},
	}}, false, false)
	require.False(t, diagnostics.HasError())
	assert.Nil(t, model.DPMLimits)
	require.NotNil(t, model.HostOrUsageLimits)
	assert.Equal(t, int64(25), model.HostOrUsageLimits.HostLimit.ValueInt64())
	assert.Equal(t, int64(-1), model.HostOrUsageLimits.ContainerLimit.ValueInt64())

	model.updateLimitsFromAPI(nil)
	assert.Nil(t, model.DPMLimits)
	assert.Nil(t, model.HostOrUsageLimits)
	assert.False(t, model.updateFromAPI(ctx, nil, false, false).HasError())
}

func TestResourceOrgTokenModelErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceOrgTokenModel{
		AuthScopes:    types.ListUnknown(types.StringType),
		Notifications: types.ListUnknown(types.StringType),
		HostOrUsageLimits: &orgTokenHostOrUsageLimitsModel{
			HostLimit: types.Int64Unknown(),
		},
	}
	_, diagnostics := model.createUpdateRequest(ctx)
	assert.True(t, diagnostics.HasError())

	model = resourceOrgTokenModel{DPMLimits: &orgTokenDPMLimitsModel{
		DPMLimit: types.Int32Unknown(), DPMNotificationThreshold: types.Int32Unknown(),
	}}
	_, diagnostics = model.createUpdateRequest(ctx)
	assert.True(t, diagnostics.HasError())

	model = resourceOrgTokenModel{Notifications: types.ListNull(types.StringType)}
	diagnostics = model.updateFromAPI(ctx, &orgtoken.Token{
		Notifications: invalidAPINotification(),
	}, false, false)
	assert.True(t, diagnostics.HasError())
}

func TestResourceOrgTokenRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceOrgToken{}
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

func TestResourceOrgTokenMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := orgtoken.Token{}
	deleted := false

	writeCurrent := func(w http.ResponseWriter, includeSecret bool) {
		mu.Lock()
		defer mu.Unlock()
		response := current
		if !includeSecret {
			response.Secret = ""
		}
		//nolint:gosec // Mock API response intentionally includes an organization-token secret.
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("write organization token response: %v", err)
		}
	}
	updateCurrent := func(payload orgtoken.CreateUpdateTokenRequest, name string) {
		mu.Lock()
		defer mu.Unlock()
		current = orgtoken.Token{
			Name: name, AuthScopes: payload.AuthScopes, Description: payload.Description,
			Limits: payload.Limits, Notifications: payload.Notifications, Disabled: payload.Disabled,
			Secret: "created-secret", Expiry: 1735689600000,
		}
	}

	endpoints := map[string]http.Handler{
		"POST /v2/token": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeOrgTokenPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "primary-token", payload.Name)
			assert.Equal(t, []string{"API", "INGEST"}, payload.AuthScopes)
			if assert.NotNil(t, payload.Limits.CategoryQuota) {
				assert.Equal(t, int64(100), *payload.Limits.CategoryQuota.HostThreshold)
			}
			if assert.NotNil(t, payload.Limits.CategoryNotificationThreshold) {
				assert.Equal(t, int64(1800), *payload.Limits.CategoryNotificationThreshold.CustomMetricThreshold)
			}
			assert.Nil(t, payload.Limits.DpmQuota)
			if assert.Len(t, payload.Notifications, 1) {
				assert.Equal(t, "Email", payload.Notifications[0].Type)
			}
			updateCurrent(payload, "primary-token")
			writeCurrent(w, true)
		}),
		"GET /v2/token/primary-token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeCurrent(w, false)
		}),
		"PUT /v2/token/primary-token": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := decodeOrgTokenPayload(t, w, r)
			if !ok {
				return
			}
			assert.Equal(t, "updated-token", payload.Name)
			assert.True(t, payload.Disabled)
			if assert.NotNil(t, payload.Limits.DpmQuota) {
				assert.Equal(t, int32(5000), *payload.Limits.DpmQuota)
			}
			if assert.NotNil(t, payload.Limits.DpmNotificationThreshold) {
				assert.Equal(t, int32(4500), *payload.Limits.DpmNotificationThreshold)
			}
			assert.Nil(t, payload.Limits.CategoryQuota)
			if assert.Len(t, payload.Notifications, 1) {
				assert.Equal(t, "Slack", payload.Notifications[0].Type)
			}
			updateCurrent(payload, "updated-token")
			writeCurrent(w, false)
		}),
		"GET /v2/token/updated-token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeCurrent(w, false)
		}),
		"DELETE /v2/token/updated-token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceOrgToken)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/org_token_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "id", "primary-token"),
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "secret", "created-secret"),
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "expires_at", "1735689600000"),
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "host_or_usage_limits.host_limit", "100"),
				testresource.TestCheckNoResourceAttr("signalfx_org_token.test", "dpm_limits"),
			)},
			{ConfigFile: config.StaticFile("testdata/org_token_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "id", "updated-token"),
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "name", "updated-token"),
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "secret", "created-secret"),
				testresource.TestCheckResourceAttr("signalfx_org_token.test", "dpm_limits.dpm_limit", "5000"),
				testresource.TestCheckNoResourceAttr("signalfx_org_token.test", "host_or_usage_limits"),
			)},
			{ConfigFile: config.StaticFile("testdata/org_token_update.tf"), PlanOnly: true},
			{
				ResourceName: "signalfx_org_token.test", ImportState: true, ImportStateId: "updated-token",
				ImportStateVerify: true, ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func TestResourceOrgTokenRemovesMissingState(t *testing.T) {
	var mu sync.Mutex
	getCalls := 0
	current := orgtoken.Token{Name: "primary-token", Expiry: 1}
	endpoints := map[string]http.Handler{
		"POST /v2/token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			//nolint:gosec // Mock API response intentionally serializes the API token model.
			_ = json.NewEncoder(w).Encode(current)
		}),
		"GET /v2/token/primary-token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			getCalls++
			if getCalls == 1 {
				//nolint:gosec // Mock API response intentionally serializes the API token model.
				_ = json.NewEncoder(w).Encode(current)
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}),
		"DELETE /v2/token/primary-token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceOrgToken)),
		Steps: []testresource.TestStep{
			{Config: `resource "signalfx_org_token" "test" { name = "primary-token" }`},
			{Config: `resource "signalfx_org_token" "test" { name = "primary-token" }`, PlanOnly: true, ExpectNonEmptyPlan: true},
		},
	})
}

func TestResourceOrgTokenErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		config    string
		endpoints map[string]http.Handler
		error     *regexp.Regexp
	}{
		{
			name: "API error", config: `resource "signalfx_org_token" "test" { name = "failed-token" }`,
			endpoints: map[string]http.Handler{"POST /v2/token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "failed", http.StatusBadGateway)
			})},
			error: regexp.MustCompile(`status code 502`),
		},
		{
			name: "missing API identifier", config: `resource "signalfx_org_token" "test" { name = "empty-token" }`,
			endpoints: map[string]http.Handler{"POST /v2/token": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{}`))
			})},
			error: regexp.MustCompile(`returned no resource identifier`),
		},
		{name: "missing name", config: `resource "signalfx_org_token" "test" {}`, error: regexp.MustCompile(`argument "name" is required`)},
		{
			name: "missing DPM limit", config: `resource "signalfx_org_token" "test" {
  name = "invalid-token"
  dpm_limits {}
}`,
			error: regexp.MustCompile(`dpm_limit attribute is required`),
		},
		{
			name: "conflicting limits", config: `resource "signalfx_org_token" "test" {
  name = "invalid-token"
  host_or_usage_limits {}
  dpm_limits { dpm_limit = 100 }
}`,
			error: regexp.MustCompile(`cannot be specified`),
		},
		{
			name: "invalid notification", config: `resource "signalfx_org_token" "test" {
  name = "invalid-token"
  notifications = ["invalid"]
}`,
			error: regexp.MustCompile(`Invalid notification destination`),
		},
		{
			name: "DPM overflow", config: `resource "signalfx_org_token" "test" {
  name = "invalid-token"
  dpm_limits { dpm_limit = 2147483648 }
}`,
			error: regexp.MustCompile(`32-bit[[:space:]]+integer`),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testresource.UnitTest(t, testresource.TestCase{
				ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, test.endpoints, fwtest.WithMockResources(NewResourceOrgToken)),
				Steps:                    []testresource.TestStep{{Config: test.config, ExpectError: test.error}},
			})
		})
	}
}

func stringList(values ...string) types.List {
	elements := make([]attr.Value, len(values))
	for index, value := range values {
		elements[index] = types.StringValue(value)
	}
	return types.ListValueMust(types.StringType, elements)
}

func invalidAPINotification() []*notification.Notification {
	return []*notification.Notification{{Type: "Unknown", Value: struct{}{}}}
}

func listStrings(t *testing.T, value types.List) []string {
	t.Helper()
	var result []string
	require.False(t, value.ElementsAs(context.Background(), &result, false).HasError())
	return result
}

func decodeOrgTokenPayload(t *testing.T, w http.ResponseWriter, r *http.Request) (orgtoken.CreateUpdateTokenRequest, bool) {
	t.Helper()
	var payload orgtoken.CreateUpdateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Errorf("decode organization token payload: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return payload, false
	}
	return payload, true
}
