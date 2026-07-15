// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
)

type ResourceVictorOps struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceVictorOpsModel struct {
	integrationModel
	PostURL types.String `tfsdk:"post_url"`
}

var (
	_ resource.Resource                = (*ResourceVictorOps)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceVictorOps)(nil)
	_ resource.ResourceWithImportState = (*ResourceVictorOps)(nil)
)

func NewResourceVictorOps() resource.Resource { return &ResourceVictorOps{} }

func (victorOps *ResourceVictorOps) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_victor_ops_integration"
}

func (victorOps *ResourceVictorOps) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["post_url"] = schema.StringAttribute{
		Optional:    true,
		Description: "Splunk On-Call endpoint that receives alert notifications.",
	}
	resp.Schema = schema.Schema{
		Description: "Manages a legacy VictorOps, now Splunk On-Call, integration in Splunk Observability Cloud.",
		Attributes:  attributes,
	}
}

func (victorOps *ResourceVictorOps) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceVictorOpsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := victorOps.Details().Client.CreateVictorOpsIntegration(ctx, model.victorOpsIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (victorOps *ResourceVictorOps) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceVictorOpsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := victorOps.Details().Client.GetVictorOpsIntegration(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model.updateFromAPI(details, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (victorOps *ResourceVictorOps) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceVictorOpsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := victorOps.Details().Client.UpdateVictorOpsIntegration(
		ctx, model.ID.ValueString(), model.victorOpsIntegration(),
	)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (victorOps *ResourceVictorOps) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceVictorOpsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(
		ctx, resp.State, victorOps.Details().Client.DeleteVictorOpsIntegration(ctx, model.ID.ValueString()),
	)...)
}

func (model resourceVictorOpsModel) victorOpsIntegration() *integration.VictorOpsIntegration {
	return &integration.VictorOpsIntegration{
		Type:    integration.VICTOR_OPS,
		Name:    model.Name.ValueString(),
		Enabled: model.Enabled.ValueBool(),
		PostUrl: model.PostURL.ValueString(),
	}
}

func (model *resourceVictorOpsModel) updateFromAPI(details *integration.VictorOpsIntegration, updateID bool) {
	if details == nil {
		return
	}
	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	// The API omits post_url. Retain the configured value in state.
}
