// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdetector

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/notification"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental"
	flow "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/signalflow"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

type AutoDetectorResource struct {
	fwembed.ResourceIDImporter
	fwembed.ResourceData
}

type AutoDetectorResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ParentID      types.String `tfsdk:"parent_id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Severity      types.String `tfsdk:"severity"`
	Tags          types.List   `tfsdk:"tags"`
	Teams         types.List   `tfsdk:"teams"`
	Notifications types.List   `tfsdk:"notifications"`
	Filters       types.List   `tfsdk:"filters"`
	Inputs        types.Map    `tfsdk:"inputs"`
}

type autoDetectorFilterModel struct {
	Key    types.String `tfsdk:"key"`
	Values types.List   `tfsdk:"values"`
}

var (
	_ resource.Resource                   = (*AutoDetectorResource)(nil)
	_ resource.ResourceWithConfigure      = (*AutoDetectorResource)(nil)
	_ resource.ResourceWithImportState    = (*AutoDetectorResource)(nil)
	_ resource.ResourceWithValidateConfig = (*AutoDetectorResource)(nil)
)

func NewAutoDetectorResource() resource.Resource {
	return &AutoDetectorResource{}
}

func (r *AutoDetectorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_customized_auto_detector"
}

func (r *AutoDetectorResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This resources allows for users to customize existing auto detectors by providing various inputs.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"parent_id": schema.StringAttribute{
				Description: "This is the id of the auto detector we are making a customisation of them",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the customized auto detector.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the customized auto detector.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"severity": schema.StringAttribute{
				Description: "The severity of the customized auto detector. " +
					"By default, the severity will be the same as the original auto detector.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("Critical", "Major", "Minor", "Warning", "Info"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.ListAttribute{
				Description: "The list of tags to be added to the customized auto detector. " +
					"By default, there will be no additional tags added to the original auto detector.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"teams": schema.ListAttribute{
				Description: "The list of teams to be added to the customized auto detector. " +
					"By default, there will be no additional teams added to the original auto detector.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"notifications": schema.ListNestedAttribute{
				Description: "The list of notifications to be added to the customized auto detector. " +
					"By default, there will be no additional notifications added to the original auto detector.",
				Optional:     true,
				Computed:     true,
				NestedObject: fwshared.NewNotificationResourceAttribute(),
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"inputs": schema.MapAttribute{
				Description: "The map of inputs to be added to the customized auto detector. " +
					"What inputs are required depends on the inherited auto detector.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"filters": schema.ListNestedAttribute{
				Description: "The list of filters to be added to the customized auto detector. " +
					"By default, there will be no additional filters added to the original auto detector.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The key of the filter.",
						},
						"values": schema.ListAttribute{
							Required:    true,
							Description: "The values of the filter.",
							ElementType: types.StringType,
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *AutoDetectorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model AutoDetectorResourceModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}

	client, err := pmeta.LoadClient(ctx, r.Details())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Empty(), "Unable to load client", err.Error())
		return
	}

	parent, err := client.GetDetector(ctx, model.ParentID.ValueString())
	if err != nil {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
		return
	}

	request, diag := r.buildRequest(ctx, parent, &model)
	if resp.Diagnostics.Append(diag...); resp.Diagnostics.HasError() {
		return
	}

	created, err := client.CreateDetector(ctx, request)
	if err != nil {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
		return
	}

	if resp.Diagnostics.Append(r.syncModelFromDetector(ctx, &model, created)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *AutoDetectorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model AutoDetectorResourceModel
	if resp.Diagnostics.Append(req.State.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}

	client, err := pmeta.LoadClient(ctx, r.Details())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Empty(), "Unable to load client", err.Error())
		return
	}

	current, err := client.GetDetector(ctx, model.ID.ValueString())
	if err != nil {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
		return
	}

	if resp.Diagnostics.Append(r.syncModelFromDetector(ctx, &model, current)...); resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *AutoDetectorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model AutoDetectorResourceModel
	if resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}

	client, err := pmeta.LoadClient(ctx, r.Details())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Empty(), "Unable to load client", err.Error())
		return
	}

	parent, err := client.GetDetector(ctx, model.ParentID.ValueString())
	if err != nil {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
		return
	}

	request, diag := r.buildRequest(ctx, parent, &model)
	if resp.Diagnostics.Append(diag...); resp.Diagnostics.HasError() {
		return
	}

	updated, err := client.UpdateDetector(ctx, model.ID.ValueString(), request)
	if err != nil {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
		return
	}

	if resp.Diagnostics.Append(r.syncModelFromDetector(ctx, &model, updated)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *AutoDetectorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model AutoDetectorResourceModel
	if resp.Diagnostics.Append(req.State.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}

	client, err := pmeta.LoadClient(ctx, r.Details())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Empty(), "Unable to load client", err.Error())
		return
	}

	if err := client.DeleteDetector(ctx, model.ID.ValueString()); err != nil {
		resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
	}
}

func (adr *AutoDetectorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if adr.Details() == nil {
		return
	}

	var model AutoDetectorResourceModel
	if resp.Diagnostics.Append(req.Config.Get(ctx, &model)...); resp.Diagnostics.HasError() {
		return
	}

	if model.ParentID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("parent_id"),
			"Invalid Parent ID",
			"The parent_id attribute must be provided and cannot be unknown.")
		return
	}

	client, err := pmeta.LoadClient(ctx, adr.Details())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Empty(), "Unable to load client", err.Error())
		return
	}

	autodetect, err := client.GetDetector(ctx, model.ParentID.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Empty(), "Unable to load auto detector", err.Error())
		return
	}

	if autodetect.DetectorOrigin != "AutoDetect" {
		resp.Diagnostics.AddAttributeError(
			path.Root("parent_id"),
			"Invalid Parent ID",
			"The provided parent_id does not belong to an auto detector.",
		)
		return
	}

	if model.Inputs.IsNull() || model.Inputs.IsUnknown() {
		return
	}

	var (
		u, _      = url.Parse(adr.Details().APIURL)
		inspector = experimental.NewInspector(u, adr.Details().AuthToken)
		allowed   = make(map[string]struct{})
	)

	values, _, err := inspector.GetAutoDetectorArgumentsAndFilters(ctx, autodetect.ProgramText)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Empty(), "Unable to load auto detector input details", err.Error())
		return
	}
	for _, value := range values {
		allowed[value.Name] = struct{}{}
	}

	for field := range model.Inputs.Elements() {
		if _, ok := allowed[field]; !ok {
			resp.Diagnostics.AddAttributeError(path.Root("inputs").AtMapKey(field), "Invalid Input", "The provided input is not supported by this auto detector.")
		}
	}
}

func (r *AutoDetectorResource) buildRequest(ctx context.Context, parent *detector.Detector, model *AutoDetectorResourceModel) (*detector.CreateUpdateDetectorRequest, diag.Diagnostics) {
	request := &detector.CreateUpdateDetectorRequest{
		Name:                 parent.Name,
		Description:          parent.Description,
		VisualizationOptions: parent.VisualizationOptions,
		Rules:                parent.Rules,
		ParentDetectorId:     model.ParentID.ValueString(),
		DetectorOrigin:       "AutoDetectCustomization",
		PackageSpecification: parent.PackageSpecification,
	}

	if !model.Name.IsNull() && !model.Name.IsUnknown() {
		request.Name = model.Name.ValueString()
	} else {
		request.Name += " (Terraform Customized)"
	}

	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		request.Description = model.Description.ValueString()
	}

	if !model.Severity.IsNull() && !model.Severity.IsUnknown() {
		severity := detector.Severity(model.Severity.ValueString())
		for _, rule := range request.Rules {
			rule.Severity = severity
		}
	}

	diags := model.Tags.ElementsAs(ctx, &request.Tags, false)
	request.Tags = common.Unique(pmeta.LoadProviderTags(ctx, r.Details()), request.Tags)

	diags.Append(model.Teams.ElementsAs(ctx, &request.Teams, false)...)
	request.Teams = pmeta.MergeProviderTeams(ctx, r.Details(), request.Teams)

	var notificationModels []fwshared.NotificationModel
	if !model.Notifications.IsNull() && !model.Notifications.IsUnknown() {
		diags.Append(model.Notifications.ElementsAs(ctx, &notificationModels, false)...)
	}

	if diags.HasError() {
		return nil, diags
	}

	if len(notificationModels) > 0 {
		notifications := make([]*notification.Notification, 0, len(notificationModels))
		for _, model := range notificationModels {
			notifications = append(notifications, model.Notification())
		}
		for _, rule := range request.Rules {
			rule.Notifications = notifications
		}
	}

	var (
		u, _ = url.Parse(r.Details().APIURL)
		vb   = experimental.NewProgramBuilderVisitor()
	)

	if results, filters, err := experimental.NewInspector(u, r.Details().AuthToken).GetAutoDetectorArgumentsAndFilters(ctx, parent.ProgramText); err == nil {
		for _, r := range results {
			if r.Type == "filter" {
				vb.WithFilterKey(r.Name)
			}
		}
		for key, values := range filters {
			vb.WithFilter(key, values...)
		}
	} else {
		tflog.Warn(ctx, "Unable to load results from server, filter key may be wrong", tfext.ErrorLogFields(err))
	}

	graph, err := flow.NewClient(u, r.Details().AuthToken).GetExecutionGraph(ctx, parent.ProgramText)
	if err != nil {
		diags.AddAttributeError(path.Empty(), "Unable to get execution graph for the parent auto detector", err.Error())
		return nil, diags
	}

	var filters []autoDetectorFilterModel
	if !model.Filters.IsNull() && !model.Filters.IsUnknown() {
		diags.Append(model.Filters.ElementsAs(ctx, &filters, false)...)
	}
	for _, filter := range filters {
		var values []string
		diags.Append(filter.Values.ElementsAs(ctx, &values, false)...)
		vb.WithFilter(filter.Key.ValueString(), values...)
	}

	inputs := make(map[string]string)
	if !model.Inputs.IsNull() && !model.Inputs.IsUnknown() {
		diags.Append(model.Inputs.ElementsAs(ctx, &inputs, false)...)
	}
	if diags.HasError() {
		return nil, diags
	}
	for key, value := range inputs {
		vb.WithInput(key, value)
	}

	if err := graph.Visit(vb); err != nil {
		diags.AddAttributeError(path.Empty(), "Unable to build program for the customized auto detector", err.Error())
		return nil, diags
	}

	request.ProgramText = vb.BuildProgramText()
	return request, nil
}

func (r *AutoDetectorResource) syncModelFromDetector(ctx context.Context, model *AutoDetectorResourceModel, current *detector.Detector) (diags diag.Diagnostics) {
	if current.Id != "" {
		model.ID = types.StringValue(current.Id)
	}

	if current.ParentDetectorId != "" {
		model.ParentID = types.StringValue(current.ParentDetectorId)
	}

	if current.Name != "" {
		model.Name = types.StringValue(current.Name)
	}

	if current.Description != "" {
		model.Description = types.StringValue(current.Description)
	}

	for _, r := range current.Rules {
		model.Severity = types.StringValue(string(r.Severity))
	}

	tags := append([]string{}, current.Tags...)
	if values, d := types.ListValueFrom(ctx, types.StringType, tags); d.HasError() {
		diags.Append(d...)
	} else {
		model.Tags = values
	}

	teams := append([]string{}, current.Teams...)
	if values, d := types.ListValueFrom(ctx, types.StringType, teams); d.HasError() {
		diags.Append(d...)
	} else {
		model.Teams = values
	}

	var notificationModels []fwshared.NotificationModel
	for _, r := range current.Rules {
		for _, n := range r.Notifications {
			notificationModels = append(notificationModels, fwshared.NewNotificationModelFromAPI(n))
		}
	}

	if values, d := types.ListValueFrom(ctx, fwshared.NewNotificationObjectType(), notificationModels); d.HasError() {
		diags.Append(d...)
	} else {
		model.Notifications = values
	}

	var (
		u, _      = url.Parse(r.Details().APIURL)
		inputs    = make(map[string]string)
		inspector = experimental.NewInspector(u, r.Details().AuthToken)
	)

	arguments, filters, err := inspector.GetAutoDetectorArgumentsAndFilters(ctx, current.ProgramText)
	if err != nil {
		diags.AddError("Unable to load auto detector input details", err.Error())
		return diags
	}

	for _, r := range arguments {
		if r.Type != "filter" {
			inputs[r.Name] = fmt.Sprint(r.DefaultValue)
		}
	}

	if values, d := types.MapValueFrom(ctx, types.StringType, inputs); d.HasError() {
		diags.Append(d...)
	} else {
		model.Inputs = values
	}

	filterDefinition := map[string]attr.Type{
		"key":    types.StringType,
		"values": types.ListType{ElemType: types.StringType},
	}

	var (
		filterModels []autoDetectorFilterModel
		currentOrder []autoDetectorFilterModel
	)
	if !model.Filters.IsNull() && !model.Filters.IsUnknown() {
		diags.Append(model.Filters.ElementsAs(ctx, &currentOrder, false)...)
	}

	appendFilter := func(key string, values []string) {
		var vals []attr.Value
		for _, value := range values {
			vals = append(vals, types.StringValue(value))
		}
		filterModels = append(filterModels, autoDetectorFilterModel{
			Key:    types.StringValue(key),
			Values: types.ListValueMust(types.StringType, vals),
		})
	}

	for _, existing := range currentOrder {
		key := existing.Key.ValueString()
		if values, ok := filters[key]; ok {
			appendFilter(key, values)
			delete(filters, key)
		}
	}

	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		appendFilter(key, filters[key])
	}

	if values, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: filterDefinition}, filterModels); d.HasError() {
		diags.Append(d...)
	} else {
		model.Filters = values
	}

	return diags
}
