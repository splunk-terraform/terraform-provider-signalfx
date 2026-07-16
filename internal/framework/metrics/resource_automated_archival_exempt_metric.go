// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwmetrics

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	automatedarchival "github.com/signalfx/signalfx-go/automated-archival"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type ResourceAutomatedArchivalExemptMetric struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceAutomatedArchivalExemptMetricModel struct {
	ID            types.String `tfsdk:"id"`
	ExemptMetrics types.List   `tfsdk:"exempt_metrics"`
}

type automatedArchivalExemptMetricModel struct {
	Creator       types.String `tfsdk:"creator"`
	LastUpdatedBy types.String `tfsdk:"last_updated_by"`
	Created       types.Int64  `tfsdk:"created"`
	LastUpdated   types.Int64  `tfsdk:"last_updated"`
	Name          types.String `tfsdk:"name"`
}

var (
	_ resource.Resource                   = (*ResourceAutomatedArchivalExemptMetric)(nil)
	_ resource.ResourceWithConfigure      = (*ResourceAutomatedArchivalExemptMetric)(nil)
	_ resource.ResourceWithImportState    = (*ResourceAutomatedArchivalExemptMetric)(nil)
	_ resource.ResourceWithValidateConfig = (*ResourceAutomatedArchivalExemptMetric)(nil)
)

var automatedArchivalExemptMetricAttributeTypes = map[string]attr.Type{
	"creator":         types.StringType,
	"last_updated_by": types.StringType,
	"created":         types.Int64Type,
	"last_updated":    types.Int64Type,
	"name":            types.StringType,
}

func NewResourceAutomatedArchivalExemptMetric() resource.Resource {
	return &ResourceAutomatedArchivalExemptMetric{}
}

func (exemptMetric *ResourceAutomatedArchivalExemptMetric) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automated_archival_exempt_metric"
}

func (exemptMetric *ResourceAutomatedArchivalExemptMetric) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a replacement-only group of metrics exempted from automated archival in Splunk Observability Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
		},
		Blocks: map[string]schema.Block{
			"exempt_metrics": schema.ListNestedBlock{
				Description: "Ordered metrics to exempt from automated archival. Changing this list replaces the resource.",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"creator": schema.StringAttribute{
						Computed:    true,
						Description: "ID of the user who created the exempt metric.",
					},
					"last_updated_by": schema.StringAttribute{
						Computed:    true,
						Description: "ID of the user who most recently updated the exempt metric.",
					},
					"created": schema.Int64Attribute{
						Computed:    true,
						Description: "Creation timestamp returned by Splunk Observability Cloud.",
					},
					"last_updated": schema.Int64Attribute{
						Computed:    true,
						Description: "Most recent update timestamp returned by Splunk Observability Cloud.",
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Name of the metric to exempt from automated archival.",
					},
				}},
			},
		},
	}
}

func (*ResourceAutomatedArchivalExemptMetric) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model resourceAutomatedArchivalExemptMetricModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() || model.ExemptMetrics.IsUnknown() {
		return
	}
	if model.ExemptMetrics.IsNull() || len(model.ExemptMetrics.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("exempt_metrics"),
			"Missing automated archival exempt metrics",
			"The exempt_metrics block must contain at least one known metric.",
		)
	}
}

func (exemptMetric *ResourceAutomatedArchivalExemptMetric) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceAutomatedArchivalExemptMetricModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.toAPI(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := exemptMetric.Details().Client.CreateExemptMetrics(ctx, &payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil || len(*details) == 0 {
		resp.Diagnostics.AddError("Unable to create automated archival exempt metrics", "The automated archival API returned no exempt metrics.")
		return
	}

	ids, err := automatedArchivalExemptMetricResponseIDs(*details)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create automated archival exempt metrics", err.Error())
		return
	}
	model.ID = types.StringValue(strings.Join(ids, ","))
	resp.Diagnostics.Append(model.updateFromAPI(ctx, *details)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (exemptMetric *ResourceAutomatedArchivalExemptMetric) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceAutomatedArchivalExemptMetricModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids, err := automatedArchivalExemptMetricIDs(model.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read automated archival exempt metrics", err.Error())
		return
	}

	details, err := exemptMetric.Details().Client.GetExemptMetrics(ctx)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	managed, foundIDs := filterAutomatedArchivalExemptMetrics(*details, ids)
	if len(managed) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	model.ID = types.StringValue(strings.Join(foundIDs, ","))
	resp.Diagnostics.Append(model.updateFromAPI(ctx, managed)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (*ResourceAutomatedArchivalExemptMetric) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var current resourceAutomatedArchivalExemptMetricModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Every configurable value requires replacement. Preserve the current
	// resource if the Framework invokes Update for a state-only transition.
	resp.Diagnostics.Append(resp.State.Set(ctx, &current)...)
}

func (exemptMetric *ResourceAutomatedArchivalExemptMetric) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceAutomatedArchivalExemptMetricModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids, err := automatedArchivalExemptMetricIDs(model.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete automated archival exempt metrics", err.Error())
		return
	}

	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, exemptMetric.Details().Client.DeleteExemptMetrics(
		ctx, &automatedarchival.ExemptMetricDeleteRequest{Ids: ids},
	))...)
}

func (model resourceAutomatedArchivalExemptMetricModel) toAPI(ctx context.Context) ([]automatedarchival.ExemptMetric, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if model.ExemptMetrics.IsNull() || model.ExemptMetrics.IsUnknown() {
		diagnostics.AddError("Invalid automated archival exempt metrics", "The exempt_metrics block must contain at least one known metric.")
		return nil, diagnostics
	}

	var values []automatedArchivalExemptMetricModel
	diagnostics.Append(model.ExemptMetrics.ElementsAs(ctx, &values, false)...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	payload := make([]automatedarchival.ExemptMetric, 0, len(values))
	for i, value := range values {
		if value.Name.IsNull() || value.Name.IsUnknown() {
			diagnostics.AddError(
				"Invalid automated archival exempt metric name",
				fmt.Sprintf("Exempt metric at index %d must have a known name.", i),
			)
			continue
		}
		payload = append(payload, automatedarchival.ExemptMetric{Name: value.Name.ValueString()})
	}
	return payload, diagnostics
}

func (model *resourceAutomatedArchivalExemptMetricModel) updateFromAPI(ctx context.Context, details []automatedarchival.ExemptMetric) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	current := make([]automatedArchivalExemptMetricModel, 0)
	if !model.ExemptMetrics.IsNull() && !model.ExemptMetrics.IsUnknown() {
		diagnostics.Append(model.ExemptMetrics.ElementsAs(ctx, &current, false)...)
		if diagnostics.HasError() {
			return diagnostics
		}
	}

	values := make([]automatedArchivalExemptMetricModel, 0, len(details))
	for i, detail := range details {
		prior := automatedArchivalExemptMetricModel{
			Creator:       types.StringNull(),
			LastUpdatedBy: types.StringNull(),
			Created:       types.Int64Null(),
			LastUpdated:   types.Int64Null(),
		}
		if i < len(current) {
			prior = current[i]
		}
		values = append(values, automatedArchivalExemptMetricModel{
			Creator:       optionalStringValue(prior.Creator, detail.Creator),
			LastUpdatedBy: optionalStringValue(prior.LastUpdatedBy, detail.LastUpdatedBy),
			Created:       optionalInt64Value(prior.Created, detail.Created),
			LastUpdated:   optionalInt64Value(prior.LastUpdated, detail.LastUpdated),
			Name:          types.StringValue(detail.Name),
		})
	}

	list, listDiagnostics := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: automatedArchivalExemptMetricAttributeTypes}, values)
	diagnostics.Append(listDiagnostics...)
	if !diagnostics.HasError() {
		model.ExemptMetrics = list
	}
	return diagnostics
}

func automatedArchivalExemptMetricIDs(id string) ([]string, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("resource ID must contain at least one exempt metric ID")
	}

	parts := strings.Split(id, ",")
	ids := make([]string, 0, len(parts))
	for i, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			return nil, fmt.Errorf("resource ID contains an empty exempt metric ID at index %d", i)
		}
		ids = append(ids, value)
	}
	return ids, nil
}

func automatedArchivalExemptMetricResponseIDs(details []automatedarchival.ExemptMetric) ([]string, error) {
	ids := make([]string, 0, len(details))
	for i, detail := range details {
		if detail.Id == nil || strings.TrimSpace(*detail.Id) == "" {
			return nil, fmt.Errorf("the automated archival API returned no ID for exempt metric at index %d", i)
		}
		ids = append(ids, strings.TrimSpace(*detail.Id))
	}
	return ids, nil
}

func filterAutomatedArchivalExemptMetrics(details []automatedarchival.ExemptMetric, ids []string) ([]automatedarchival.ExemptMetric, []string) {
	byID := make(map[string]automatedarchival.ExemptMetric, len(details))
	for _, detail := range details {
		if detail.Id == nil || strings.TrimSpace(*detail.Id) == "" {
			continue
		}
		byID[strings.TrimSpace(*detail.Id)] = detail
	}

	managed := make([]automatedarchival.ExemptMetric, 0, len(ids))
	foundIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		if detail, ok := byID[id]; ok {
			managed = append(managed, detail)
			foundIDs = append(foundIDs, id)
		}
	}
	return managed, foundIDs
}
