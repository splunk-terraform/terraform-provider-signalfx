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

type ResourceAWSToken struct {
	fwembed.ResourceData
}

type resourceAWSTokenModel struct {
	awsBootstrapModel
	TokenID types.String `tfsdk:"token_id"`
}

var (
	_ resource.Resource              = (*ResourceAWSToken)(nil)
	_ resource.ResourceWithConfigure = (*ResourceAWSToken)(nil)
)

func NewResourceAWSToken() resource.Resource {
	return &ResourceAWSToken{}
}

func (awsToken *ResourceAWSToken) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_token_integration"
}

func (awsToken *ResourceAWSToken) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := awsBootstrapAttributes()
	attributes["token_id"] = schema.StringAttribute{
		Computed:    true,
		Sensitive:   true,
		Description: "Legacy computed token identifier. The service does not currently return a value for this attribute.",
	}

	resp.Schema = schema.Schema{
		Description: "Bootstraps an AWS CloudWatch integration using AWS security-token authentication. Complete the workflow with signalfx_aws_integration.",
		Attributes:  attributes,
	}
}

func (awsToken *ResourceAWSToken) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceAWSTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := awsToken.Details().Client.CreateAWSCloudWatchIntegration(
		ctx,
		model.awsIntegration(integration.SECURITY_TOKEN),
	)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.Diagnostics.AddError("Missing AWS integration response", "Splunk Observability Cloud returned no AWS integration after creation.")
		return
	}

	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (awsToken *ResourceAWSToken) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceAWSTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, err := awsToken.Details().Client.GetAWSCloudWatchIntegration(ctx, model.ID.ValueString())
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

func (*ResourceAWSToken) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceAWSTokenModel
	var current resourceAWSTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// All configurable fields require replacement. Retain generated values if
	// the Framework invokes Update solely for a state-only transition.
	plan.ID = current.ID
	plan.TokenID = current.TokenID
	plan.SignalFxAWSAccount = current.SignalFxAWSAccount
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (*ResourceAWSToken) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceAWSTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	// Deletion is intentionally owned by signalfx_aws_integration. Returning
	// without diagnostics removes this bootstrap resource from Terraform state.
}

func (model *resourceAWSTokenModel) updateFromAPI(details *integration.AwsCloudWatchIntegration, updateIdentity bool) {
	if details == nil {
		return
	}

	model.awsBootstrapModel.updateFromAPI(details, updateIdentity)
	if model.TokenID.IsUnknown() {
		// The SDK resource left token_id unset because the API does not return a
		// token identifier. Framework computed values must be known after apply.
		model.TokenID = types.StringValue("")
	}
}
