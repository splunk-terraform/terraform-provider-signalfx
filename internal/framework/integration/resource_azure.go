// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	azureEnvironmentDefault      = "azure"
	azureEnvironmentUSGovernment = "azure_us_government"
)

type ResourceAzure struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceAzureModel struct {
	integrationModel
	Environment                types.String `tfsdk:"environment"`
	AppID                      types.String `tfsdk:"app_id"`
	CustomNamespacesPerService types.Set    `tfsdk:"custom_namespaces_per_service"`
	SecretKey                  types.String `tfsdk:"secret_key"`
	PollRate                   types.Int64  `tfsdk:"poll_rate"`
	Services                   types.Set    `tfsdk:"services"`
	AdditionalServices         types.List   `tfsdk:"additional_services"`
	ResourceFilterRules        types.List   `tfsdk:"resource_filter_rules"`
	Subscriptions              types.Set    `tfsdk:"subscriptions"`
	SyncGuestOSNamespaces      types.Bool   `tfsdk:"sync_guest_os_namespaces"`
	ImportAzureMonitor         types.Bool   `tfsdk:"import_azure_monitor"`
	TenantID                   types.String `tfsdk:"tenant_id"`
	NamedToken                 types.String `tfsdk:"named_token"`
	UseBatchAPI                types.Bool   `tfsdk:"use_batch_api"`
}

type azureCustomNamespaceModel struct {
	Service    types.String `tfsdk:"service"`
	Namespaces types.Set    `tfsdk:"namespaces"`
}

type azureResourceFilterRuleModel struct {
	FilterSource types.String `tfsdk:"filter_source"`
}

var (
	_ resource.Resource                = (*ResourceAzure)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceAzure)(nil)
	_ resource.ResourceWithImportState = (*ResourceAzure)(nil)

	azureCustomNamespaceAttributeTypes = map[string]attr.Type{
		"service":    types.StringType,
		"namespaces": types.SetType{ElemType: types.StringType},
	}
	azureCustomNamespaceObjectType = types.ObjectType{AttrTypes: azureCustomNamespaceAttributeTypes}
	azureFilterRuleAttributeTypes  = map[string]attr.Type{"filter_source": types.StringType}
	azureFilterRuleObjectType      = types.ObjectType{AttrTypes: azureFilterRuleAttributeTypes}
)

func NewResourceAzure() resource.Resource { return &ResourceAzure{} }

func (azure *ResourceAzure) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azure_integration"
}

func (azure *ResourceAzure) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["environment"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Sensitive:   true,
		Description: "Azure environment. Allowed values are `azure` and `azure_us_government`. Defaults to `azure`.",
		Default:     stringdefault.StaticString(azureEnvironmentDefault),
		Validators: []validator.String{
			stringvalidator.OneOfCaseInsensitive(azureEnvironmentDefault, azureEnvironmentUSGovernment),
		},
	}
	attributes["app_id"] = schema.StringAttribute{
		Required:    true,
		Sensitive:   true,
		Description: "Azure application ID for the Splunk Observability Cloud application.",
	}
	attributes["secret_key"] = schema.StringAttribute{
		Required:    true,
		Sensitive:   true,
		Description: "Secret key that associates the Splunk Observability Cloud application with the Azure tenant.",
	}
	attributes["poll_rate"] = schema.Int64Attribute{
		Optional:    true,
		Computed:    true,
		Description: "Azure polling interval in seconds. Must be between 60 and 600. Defaults to 300.",
		Default:     int64default.StaticInt64(300),
		Validators:  []validator.Int64{int64validator.Between(60, 600)},
	}
	attributes["services"] = schema.SetAttribute{
		Required:    true,
		ElementType: types.StringType,
		Description: "One or more Microsoft Azure service names to monitor.",
		Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
	}
	attributes["additional_services"] = schema.ListAttribute{
		Optional:    true,
		ElementType: types.StringType,
		Description: "Additional Azure resource types to synchronize.",
	}
	attributes["subscriptions"] = schema.SetAttribute{
		Required:    true,
		ElementType: types.StringType,
		Description: "Azure subscriptions to monitor.",
	}
	attributes["sync_guest_os_namespaces"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Whether to synchronize recommended guest operating-system metric namespaces for virtual machines. Defaults to false.",
		Default:     booldefault.StaticBool(false),
	}
	attributes["import_azure_monitor"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Whether to synchronize Azure Monitor metrics in addition to metadata. Defaults to true.",
		Default:     booldefault.StaticBool(true),
	}
	attributes["tenant_id"] = schema.StringAttribute{
		Required:    true,
		Description: "Azure tenant ID.",
	}
	attributes["named_token"] = schema.StringAttribute{
		Optional:    true,
		Description: "Named organization token used for data ingestion. Changing this value replaces the integration.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attributes["use_batch_api"] = schema.BoolAttribute{
		Optional:    true,
		Description: "Whether to collect datapoints with the paid Azure Metrics Batch API.",
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Microsoft Azure integration in Splunk Observability Cloud.",
		Attributes:  attributes,
		Blocks: map[string]schema.Block{
			"custom_namespaces_per_service": schema.SetNestedBlock{
				Description: "Additional metric namespaces to synchronize for individual Azure services.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"service": schema.StringAttribute{Required: true, Description: "Azure service name."},
					"namespaces": schema.SetAttribute{
						Required: true, ElementType: types.StringType, Description: "Additional namespaces to synchronize.",
					},
				}},
			},
			"resource_filter_rules": schema.ListNestedBlock{
				Description: "Rules that filter Azure resources by tags.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"filter_source": schema.StringAttribute{
						Required:    true,
						Description: "SignalFlow filter expression whose referenced tag keys begin with `azure_tag_`.",
					},
				}},
			},
		},
	}
}

func (azure *ResourceAzure) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceAzureModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := model.azureIntegration(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := azure.Details().Client.CreateAzureIntegration(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(ctx, details, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (azure *ResourceAzure) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceAzureModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := azure.Details().Client.GetAzureIntegration(ctx, model.ID.ValueString())
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

func (azure *ResourceAzure) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceAzureModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := model.azureIntegration(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := azure.Details().Client.UpdateAzureIntegration(ctx, model.ID.ValueString(), payload)
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

func (azure *ResourceAzure) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceAzureModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(
		ctx, resp.State, azure.Details().Client.DeleteAzureIntegration(ctx, model.ID.ValueString()),
	)...)
}

func (model resourceAzureModel) azureIntegration(ctx context.Context) (*integration.AzureIntegration, diag.Diagnostics) {
	importAzureMonitor := model.ImportAzureMonitor.ValueBool()
	useBatchAPI := model.UseBatchAPI.ValueBool()
	payload := &integration.AzureIntegration{
		Type:                  integration.Type("Azure"),
		Name:                  model.Name.ValueString(),
		Enabled:               model.Enabled.ValueBool(),
		AppId:                 model.AppID.ValueString(),
		AzureEnvironment:      integration.AzureEnvironment(strings.ToUpper(model.Environment.ValueString())),
		SecretKey:             model.SecretKey.ValueString(),
		PollRateMs:            model.PollRate.ValueInt64() * 1000,
		TenantId:              model.TenantID.ValueString(),
		NamedToken:            model.NamedToken.ValueString(),
		SyncGuestOsNamespaces: model.SyncGuestOSNamespaces.ValueBool(),
		ImportAzureMonitor:    &importAzureMonitor,
		UseBatchApi:           &useBatchAPI,
	}

	var diags diag.Diagnostics
	var services []string
	diags.Append(model.Services.ElementsAs(ctx, &services, false)...)
	for _, service := range services {
		payload.Services = append(payload.Services, integration.AzureService(service))
	}
	if !model.AdditionalServices.IsNull() && !model.AdditionalServices.IsUnknown() {
		diags.Append(model.AdditionalServices.ElementsAs(ctx, &payload.AdditionalServices, false)...)
	}
	if !model.Subscriptions.IsNull() && !model.Subscriptions.IsUnknown() {
		diags.Append(model.Subscriptions.ElementsAs(ctx, &payload.Subscriptions, false)...)
	}
	if !model.CustomNamespacesPerService.IsNull() && !model.CustomNamespacesPerService.IsUnknown() {
		var configured []azureCustomNamespaceModel
		diags.Append(model.CustomNamespacesPerService.ElementsAs(ctx, &configured, false)...)
		if len(configured) > 0 {
			payload.CustomNamespacesPerService = make(map[string][]string, len(configured))
		}
		for _, item := range configured {
			var namespaces []string
			diags.Append(item.Namespaces.ElementsAs(ctx, &namespaces, false)...)
			payload.CustomNamespacesPerService[item.Service.ValueString()] = namespaces
		}
	}
	if !model.ResourceFilterRules.IsNull() && !model.ResourceFilterRules.IsUnknown() {
		var configured []azureResourceFilterRuleModel
		diags.Append(model.ResourceFilterRules.ElementsAs(ctx, &configured, false)...)
		for _, item := range configured {
			payload.ResourceFilterRules = append(payload.ResourceFilterRules, integration.AzureFilterRule{
				Filter: integration.AzureFilterExpression{Source: item.FilterSource.ValueString()},
			})
		}
	}
	if diags.HasError() {
		return nil, diags
	}
	return payload, diags
}

func (model *resourceAzureModel) updateFromAPI(ctx context.Context, details *integration.AzureIntegration, updateID bool) diag.Diagnostics {
	if details == nil {
		return nil
	}
	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	updateOptionalString(&model.AppID, details.AppId)
	updateOptionalString(&model.SecretKey, details.SecretKey)
	updateOptionalString(&model.TenantID, details.TenantId)
	updateOptionalString(&model.NamedToken, details.NamedToken)
	if details.AzureEnvironment != "" {
		apiEnvironment := strings.ToLower(string(details.AzureEnvironment))
		if model.Environment.IsNull() || model.Environment.IsUnknown() ||
			!strings.EqualFold(model.Environment.ValueString(), apiEnvironment) {
			model.Environment = types.StringValue(apiEnvironment)
		}
	}
	if details.PollRateMs > 0 {
		model.PollRate = types.Int64Value(details.PollRateMs / 1000)
	}
	model.SyncGuestOSNamespaces = types.BoolValue(details.SyncGuestOsNamespaces)
	if details.ImportAzureMonitor != nil {
		model.ImportAzureMonitor = types.BoolValue(*details.ImportAzureMonitor)
	}
	if details.UseBatchApi != nil && (!model.UseBatchAPI.IsNull() || *details.UseBatchApi) {
		model.UseBatchAPI = types.BoolValue(*details.UseBatchApi)
	}

	var diags diag.Diagnostics
	if len(details.Services) > 0 {
		services := make([]string, len(details.Services))
		for i, service := range details.Services {
			services[i] = string(service)
		}
		value, valueDiags := types.SetValueFrom(ctx, types.StringType, services)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.Services = value
		}
	}
	if len(details.AdditionalServices) > 0 {
		value, valueDiags := types.ListValueFrom(ctx, types.StringType, details.AdditionalServices)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.AdditionalServices = value
		}
	}
	if len(details.Subscriptions) > 0 {
		value, valueDiags := types.SetValueFrom(ctx, types.StringType, details.Subscriptions)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.Subscriptions = value
		}
	}
	if len(details.CustomNamespacesPerService) > 0 {
		keys := make([]string, 0, len(details.CustomNamespacesPerService))
		for service := range details.CustomNamespacesPerService {
			keys = append(keys, service)
		}
		sort.Strings(keys)
		configured := make([]azureCustomNamespaceModel, 0, len(keys))
		for _, service := range keys {
			namespaces, namespaceDiags := types.SetValueFrom(ctx, types.StringType, details.CustomNamespacesPerService[service])
			diags.Append(namespaceDiags...)
			configured = append(configured, azureCustomNamespaceModel{Service: types.StringValue(service), Namespaces: namespaces})
		}
		value, valueDiags := types.SetValueFrom(ctx, azureCustomNamespaceObjectType, configured)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.CustomNamespacesPerService = value
		}
	}
	if len(details.ResourceFilterRules) > 0 {
		configured := make([]azureResourceFilterRuleModel, len(details.ResourceFilterRules))
		for i, rule := range details.ResourceFilterRules {
			configured[i] = azureResourceFilterRuleModel{FilterSource: types.StringValue(rule.Filter.Source)}
		}
		value, valueDiags := types.ListValueFrom(ctx, azureFilterRuleObjectType, configured)
		diags.Append(valueDiags...)
		if !valueDiags.HasError() {
			model.ResourceFilterRules = value
		}
	}
	return diags
}
