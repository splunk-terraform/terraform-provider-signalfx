// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
)

const (
	gcpAuthServiceAccount = "SERVICE_ACCOUNT_KEY"
	gcpAuthWIF            = "WORKLOAD_IDENTITY_FEDERATION"
	gcpSyncSelected       = "SELECTED"
	gcpSyncAllReachable   = "ALL_REACHABLE"
)

type ResourceGCP struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceGCPModel struct {
	integrationModel
	PollRate                       types.Int64  `tfsdk:"poll_rate"`
	Services                       types.Set    `tfsdk:"services"`
	CustomMetricTypeDomains        types.Set    `tfsdk:"custom_metric_type_domains"`
	AuthMethod                     types.String `tfsdk:"auth_method"`
	ProjectServiceKeys             types.Set    `tfsdk:"project_service_keys"`
	ProjectWIFConfigs              types.Set    `tfsdk:"project_wif_configs"`
	WorkloadIdentityConfig         types.String `tfsdk:"workload_identity_federation_config"`
	Projects                       types.List   `tfsdk:"projects"`
	WIFSplunkIdentity              types.Map    `tfsdk:"wif_splunk_identity"`
	UseMetricSourceProjectForQuota types.Bool   `tfsdk:"use_metric_source_project_for_quota"`
	IncludeList                    types.Set    `tfsdk:"include_list"`
	NamedToken                     types.String `tfsdk:"named_token"`
	ImportGCPMetrics               types.Bool   `tfsdk:"import_gcp_metrics"`
	ExcludeGCEInstancesWithLabels  types.Set    `tfsdk:"exclude_gce_instances_with_labels"`
}

type gcpProjectServiceKeyModel struct {
	ProjectID  types.String `tfsdk:"project_id"`
	ProjectKey types.String `tfsdk:"project_key"`
}

type gcpProjectWIFConfigModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	WIFConfig types.String `tfsdk:"wif_config"`
}

type gcpProjectsModel struct {
	SyncMode           types.String `tfsdk:"sync_mode"`
	SelectedProjectIDs types.Set    `tfsdk:"selected_project_ids"`
}

var (
	_ resource.Resource                = (*ResourceGCP)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceGCP)(nil)
	_ resource.ResourceWithImportState = (*ResourceGCP)(nil)

	gcpProjectServiceKeyAttributeTypes = map[string]attr.Type{
		"project_id": types.StringType, "project_key": types.StringType,
	}
	gcpProjectServiceKeyObjectType = types.ObjectType{AttrTypes: gcpProjectServiceKeyAttributeTypes}
	gcpProjectWIFAttributeTypes    = map[string]attr.Type{
		"project_id": types.StringType, "wif_config": types.StringType,
	}
	gcpProjectWIFObjectType   = types.ObjectType{AttrTypes: gcpProjectWIFAttributeTypes}
	gcpProjectsAttributeTypes = map[string]attr.Type{
		"sync_mode": types.StringType, "selected_project_ids": types.SetType{ElemType: types.StringType},
	}
	gcpProjectsObjectType = types.ObjectType{AttrTypes: gcpProjectsAttributeTypes}
)

func NewResourceGCP() resource.Resource { return &ResourceGCP{} }

func (gcp *ResourceGCP) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gcp_integration"
}

func (gcp *ResourceGCP) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["poll_rate"] = schema.Int64Attribute{
		Optional: true, Computed: true, Default: int64default.StaticInt64(300),
		Description: "GCP polling interval in seconds. Must be between 60 and 600. Defaults to 300.",
		Validators:  []validator.Int64{int64validator.Between(60, 600)},
	}
	attributes["services"] = schema.SetAttribute{
		Optional: true, ElementType: types.StringType,
		Description: "GCP services whose metrics are imported. Omit or use an empty set to import all supported services.",
	}
	attributes["custom_metric_type_domains"] = schema.SetAttribute{
		Optional: true, ElementType: types.StringType,
		Description: "Additional GCP service domains to monitor for custom metric types.",
	}
	attributes["auth_method"] = schema.StringAttribute{
		Optional: true, Computed: true,
		Description: "Authentication method. Allowed values are `SERVICE_ACCOUNT_KEY` and `WORKLOAD_IDENTITY_FEDERATION`; the service defaults to `SERVICE_ACCOUNT_KEY`.",
		Validators: []validator.String{
			stringvalidator.OneOfCaseInsensitive(gcpAuthServiceAccount, gcpAuthWIF),
		},
	}
	attributes["workload_identity_federation_config"] = schema.StringAttribute{
		Optional:    true,
		Description: "Workload Identity Federation configuration JSON.",
		Validators: []validator.String{stringvalidator.ConflictsWith(
			path.MatchRoot("project_service_keys"), path.MatchRoot("project_wif_configs"),
		)},
	}
	attributes["wif_splunk_identity"] = schema.MapAttribute{
		Optional: true, Computed: true, ElementType: types.StringType,
		Description: "Splunk Observability Cloud GCP identity to include in the GCP WIF provider definition.",
	}
	attributes["use_metric_source_project_for_quota"] = schema.BoolAttribute{
		Optional: true, Computed: true, Default: booldefault.StaticBool(false),
		Description: "Whether to consume quota from the metric source project. Defaults to false.",
	}
	attributes["include_list"] = schema.SetAttribute{
		Optional: true, ElementType: types.StringType,
		Description: "Custom metadata keys to collect for Compute Engine instances.",
	}
	attributes["named_token"] = schema.StringAttribute{
		Optional:    true,
		Description: "Named organization token used for data ingestion. Changing this value replaces the integration.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attributes["import_gcp_metrics"] = schema.BoolAttribute{
		Optional: true, Computed: true, Default: booldefault.StaticBool(true),
		Description: "Whether to import Google Cloud Monitoring metrics in addition to metadata. Defaults to true.",
	}
	attributes["exclude_gce_instances_with_labels"] = schema.SetAttribute{
		Optional: true, ElementType: types.StringType,
		Description: "Label keys that exclude Compute Engine instances from metric synchronization.",
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Google Cloud Platform integration in Splunk Observability Cloud.",
		Attributes:  attributes,
		Blocks: map[string]schema.Block{
			"project_service_keys": schema.SetNestedBlock{
				Description: "GCP projects authenticated with service account keys.",
				Validators: []validator.Set{setvalidator.ConflictsWith(
					path.MatchRoot("project_wif_configs"), path.MatchRoot("workload_identity_federation_config"),
				)},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"project_id": schema.StringAttribute{
						Required: true, Sensitive: true, Description: "GCP project ID.",
					},
					"project_key": schema.StringAttribute{
						Required: true, Sensitive: true, Description: "Google service account key JSON for the project.",
					},
				}},
			},
			"project_wif_configs": schema.SetNestedBlock{
				Description:        "Deprecated per-project GCP Workload Identity Federation configurations.",
				DeprecationMessage: "Use workload_identity_federation_config with projects instead.",
				Validators: []validator.Set{setvalidator.ConflictsWith(
					path.MatchRoot("project_service_keys"), path.MatchRoot("workload_identity_federation_config"),
				)},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"project_id": schema.StringAttribute{Required: true, Description: "GCP project ID."},
					"wif_config": schema.StringAttribute{Required: true, Description: "Workload Identity Federation configuration JSON for the project."},
				}},
			},
			"projects": schema.ListNestedBlock{
				Description: "Project discovery and selection configuration for Workload Identity Federation.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"sync_mode": schema.StringAttribute{
						Optional: true, Computed: true, Default: stringdefault.StaticString(gcpSyncSelected),
						Description: "Project synchronization mode. Allowed values are `SELECTED` and `ALL_REACHABLE`. Defaults to `SELECTED`.",
						Validators:  []validator.String{stringvalidator.OneOf(gcpSyncSelected, gcpSyncAllReachable)},
					},
					"selected_project_ids": schema.SetAttribute{
						Optional: true, ElementType: types.StringType,
						Description: "Project IDs to synchronize when sync_mode is `SELECTED`.",
					},
				}},
			},
		},
	}
}

func (gcp *ResourceGCP) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceGCPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := model.gcpIntegration(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := gcp.Details().Client.CreateGCPIntegration(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (gcp *ResourceGCP) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceGCPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := gcp.Details().Client.GetGCPIntegration(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (gcp *ResourceGCP) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceGCPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := model.gcpIntegration(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := gcp.Details().Client.UpdateGCPIntegration(ctx, model.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (gcp *ResourceGCP) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceGCPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(
		ctx, resp.State, gcp.Details().Client.DeleteGCPIntegration(ctx, model.ID.ValueString()),
	)...)
}

func (model resourceGCPModel) gcpIntegration(ctx context.Context) (*integration.GCPIntegration, diag.Diagnostics) {
	importGCPMetrics := model.ImportGCPMetrics.ValueBool()
	payload := &integration.GCPIntegration{
		Type:                             integration.Type("GCP"),
		Name:                             model.Name.ValueString(),
		Enabled:                          model.Enabled.ValueBool(),
		PollRateMs:                       model.PollRate.ValueInt64() * 1000,
		AuthMethod:                       integration.GCPAuthMethod(strings.ToUpper(model.AuthMethod.ValueString())),
		WorkloadIdentityFederationConfig: model.WorkloadIdentityConfig.ValueString(),
		UseMetricSourceProjectForQuota:   model.UseMetricSourceProjectForQuota.ValueBool(),
		NamedToken:                       model.NamedToken.ValueString(),
		ImportGCPMetrics:                 &importGCPMetrics,
	}

	var diags diag.Diagnostics
	services, valueDiags := stringSetElements(ctx, model.Services)
	diags.Append(valueDiags...)
	for _, service := range services {
		payload.Services = append(payload.Services, integration.GcpService(service))
	}
	payload.CustomMetricTypeDomains, valueDiags = stringSetElements(ctx, model.CustomMetricTypeDomains)
	diags.Append(valueDiags...)
	payload.IncludeList, valueDiags = stringSetElements(ctx, model.IncludeList)
	diags.Append(valueDiags...)
	payload.ExcludeGCEInstancesWithLabels, valueDiags = stringSetElements(ctx, model.ExcludeGCEInstancesWithLabels)
	diags.Append(valueDiags...)

	if !model.ProjectServiceKeys.IsNull() && !model.ProjectServiceKeys.IsUnknown() {
		var configured []gcpProjectServiceKeyModel
		diags.Append(model.ProjectServiceKeys.ElementsAs(ctx, &configured, false)...)
		for _, item := range configured {
			payload.ProjectServiceKeys = append(payload.ProjectServiceKeys, &integration.GCPProject{
				ProjectId: item.ProjectID.ValueString(), ProjectKey: item.ProjectKey.ValueString(),
			})
		}
	}
	if !model.ProjectWIFConfigs.IsNull() && !model.ProjectWIFConfigs.IsUnknown() {
		var configured []gcpProjectWIFConfigModel
		diags.Append(model.ProjectWIFConfigs.ElementsAs(ctx, &configured, false)...)
		for _, item := range configured {
			payload.WifConfigs = append(payload.WifConfigs, &integration.GCPProjectWIFConfig{
				ProjectId: item.ProjectID.ValueString(), WIFConfig: item.WIFConfig.ValueString(),
			})
		}
	}
	if !model.Projects.IsNull() && !model.Projects.IsUnknown() {
		var configured []gcpProjectsModel
		diags.Append(model.Projects.ElementsAs(ctx, &configured, false)...)
		if len(configured) > 0 {
			selected, selectedDiags := stringSetElements(ctx, configured[0].SelectedProjectIDs)
			diags.Append(selectedDiags...)
			payload.Projects = &integration.GCPProjects{
				SyncMode: integration.SyncMode(configured[0].SyncMode.ValueString()), SelectedProjectIds: selected,
			}
		}
	}
	if diags.HasError() {
		return nil, diags
	}
	return payload, diags
}

func (model *resourceGCPModel) updateFromAPI(ctx context.Context, details *integration.GCPIntegration, updateID bool) diag.Diagnostics {
	if details == nil {
		return nil
	}
	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	if details.PollRateMs > 0 {
		model.PollRate = types.Int64Value(details.PollRateMs / 1000)
	}
	updateOptionalString(&model.NamedToken, details.NamedToken)
	model.UseMetricSourceProjectForQuota = types.BoolValue(details.UseMetricSourceProjectForQuota)
	if details.ImportGCPMetrics != nil {
		model.ImportGCPMetrics = types.BoolValue(*details.ImportGCPMetrics)
	}
	if details.AuthMethod != "" {
		apiAuthMethod := string(details.AuthMethod)
		if model.AuthMethod.IsNull() || model.AuthMethod.IsUnknown() ||
			!strings.EqualFold(model.AuthMethod.ValueString(), apiAuthMethod) {
			model.AuthMethod = types.StringValue(apiAuthMethod)
		}
	} else if model.AuthMethod.IsUnknown() {
		model.AuthMethod = types.StringNull()
	}

	var diags diag.Diagnostics
	if len(details.Services) > 0 {
		services := make([]string, len(details.Services))
		for i, service := range details.Services {
			services[i] = string(service)
		}
		diags.Append(updateStringSet(ctx, &model.Services, services, false)...)
	}
	diags.Append(updateStringSet(ctx, &model.CustomMetricTypeDomains, details.CustomMetricTypeDomains, true)...)
	diags.Append(updateStringSet(ctx, &model.IncludeList, details.IncludeList, true)...)
	diags.Append(updateStringSet(ctx, &model.ExcludeGCEInstancesWithLabels, details.ExcludeGCEInstancesWithLabels, true)...)

	if len(details.WifConfigs) > 0 {
		configured := make([]gcpProjectWIFConfigModel, 0, len(details.WifConfigs))
		for _, item := range details.WifConfigs {
			if item == nil {
				continue
			}
			configured = append(configured, gcpProjectWIFConfigModel{
				ProjectID: types.StringValue(item.ProjectId), WIFConfig: types.StringValue(item.WIFConfig),
			})
		}
		value, valueDiags := types.SetValueFrom(ctx, gcpProjectWIFObjectType, configured)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.ProjectWIFConfigs = value
		}
	} else if !model.ProjectWIFConfigs.IsNull() && !model.ProjectWIFConfigs.IsUnknown() {
		model.ProjectWIFConfigs = types.SetValueMust(gcpProjectWIFObjectType, nil)
	}

	if details.WorkloadIdentityFederationConfig != "" {
		model.WorkloadIdentityConfig = types.StringValue(details.WorkloadIdentityFederationConfig)
	} else if !model.WorkloadIdentityConfig.IsNull() && !model.WorkloadIdentityConfig.IsUnknown() {
		model.WorkloadIdentityConfig = types.StringNull()
	}
	if details.Projects != nil {
		selected, selectedDiags := types.SetValueFrom(ctx, types.StringType, details.Projects.SelectedProjectIds)
		diags.Append(selectedDiags...)
		configured := []gcpProjectsModel{{
			SyncMode: types.StringValue(string(details.Projects.SyncMode)), SelectedProjectIDs: selected,
		}}
		value, valueDiags := types.ListValueFrom(ctx, gcpProjectsObjectType, configured)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.Projects = value
		}
	}
	if details.WifSplunkIdentity != nil {
		value, valueDiags := types.MapValueFrom(ctx, types.StringType, details.WifSplunkIdentity)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.WIFSplunkIdentity = value
		}
	} else {
		model.WIFSplunkIdentity = types.MapNull(types.StringType)
	}
	return diags
}

func stringSetElements(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var result []string
	diags := value.ElementsAs(ctx, &result, false)
	return result, diags
}

func updateStringSet(ctx context.Context, target *types.Set, values []string, clearKnown bool) diag.Diagnostics {
	if len(values) == 0 {
		if clearKnown && !target.IsNull() && !target.IsUnknown() {
			*target = types.SetValueMust(types.StringType, []attr.Value{})
		}
		return nil
	}
	value, diags := types.SetValueFrom(ctx, types.StringType, values)
	if !diags.HasError() {
		*target = value
	}
	return diags
}
