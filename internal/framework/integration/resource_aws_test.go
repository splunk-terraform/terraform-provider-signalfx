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

func TestResourceAWSMetadataAndSchema(t *testing.T) {
	t.Parallel()
	implementation := NewResourceAWS()
	metadata := &resource.MetadataResponse{}
	implementation.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "signalfx"}, metadata)
	assert.Equal(t, "signalfx_aws_integration", metadata.TypeName)
	modelWithoutNestedBlocks := struct {
		ID                             types.String `tfsdk:"id"`
		IntegrationID                  types.String `tfsdk:"integration_id"`
		Name                           types.String `tfsdk:"name"`
		Enabled                        types.Bool   `tfsdk:"enabled"`
		AuthMethod                     types.String `tfsdk:"auth_method"`
		CustomCloudWatchNamespaces     types.Set    `tfsdk:"custom_cloudwatch_namespaces"`
		EnableAWSUsage                 types.Bool   `tfsdk:"enable_aws_usage"`
		ImportCloudWatch               types.Bool   `tfsdk:"import_cloud_watch"`
		Key                            types.String `tfsdk:"key"`
		Regions                        types.Set    `tfsdk:"regions"`
		RoleARN                        types.String `tfsdk:"role_arn"`
		Services                       types.Set    `tfsdk:"services"`
		Token                          types.String `tfsdk:"token"`
		PollRate                       types.Int64  `tfsdk:"poll_rate"`
		InactiveMetricsPollRate        types.Int64  `tfsdk:"inactive_metrics_poll_rate"`
		ExternalID                     types.String `tfsdk:"external_id"`
		UseMetricStreamsSync           types.Bool   `tfsdk:"use_metric_streams_sync"`
		NamedToken                     types.String `tfsdk:"named_token"`
		EnableCheckLargeVolume         types.Bool   `tfsdk:"enable_check_large_volume"`
		SyncCustomNamespacesOnly       types.Bool   `tfsdk:"sync_custom_namespaces_only"`
		CollectOnlyRecommendedStats    types.Bool   `tfsdk:"collect_only_recommended_stats"`
		MetricStreamsManagedExternally types.Bool   `tfsdk:"metric_streams_managed_externally"`
	}{
		CustomCloudWatchNamespaces: types.SetValueMust(types.StringType, nil),
		Regions:                    types.SetValueMust(types.StringType, nil),
		Services:                   types.SetValueMust(types.StringType, nil),
	}
	assert.NoError(t, fwtest.ResourceSchemaValidate(implementation, modelWithoutNestedBlocks))

	response := &resource.SchemaResponse{}
	implementation.Schema(context.Background(), resource.SchemaRequest{}, response)
	assert.Len(t, response.Schema.Attributes["integration_id"].(schema.StringAttribute).PlanModifiers, 1)
	assert.True(t, response.Schema.Attributes["name"].IsComputed())
	assert.True(t, response.Schema.Attributes["auth_method"].IsComputed())
	assert.True(t, response.Schema.Attributes["token"].IsSensitive())
	assert.True(t, response.Schema.Attributes["key"].IsSensitive())
	assert.True(t, response.Schema.Attributes["external_id"].IsSensitive())
	assert.Len(t, response.Schema.Attributes["regions"].(schema.SetAttribute).Validators, 1)
	assert.Len(t, response.Schema.Attributes["external_id"].(schema.StringAttribute).Validators, 1)
	assert.Len(t, response.Schema.Blocks["namespace_sync_rule"].(schema.SetNestedBlock).Validators, 1)
	assert.Len(t, response.Schema.Blocks["custom_namespace_sync_rule"].(schema.SetNestedBlock).Validators, 1)
	_, supportsImport := implementation.(resource.ResourceWithImportState)
	assert.False(t, supportsImport, "the SDK resource did not support import")
}

func TestResourceAWSModelExternalAndTokenPayloads(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rules := awsRuleSet(t, awsNamespaceSyncRuleModel{
		DefaultAction: types.StringValue("Exclude"),
		FilterAction:  types.StringValue("Include"),
		FilterSource:  types.StringValue("filter('aws_tag_env', 'prod')"),
		Namespace:     types.StringValue("AWS/EC2"),
	})
	stats := awsMetricSet(t, awsMetricStatsModel{
		Namespace: types.StringValue("AWS/EC2"), Metric: types.StringValue("CPUUtilization"),
		Stats: awsStringSet(t, "Average", "p99"),
	})
	model := resourceAWSModel{
		ID: types.StringValue("aws-id"), IntegrationID: types.StringValue("aws-id"), Name: types.StringValue("Primary AWS"),
		Enabled: types.BoolValue(true), ExternalID: types.StringValue("external-id"), RoleARN: types.StringValue("role-arn"),
		Regions: awsStringSet(t, "us-east-1", "us-west-2"), Services: awsStringSet(t, "ec2", "s3"),
		CustomCloudWatchNamespaces: awsStringSet(t, "Example/One", "Example/Two"),
		CustomNamespaceSyncRule:    rules, NamespaceSyncRule: rules, MetricStatsToSync: stats,
		EnableAWSUsage: types.BoolValue(true), ImportCloudWatch: types.BoolValue(true), PollRate: types.Int64Value(60),
		InactiveMetricsPollRate: types.Int64Value(3600), UseMetricStreamsSync: types.BoolValue(false),
		NamedToken: types.StringValue("ingest"), EnableCheckLargeVolume: types.BoolValue(true),
		SyncCustomNamespacesOnly: types.BoolValue(true), CollectOnlyRecommendedStats: types.BoolValue(true),
		MetricStreamsManagedExternally: types.BoolValue(true),
	}
	payload, diags := model.awsIntegration(ctx, true)
	require.False(t, diags.HasError())
	assert.Equal(t, integration.EXTERNAL_ID, payload.AuthMethod)
	assert.Equal(t, "external-id", payload.ExternalId)
	assert.Equal(t, "role-arn", payload.RoleArn)
	assert.Equal(t, int64(60000), payload.PollRate)
	assert.Equal(t, int64(3600000), payload.InactiveMetricsPollRate)
	assert.Equal(t, awsMetricStreamsCancelling, payload.MetricStreamsSyncState)
	assert.ElementsMatch(t, []string{"us-east-1", "us-west-2"}, payload.Regions)
	assert.ElementsMatch(t, []integration.AwsService{"ec2", "s3"}, payload.Services)
	assert.Len(t, payload.CustomNamespaceSyncRules, 1)
	assert.Len(t, payload.NamespaceSyncRules, 1)
	assert.Equal(t, []string{"Average", "p99"}, payload.MetricStatsToSync["AWS/EC2"]["CPUUtilization"])
	assert.Contains(t, []string{"Example/One,Example/Two", "Example/Two,Example/One"}, payload.CustomCloudWatchNamespaces)

	model.ExternalID = types.StringNull()
	model.RoleARN = types.StringNull()
	model.Token = types.StringValue("security-token")
	model.Key = types.StringValue("access-key")
	model.UseMetricStreamsSync = types.BoolValue(true)
	payload, diags = model.awsIntegration(ctx, true)
	require.False(t, diags.HasError())
	assert.Equal(t, integration.SECURITY_TOKEN, payload.AuthMethod)
	assert.Equal(t, "security-token", payload.Token)
	assert.Equal(t, "access-key", payload.Key)
	assert.Equal(t, awsMetricStreamsEnabled, payload.MetricStreamsSyncState)
}

func TestResourceAWSModelValidationAndHelpers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceAWSModel{Regions: awsStringSet(t, "us-east-1")}
	payload, diags := model.awsIntegration(ctx, false)
	assert.Nil(t, payload)
	assert.True(t, diags.HasError())

	model.Token = types.StringValue("token")
	model.Regions = types.SetValueMust(types.StringType, nil)
	payload, diags = model.awsIntegration(ctx, false)
	assert.Nil(t, payload)
	assert.True(t, diags.HasError())

	assert.Nil(t, namespaceFilter(awsNamespaceSyncRuleModel{FilterAction: types.StringNull()}))
	assert.NotNil(t, namespaceFilter(awsNamespaceSyncRuleModel{
		FilterAction: types.StringValue("Include"), FilterSource: types.StringValue("filter"),
	}))
	custom, customDiags := customNamespaceRules(ctx, types.SetNull(awsNamespaceRuleObjectType))
	assert.Empty(t, custom)
	assert.Empty(t, customDiags)
	rules, ruleDiags := namespaceRules(ctx, types.SetNull(awsNamespaceRuleObjectType))
	assert.Empty(t, rules)
	assert.Empty(t, ruleDiags)
	metricValues, metricDiags := metricStats(ctx, types.SetNull(awsMetricStatsObjectType))
	assert.Nil(t, metricValues)
	assert.Empty(t, metricDiags)
	assert.Equal(t, types.StringNull(), optionalStringValue(""))
	assert.Equal(t, types.StringValue("value"), optionalStringValue("value"))
}

func TestResourceAWSUpdateFromAPI(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	model := resourceAWSModel{
		ID: types.StringValue("old-id"), IntegrationID: types.StringValue("bootstrap-id"),
		Token: types.StringValue("preserved-token"), Key: types.StringValue("preserved-key"),
		ExternalID: types.StringValue("preserved-external"), RoleARN: types.StringValue("preserved-role"),
		Services: awsStringSet(t, "ec2"), Regions: awsStringSet(t, "us-east-1"),
		CustomNamespaceSyncRule: types.SetNull(awsNamespaceRuleObjectType),
		NamespaceSyncRule:       types.SetNull(awsNamespaceRuleObjectType),
		MetricStatsToSync:       types.SetNull(awsMetricStatsObjectType),
	}
	assert.Empty(t, model.updateFromAPI(ctx, nil, true))
	diags := model.updateFromAPI(ctx, &integration.AwsCloudWatchIntegration{
		Id: "new-id", Name: "API AWS", Enabled: true, AuthMethod: integration.EXTERNAL_ID,
		PollRate: 600000, InactiveMetricsPollRate: 3600000, Regions: []string{"eu-west-1"},
		Services: []integration.AwsService{"s3"}, EnableAwsUsage: true, ImportCloudWatch: true,
		MetricStreamsSyncState: awsMetricStreamsEnabled, EnableCheckLargeVolume: true,
		SyncCustomNamespacesOnly: true, CollectOnlyRecommendedStats: true, MetricStreamsManagedExternally: true,
		CustomNamespaceSyncRules: []*integration.AwsCustomNameSpaceSyncRule{{
			DefaultAction: integration.EXCLUDE, Namespace: "Example/Custom",
			Filter: &integration.AwsSyncRuleFilter{Action: integration.INCLUDE, Source: "filter"},
		}},
		NamespaceSyncRules: []*integration.AwsNameSpaceSyncRule{{Namespace: "lambda"}},
		MetricStatsToSync:  map[string]map[string][]string{"AWS/EC2": {"CPUUtilization": {"Average"}}},
	}, true)
	require.False(t, diags.HasError())
	assert.Equal(t, types.StringValue("new-id"), model.ID)
	assert.Equal(t, types.StringValue("bootstrap-id"), model.IntegrationID)
	assert.Equal(t, types.StringValue("preserved-token"), model.Token)
	assert.Equal(t, types.StringValue("preserved-key"), model.Key)
	assert.Equal(t, types.StringValue("preserved-external"), model.ExternalID)
	assert.Equal(t, types.StringValue("preserved-role"), model.RoleARN)
	assert.Equal(t, types.Int64Value(600), model.PollRate)
	assert.Equal(t, types.Int64Value(3600), model.InactiveMetricsPollRate)
	assert.True(t, model.UseMetricStreamsSync.ValueBool())
	assert.Equal(t, "s3", model.Services.Elements()[0].(types.String).ValueString())
	assert.True(t, model.NamespaceSyncRule.IsNull(), "configured services keep the services representation")
	assert.Len(t, model.CustomNamespaceSyncRule.Elements(), 1)
	assert.Len(t, model.MetricStatsToSync.Elements(), 1)

	model.Services = types.SetNull(types.StringType)
	diags = model.updateFromAPI(ctx, &integration.AwsCloudWatchIntegration{
		Name: "Rules", NamespaceSyncRules: []*integration.AwsNameSpaceSyncRule{{
			DefaultAction: integration.INCLUDE, Namespace: "lambda",
		}},
		CustomCloudWatchNamespaces: "Example/One,Example/Two",
	}, false)
	require.False(t, diags.HasError())
	assert.Len(t, model.NamespaceSyncRule.Elements(), 1)
	assert.Len(t, model.CustomCloudWatchNamespaces.Elements(), 2)
	assert.Equal(t, types.StringValue("new-id"), model.ID)
}

func TestResourceAWSRejectsInvalidFrameworkData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	implementation := &ResourceAWS{}
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
	implementation.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, updateResponse)
	assert.True(t, updateResponse.Diagnostics.HasError())
	deleteResponse := &resource.DeleteResponse{}
	implementation.Delete(ctx, resource.DeleteRequest{State: state}, deleteResponse)
	assert.True(t, deleteResponse.Diagnostics.HasError())
}

func TestResourceAWSMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := integration.AwsCloudWatchIntegration{
		Id: "aws-id", Name: "Primary AWS", Type: "AWSCloudWatch", AuthMethod: integration.EXTERNAL_ID,
		ExternalId: "external-id", Enabled: true, PollRate: 300000, InactiveMetricsPollRate: 1200000,
		Regions: []string{"us-east-1", "us-west-2"}, MetricStreamsSyncState: awsMetricStreamsDisabled,
	}
	deleted := false
	putStates := make([]string, 0, 3)
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		require.NoError(t, json.NewEncoder(w).Encode(current))
	}
	endpoints := map[string]http.Handler{
		"GET /v2/integration/aws-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			isDeleted := deleted
			mu.Unlock()
			if isDeleted {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			writeCurrent(w)
		}),
		"PUT /v2/integration/aws-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload integration.AwsCloudWatchIntegration
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode AWS payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.Equal(t, integration.Type("AWSCloudWatch"), payload.Type)
			assert.Equal(t, integration.EXTERNAL_ID, payload.AuthMethod)
			if payload.MetricStreamsSyncState != awsMetricStreamsCancelling {
				assert.Equal(t, "external-id", payload.ExternalId)
			}
			mu.Lock()
			putStates = append(putStates, payload.MetricStreamsSyncState)
			payload.Id = "aws-id"
			payload.ExternalId = ""
			payload.RoleArn = ""
			switch payload.MetricStreamsSyncState {
			case awsMetricStreamsCancelling, "":
				payload.MetricStreamsSyncState = awsMetricStreamsDisabled
			}
			current = payload
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/aws-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWS)),
		Steps: []testresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/aws_create.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "id", "aws-id"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "name", "Primary AWS"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "auth_method", string(integration.EXTERNAL_ID)),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "services.#", "2"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "metric_stats_to_sync.#", "1"),
			)},
			{ConfigFile: config.StaticFile("testdata/aws_create.tf"), PlanOnly: true},
			{ConfigFile: config.StaticFile("testdata/aws_update.tf"), Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "poll_rate", "600"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "use_metric_streams_sync", "true"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "namespace_sync_rule.#", "1"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "custom_cloudwatch_namespaces.#", "1"),
			)},
			{ConfigFile: config.StaticFile("testdata/aws_update.tf"), PlanOnly: true},
		},
	})

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
	assert.Contains(t, putStates, awsMetricStreamsEnabled)
	assert.Contains(t, putStates, awsMetricStreamsCancelling)
}

func TestResourceAWSInvalidConfiguration(t *testing.T) {
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, nil, fwtest.WithMockResources(NewResourceAWS)),
		Steps: []testresource.TestStep{{
			ConfigFile:  config.StaticFile("testdata/aws_invalid.tf"),
			PlanOnly:    true,
			ExpectError: regexp.MustCompile("(exactly one|at least 1|conflicts)"),
		}},
	})
}

func TestResourceAWSFullTokenMockedLifecycle(t *testing.T) {
	var mu sync.Mutex
	current := integration.AwsCloudWatchIntegration{
		Id: "aws-token-id", Name: "Token AWS", Type: "AWSCloudWatch", AuthMethod: integration.SECURITY_TOKEN,
		Enabled: true, PollRate: 300000, InactiveMetricsPollRate: 1200000, Regions: []string{"ap-southeast-2"},
	}
	deleted := false
	writeCurrent := func(w http.ResponseWriter) {
		mu.Lock()
		defer mu.Unlock()
		require.NoError(t, json.NewEncoder(w).Encode(current))
	}
	endpoints := map[string]http.Handler{
		"GET /v2/integration/aws-token-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { writeCurrent(w) }),
		"PUT /v2/integration/aws-token-id": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var payload integration.AwsCloudWatchIntegration
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode AWS token payload: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.Equal(t, integration.SECURITY_TOKEN, payload.AuthMethod)
			assert.Equal(t, "security-token", payload.Token)
			assert.Equal(t, "access-key", payload.Key)
			mu.Lock()
			payload.Id = "aws-token-id"
			payload.Token = ""
			payload.Key = ""
			current = payload
			mu.Unlock()
			writeCurrent(w)
		}),
		"DELETE /v2/integration/aws-token-id": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			deleted = true
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		}),
	}
	testresource.UnitTest(t, testresource.TestCase{
		ProtoV6ProviderFactories: fwtest.NewMockProto6Server(t, endpoints, fwtest.WithMockResources(NewResourceAWS)),
		Steps: []testresource.TestStep{{
			ConfigFile: config.StaticFile("testdata/aws_token_full.tf"),
			Check: testresource.ComposeAggregateTestCheckFunc(
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "id", "aws-token-id"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "auth_method", string(integration.SECURITY_TOKEN)),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "token", "security-token"),
				testresource.TestCheckResourceAttr("signalfx_aws_integration.test", "key", "access-key"),
			),
		}},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, deleted)
}

func awsStringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	value, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	require.False(t, diags.HasError())
	return value
}

func awsRuleSet(t *testing.T, values ...awsNamespaceSyncRuleModel) types.Set {
	t.Helper()
	value, diags := types.SetValueFrom(context.Background(), awsNamespaceRuleObjectType, values)
	require.False(t, diags.HasError())
	return value
}

func awsMetricSet(t *testing.T, values ...awsMetricStatsModel) types.Set {
	t.Helper()
	value, diags := types.SetValueFrom(context.Background(), awsMetricStatsObjectType, values)
	require.False(t, diags.HasError())
	return value
}
