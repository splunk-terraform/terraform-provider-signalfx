// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

const (
	awsMetricStreamsEnabled    = "ENABLED"
	awsMetricStreamsDisabled   = "DISABLED"
	awsMetricStreamsCancelling = "CANCELLING" //nolint:misspell // The API uses this spelling.
	awsMetricStreamsFailed     = "CANCELLATION_FAILED"
	awsPollInterval            = 5 * time.Second
	awsMetricStreamsTimeout    = 19 * time.Minute
)

type ResourceAWS struct {
	fwembed.ResourceData
	pollInterval time.Duration
}

type resourceAWSModel struct {
	ID                             types.String `tfsdk:"id"`
	IntegrationID                  types.String `tfsdk:"integration_id"`
	Name                           types.String `tfsdk:"name"`
	Enabled                        types.Bool   `tfsdk:"enabled"`
	AuthMethod                     types.String `tfsdk:"auth_method"`
	CustomCloudWatchNamespaces     types.Set    `tfsdk:"custom_cloudwatch_namespaces"`
	CustomNamespaceSyncRule        types.Set    `tfsdk:"custom_namespace_sync_rule"`
	NamespaceSyncRule              types.Set    `tfsdk:"namespace_sync_rule"`
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
	MetricStatsToSync              types.Set    `tfsdk:"metric_stats_to_sync"`
	SyncCustomNamespacesOnly       types.Bool   `tfsdk:"sync_custom_namespaces_only"`
	CollectOnlyRecommendedStats    types.Bool   `tfsdk:"collect_only_recommended_stats"`
	MetricStreamsManagedExternally types.Bool   `tfsdk:"metric_streams_managed_externally"`
}

type awsNamespaceSyncRuleModel struct {
	DefaultAction types.String `tfsdk:"default_action"`
	FilterAction  types.String `tfsdk:"filter_action"`
	FilterSource  types.String `tfsdk:"filter_source"`
	Namespace     types.String `tfsdk:"namespace"`
}

type awsMetricStatsModel struct {
	Namespace types.String `tfsdk:"namespace"`
	Metric    types.String `tfsdk:"metric"`
	Stats     types.Set    `tfsdk:"stats"`
}

var (
	_ resource.Resource              = (*ResourceAWS)(nil)
	_ resource.ResourceWithConfigure = (*ResourceAWS)(nil)

	awsNamespaceRuleAttributeTypes = map[string]attr.Type{
		"default_action": types.StringType,
		"filter_action":  types.StringType,
		"filter_source":  types.StringType,
		"namespace":      types.StringType,
	}
	awsNamespaceRuleObjectType   = types.ObjectType{AttrTypes: awsNamespaceRuleAttributeTypes}
	awsMetricStatsAttributeTypes = map[string]attr.Type{
		"namespace": types.StringType,
		"metric":    types.StringType,
		"stats":     types.SetType{ElemType: types.StringType},
	}
	awsMetricStatsObjectType = types.ObjectType{AttrTypes: awsMetricStatsAttributeTypes}
)

func NewResourceAWS() resource.Resource {
	return &ResourceAWS{pollInterval: awsPollInterval}
}

func (aws *ResourceAWS) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_integration"
}

func (aws *ResourceAWS) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	filterAction := []validator.String{stringvalidator.OneOf(string(integration.INCLUDE), string(integration.EXCLUDE))}
	attributes := map[string]schema.Attribute{
		"id": fwshared.ResourceIDAttribute(),
		"integration_id": schema.StringAttribute{
			Required: true, Description: "ID of the AWS bootstrap integration to complete.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"name": schema.StringAttribute{
			Computed: true, Description: "Name configured by the AWS bootstrap integration resource.",
		},
		"enabled": schema.BoolAttribute{Required: true, Description: "Whether the integration is enabled."},
		"auth_method": schema.StringAttribute{
			Computed: true, Description: "AWS authentication mechanism selected by the bootstrap integration.",
		},
		"custom_cloudwatch_namespaces": schema.SetAttribute{
			Optional: true, ElementType: types.StringType,
			Description: "Custom AWS CloudWatch namespaces to monitor.",
			Validators:  []validator.Set{setvalidator.ConflictsWith(path.MatchRoot("custom_namespace_sync_rule"))},
		},
		"enable_aws_usage":   optionalComputedBool("Whether to import AWS usage metrics for AWS Cost Optimizer."),
		"import_cloud_watch": optionalComputedBool("Whether to import CloudWatch metrics."),
		"key": schema.StringAttribute{
			Optional: true, Sensitive: true, Description: "AWS access key used with security-token authentication.",
			Validators: []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("role_arn"), path.MatchRoot("external_id"))},
		},
		"regions": schema.SetAttribute{
			Required: true, ElementType: types.StringType, Description: "AWS regions to monitor.",
			Validators: []validator.Set{setvalidator.SizeAtLeast(1)},
		},
		"role_arn": schema.StringAttribute{
			Optional: true, Description: "AWS IAM role ARN used with external-ID authentication.",
			Validators: []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("token"), path.MatchRoot("key"))},
		},
		"services": schema.SetAttribute{
			Optional: true, ElementType: types.StringType, Description: "AWS services to monitor.",
			Validators: []validator.Set{setvalidator.ConflictsWith(path.MatchRoot("namespace_sync_rule"))},
		},
		"token": schema.StringAttribute{
			Optional: true, Sensitive: true, Description: "AWS security token used with token authentication.",
			Validators: []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("role_arn"), path.MatchRoot("external_id"))},
		},
		"poll_rate": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(300),
			Description: "AWS polling interval in seconds. Defaults to 300.",
			Validators:  []validator.Int64{int64validator.Between(60, 600)},
		},
		"inactive_metrics_poll_rate": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(1200),
			Description: "Inactive-metric polling interval in seconds. Defaults to 1200.",
			Validators:  []validator.Int64{int64validator.Between(60, 3600)},
		},
		"external_id": schema.StringAttribute{
			Optional: true, Sensitive: true, Description: "External ID generated by the AWS external bootstrap integration.",
			Validators: []validator.String{stringvalidator.ConflictsWith(path.MatchRoot("token"), path.MatchRoot("key"))},
		},
		"use_metric_streams_sync": optionalComputedBool(
			"Whether to synchronize metrics using CloudWatch Metric Streams. This requires CloudWatch Metric Streams lifecycle permissions and `iam:PassRole` in AWS.",
		),
		"named_token":                    schema.StringAttribute{Optional: true, Description: "Named organization token used for ingestion."},
		"enable_check_large_volume":      optionalComputedBool("Whether to check for unusually large data volumes."),
		"sync_custom_namespaces_only":    optionalComputedBool("Whether to synchronize only custom namespace metrics and metadata."),
		"collect_only_recommended_stats": optionalComputedBool("Whether to synchronize only AWS-recommended statistics."),
		"metric_streams_managed_externally": optionalComputedBool(
			"Whether AWS Metric Streams are managed outside Splunk Observability Cloud. Requires `use_metric_streams_sync` to be true.",
		),
	}

	ruleAttributes := func(namespaceDescription string) map[string]schema.Attribute {
		return map[string]schema.Attribute{
			"default_action": schema.StringAttribute{Optional: true, Description: "Action for data that does not match the filter.", Validators: filterAction},
			"filter_action":  schema.StringAttribute{Optional: true, Description: "Action for data matching the filter.", Validators: filterAction},
			"filter_source":  schema.StringAttribute{Optional: true, Description: "SignalFlow filter expression."},
			"namespace":      schema.StringAttribute{Required: true, Description: namespaceDescription},
		}
	}

	resp.Schema = schema.Schema{
		Description: "Completes and manages an AWS CloudWatch integration bootstrapped by an AWS external-ID or token integration resource.",
		Attributes:  attributes,
		Blocks: map[string]schema.Block{
			"custom_namespace_sync_rule": schema.SetNestedBlock{
				Description:  "Rules controlling synchronization of custom AWS namespaces.",
				Validators:   []validator.Set{setvalidator.ConflictsWith(path.MatchRoot("custom_cloudwatch_namespaces"))},
				NestedObject: schema.NestedBlockObject{Attributes: ruleAttributes("Custom AWS namespace to synchronize.")},
			},
			"namespace_sync_rule": schema.SetNestedBlock{
				Description:  "Rules controlling synchronization of AWS service namespaces.",
				Validators:   []validator.Set{setvalidator.ConflictsWith(path.MatchRoot("services"))},
				NestedObject: schema.NestedBlockObject{Attributes: ruleAttributes("AWS service namespace to synchronize.")},
			},
			"metric_stats_to_sync": schema.SetNestedBlock{
				Description: "AWS statistics to collect for individual metrics. Without Metric Streams, only the configured statistics are retrieved; with Metric Streams, use this block for additional percentile statistics. Omit it to retrieve the standard statistics.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"namespace": schema.StringAttribute{Required: true, Description: "AWS metric namespace."},
					"metric":    schema.StringAttribute{Required: true, Description: "AWS metric name."},
					"stats": schema.SetAttribute{
						Required: true, ElementType: types.StringType, Description: "AWS statistics to collect.",
					},
				}},
			},
		},
	}
}

func optionalComputedBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: description + " Defaults to false.",
	}
}

func (aws *ResourceAWS) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceAWSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bootstrap, err := aws.Details().Client.GetAWSCloudWatchIntegration(ctx, model.IntegrationID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); err != nil || resp.Diagnostics.HasError() {
		return
	}
	if bootstrap == nil {
		resp.Diagnostics.AddError("Unable to complete AWS integration", "The bootstrap integration response was empty.")
		return
	}
	model.ID = types.StringValue(bootstrap.Id)
	model.Name = types.StringValue(bootstrap.Name)
	model.AuthMethod = types.StringValue(string(bootstrap.AuthMethod))
	if bootstrap.AuthMethod == integration.EXTERNAL_ID && bootstrap.ExternalId != "" {
		model.ExternalID = types.StringValue(bootstrap.ExternalId)
	}

	details := aws.update(ctx, &model, model.UseMetricStreamsSync.ValueBool(), model.UseMetricStreamsSync.ValueBool(), &resp.Diagnostics, resp.State)
	if resp.Diagnostics.HasError() || details == nil {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (aws *ResourceAWS) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceAWSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := aws.Details().Client.GetAWSCloudWatchIntegration(ctx, model.IntegrationID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); err != nil || resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (aws *ResourceAWS) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resourceAWSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	metricStreamsChanged := plan.UseMetricStreamsSync.ValueBool() != state.UseMetricStreamsSync.ValueBool()
	details := aws.update(ctx, &plan, metricStreamsChanged, metricStreamsChanged, &resp.Diagnostics, resp.State)
	if resp.Diagnostics.HasError() || details == nil {
		return
	}
	resp.Diagnostics.Append(plan.updateFromAPI(ctx, details, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (aws *ResourceAWS) update(
	ctx context.Context,
	model *resourceAWSModel,
	metricStreamsChanged bool,
	wait bool,
	diagnostics *diag.Diagnostics,
	state tfsdk.State,
) *integration.AwsCloudWatchIntegration {
	payload, modelDiags := model.awsIntegration(ctx, metricStreamsChanged)
	diagnostics.Append(modelDiags...)
	if diagnostics.HasError() {
		return nil
	}
	details, err := aws.Details().Client.UpdateAWSCloudWatchIntegration(ctx, model.ID.ValueString(), payload)
	if err != nil {
		diagnostics.Append(fwerr.ErrorHandler(ctx, state, withAdminTokenHelp(err))...)
		return nil
	}
	if wait {
		details, err = aws.waitForMetricStreams(ctx, model.ID.ValueString(), model.UseMetricStreamsSync.ValueBool())
		if err != nil {
			diagnostics.AddError("Unable to update AWS integration", err.Error())
			return nil
		}
	}
	return details
}

func (aws *ResourceAWS) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceAWSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := aws.Details().Client.GetAWSCloudWatchIntegration(ctx, model.ID.ValueString())
	if err != nil {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
		return
	}
	if details != nil && details.Enabled && details.MetricStreamsSyncState != "" && details.MetricStreamsSyncState != awsMetricStreamsDisabled {
		details.MetricStreamsSyncState = awsMetricStreamsCancelling
		_, err = aws.Details().Client.UpdateAWSCloudWatchIntegration(ctx, model.ID.ValueString(), details)
		if err != nil {
			resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...)
			return
		}
		if _, err = aws.waitForMetricStreams(ctx, model.ID.ValueString(), false); err != nil {
			resp.Diagnostics.AddError("Unable to disable AWS Metric Streams", err.Error())
			return
		}
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, aws.Details().Client.DeleteAWSCloudWatchIntegration(ctx, model.ID.ValueString()))...)
}

func (aws *ResourceAWS) waitForMetricStreams(ctx context.Context, id string, enabled bool) (*integration.AwsCloudWatchIntegration, error) {
	waitCtx, cancel := context.WithTimeout(ctx, awsMetricStreamsTimeout)
	defer cancel()

	interval := aws.pollInterval
	if interval <= 0 {
		interval = awsPollInterval
	}
	for {
		details, err := aws.Details().Client.GetAWSCloudWatchIntegration(waitCtx, id)
		if err != nil {
			return nil, err
		}
		state := details.MetricStreamsSyncState
		if enabled && state == awsMetricStreamsEnabled {
			return details, nil
		}
		if !enabled && (state == "" || state == awsMetricStreamsDisabled) {
			return details, nil
		}
		if enabled && state != awsMetricStreamsDisabled && state != awsMetricStreamsCancelling && state != awsMetricStreamsFailed {
			return nil, fmt.Errorf("unexpected Metric Streams state %q while waiting for enabled", state)
		}
		if !enabled && state != awsMetricStreamsEnabled && state != awsMetricStreamsCancelling {
			return nil, fmt.Errorf("unexpected Metric Streams state %q while waiting for disabled", state)
		}

		timer := time.NewTimer(interval)
		select {
		case <-waitCtx.Done():
			timer.Stop()
			return nil, fmt.Errorf("waiting for AWS integration %s Metric Streams state: %w", id, waitCtx.Err())
		case <-timer.C:
		}
	}
}

func (model resourceAWSModel) awsIntegration(ctx context.Context, metricStreamsChanged bool) (*integration.AwsCloudWatchIntegration, diag.Diagnostics) {
	payload := &integration.AwsCloudWatchIntegration{
		Type:                           "AWSCloudWatch",
		Name:                           model.Name.ValueString(),
		Enabled:                        model.Enabled.ValueBool(),
		EnableAwsUsage:                 model.EnableAWSUsage.ValueBool(),
		ImportCloudWatch:               model.ImportCloudWatch.ValueBool(),
		PollRate:                       model.PollRate.ValueInt64() * 1000,
		InactiveMetricsPollRate:        model.InactiveMetricsPollRate.ValueInt64() * 1000,
		NamedToken:                     model.NamedToken.ValueString(),
		EnableCheckLargeVolume:         model.EnableCheckLargeVolume.ValueBool(),
		SyncCustomNamespacesOnly:       model.SyncCustomNamespacesOnly.ValueBool(),
		CollectOnlyRecommendedStats:    model.CollectOnlyRecommendedStats.ValueBool(),
		MetricStreamsManagedExternally: model.MetricStreamsManagedExternally.ValueBool(),
	}
	if model.UseMetricStreamsSync.ValueBool() {
		payload.MetricStreamsSyncState = awsMetricStreamsEnabled
	} else if metricStreamsChanged {
		payload.MetricStreamsSyncState = awsMetricStreamsCancelling
	}

	var diags diag.Diagnostics
	switch {
	case !model.ExternalID.IsNull() && model.ExternalID.ValueString() != "":
		payload.AuthMethod = integration.EXTERNAL_ID
		payload.ExternalId = model.ExternalID.ValueString()
		payload.RoleArn = model.RoleARN.ValueString()
	case !model.Token.IsNull() && model.Token.ValueString() != "":
		payload.AuthMethod = integration.SECURITY_TOKEN
		payload.Token = model.Token.ValueString()
		payload.Key = model.Key.ValueString()
	default:
		diags.AddError("Invalid AWS authentication", "Specify exactly one of external_id or token.")
		return nil, diags
	}

	values, valueDiags := stringSetElements(ctx, model.CustomCloudWatchNamespaces)
	diags.Append(valueDiags...)
	payload.CustomCloudWatchNamespaces = strings.Join(values, ",")
	payload.Regions, valueDiags = stringSetElements(ctx, model.Regions)
	diags.Append(valueDiags...)
	if len(payload.Regions) == 0 {
		diags.AddError("Invalid AWS regions", "Configure at least one AWS region explicitly.")
	}
	services, valueDiags := stringSetElements(ctx, model.Services)
	diags.Append(valueDiags...)
	for _, service := range services {
		payload.Services = append(payload.Services, integration.AwsService(service))
	}

	payload.CustomNamespaceSyncRules, valueDiags = customNamespaceRules(ctx, model.CustomNamespaceSyncRule)
	diags.Append(valueDiags...)
	payload.NamespaceSyncRules, valueDiags = namespaceRules(ctx, model.NamespaceSyncRule)
	diags.Append(valueDiags...)
	payload.MetricStatsToSync, valueDiags = metricStats(ctx, model.MetricStatsToSync)
	diags.Append(valueDiags...)
	if diags.HasError() {
		return nil, diags
	}
	return payload, diags
}

func customNamespaceRules(ctx context.Context, value types.Set) ([]*integration.AwsCustomNameSpaceSyncRule, diag.Diagnostics) {
	models, diags := namespaceRuleModels(ctx, value)
	result := make([]*integration.AwsCustomNameSpaceSyncRule, 0, len(models))
	for _, model := range models {
		result = append(result, &integration.AwsCustomNameSpaceSyncRule{
			DefaultAction: integration.AwsSyncRuleFilterAction(model.DefaultAction.ValueString()),
			Filter:        namespaceFilter(model),
			Namespace:     model.Namespace.ValueString(),
		})
	}
	return result, diags
}

func namespaceRules(ctx context.Context, value types.Set) ([]*integration.AwsNameSpaceSyncRule, diag.Diagnostics) {
	models, diags := namespaceRuleModels(ctx, value)
	result := make([]*integration.AwsNameSpaceSyncRule, 0, len(models))
	for _, model := range models {
		result = append(result, &integration.AwsNameSpaceSyncRule{
			DefaultAction: integration.AwsSyncRuleFilterAction(model.DefaultAction.ValueString()),
			Filter:        namespaceFilter(model),
			Namespace:     integration.AwsService(model.Namespace.ValueString()),
		})
	}
	return result, diags
}

func namespaceRuleModels(ctx context.Context, value types.Set) ([]awsNamespaceSyncRuleModel, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var models []awsNamespaceSyncRuleModel
	diags := value.ElementsAs(ctx, &models, false)
	return models, diags
}

func namespaceFilter(model awsNamespaceSyncRuleModel) *integration.AwsSyncRuleFilter {
	if model.FilterAction.IsNull() || model.FilterAction.ValueString() == "" {
		return nil
	}
	return &integration.AwsSyncRuleFilter{
		Action: integration.AwsSyncRuleFilterAction(model.FilterAction.ValueString()),
		Source: model.FilterSource.ValueString(),
	}
}

func metricStats(ctx context.Context, value types.Set) (map[string]map[string][]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var models []awsMetricStatsModel
	diags := value.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	result := make(map[string]map[string][]string)
	for _, model := range models {
		stats, valueDiags := stringSetElements(ctx, model.Stats)
		diags.Append(valueDiags...)
		namespace := model.Namespace.ValueString()
		if result[namespace] == nil {
			result[namespace] = make(map[string][]string)
		}
		result[namespace][model.Metric.ValueString()] = stats
	}
	return result, diags
}

func (model *resourceAWSModel) updateFromAPI(ctx context.Context, details *integration.AwsCloudWatchIntegration, updateID bool) diag.Diagnostics {
	if details == nil {
		return nil
	}
	if updateID {
		model.ID = types.StringValue(details.Id)
	}
	model.Name = types.StringValue(details.Name)
	model.Enabled = types.BoolValue(details.Enabled)
	model.AuthMethod = types.StringValue(string(details.AuthMethod))
	model.EnableAWSUsage = types.BoolValue(details.EnableAwsUsage)
	model.ImportCloudWatch = types.BoolValue(details.ImportCloudWatch)
	model.UseMetricStreamsSync = types.BoolValue(details.MetricStreamsSyncState == awsMetricStreamsEnabled)
	model.EnableCheckLargeVolume = types.BoolValue(details.EnableCheckLargeVolume)
	model.SyncCustomNamespacesOnly = types.BoolValue(details.SyncCustomNamespacesOnly)
	model.CollectOnlyRecommendedStats = types.BoolValue(details.CollectOnlyRecommendedStats)
	model.MetricStreamsManagedExternally = types.BoolValue(details.MetricStreamsManagedExternally)
	if details.PollRate > 0 {
		model.PollRate = types.Int64Value(details.PollRate / 1000)
	}
	if details.InactiveMetricsPollRate > 0 {
		model.InactiveMetricsPollRate = types.Int64Value(details.InactiveMetricsPollRate / 1000)
	}
	updateOptionalString(&model.NamedToken, details.NamedToken)
	updateOptionalString(&model.Token, details.Token)
	updateOptionalString(&model.Key, details.Key)
	updateOptionalString(&model.ExternalID, details.ExternalId)
	updateOptionalString(&model.RoleARN, details.RoleArn)

	var diags diag.Diagnostics
	if len(details.Regions) > 0 {
		diags.Append(updateStringSet(ctx, &model.Regions, details.Regions, false)...)
	}
	if len(details.CustomNamespaceSyncRules) > 0 {
		value, valueDiags := customRulesValue(ctx, details.CustomNamespaceSyncRules)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.CustomNamespaceSyncRule = value
		}
	} else if details.CustomCloudWatchNamespaces != "" {
		diags.Append(updateStringSet(ctx, &model.CustomCloudWatchNamespaces, strings.Split(details.CustomCloudWatchNamespaces, ","), false)...)
	}
	if !model.Services.IsNull() && !model.Services.IsUnknown() {
		if len(details.Services) > 0 {
			services := make([]string, len(details.Services))
			for i, service := range details.Services {
				services[i] = string(service)
			}
			diags.Append(updateStringSet(ctx, &model.Services, services, false)...)
		}
	} else if len(details.NamespaceSyncRules) > 0 {
		value, valueDiags := rulesValue(ctx, details.NamespaceSyncRules)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.NamespaceSyncRule = value
		}
	}
	if len(details.MetricStatsToSync) > 0 {
		value, valueDiags := metricStatsValue(ctx, details.MetricStatsToSync)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.MetricStatsToSync = value
		}
	}
	return diags
}

func customRulesValue(ctx context.Context, rules []*integration.AwsCustomNameSpaceSyncRule) (types.Set, diag.Diagnostics) {
	models := make([]awsNamespaceSyncRuleModel, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		models = append(models, namespaceRuleModel(string(rule.DefaultAction), rule.Filter, rule.Namespace))
	}
	return types.SetValueFrom(ctx, awsNamespaceRuleObjectType, models)
}

func rulesValue(ctx context.Context, rules []*integration.AwsNameSpaceSyncRule) (types.Set, diag.Diagnostics) {
	models := make([]awsNamespaceSyncRuleModel, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		models = append(models, namespaceRuleModel(string(rule.DefaultAction), rule.Filter, string(rule.Namespace)))
	}
	return types.SetValueFrom(ctx, awsNamespaceRuleObjectType, models)
}

func namespaceRuleModel(defaultAction string, filter *integration.AwsSyncRuleFilter, namespace string) awsNamespaceSyncRuleModel {
	model := awsNamespaceSyncRuleModel{
		DefaultAction: optionalStringValue(defaultAction),
		FilterAction:  types.StringNull(),
		FilterSource:  types.StringNull(),
		Namespace:     types.StringValue(namespace),
	}
	if filter != nil {
		model.FilterAction = optionalStringValue(string(filter.Action))
		model.FilterSource = optionalStringValue(filter.Source)
	}
	return model
}

func metricStatsValue(ctx context.Context, values map[string]map[string][]string) (types.Set, diag.Diagnostics) {
	models := make([]awsMetricStatsModel, 0)
	var diags diag.Diagnostics
	for namespace, metrics := range values {
		for metric, stats := range metrics {
			statsValue, valueDiags := types.SetValueFrom(ctx, types.StringType, stats)
			diags.Append(valueDiags...)
			models = append(models, awsMetricStatsModel{
				Namespace: types.StringValue(namespace), Metric: types.StringValue(metric), Stats: statsValue,
			})
		}
	}
	if diags.HasError() {
		return types.SetNull(awsMetricStatsObjectType), diags
	}
	value, valueDiags := types.SetValueFrom(ctx, awsMetricStatsObjectType, models)
	diags.Append(valueDiags...)
	return value, diags
}
