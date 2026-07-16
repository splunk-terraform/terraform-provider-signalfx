// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

// awsBootstrapModel contains state shared by the external-ID and security-token
// resources that bootstrap an AWS CloudWatch integration. The full AWS resource
// completes and ultimately deletes the integration created by these resources.
type awsBootstrapModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	SignalFxAWSAccount types.String `tfsdk:"signalfx_aws_account"`
}

func awsBootstrapAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": fwshared.ResourceIDAttribute(),
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Name of the integration.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"signalfx_aws_account": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "Splunk Observability Cloud AWS account ARN to use with the AWS role.",
		},
	}
}

func (model awsBootstrapModel) awsIntegration(authMethod integration.AwsAuthMethod) *integration.AwsCloudWatchIntegration {
	return &integration.AwsCloudWatchIntegration{
		Type:       "AWSCloudWatch",
		AuthMethod: authMethod,
		Name:       model.Name.ValueString(),
		PollRate:   300000,
	}
}

func (model *awsBootstrapModel) updateFromAPI(details *integration.AwsCloudWatchIntegration, updateIdentity bool) {
	if details == nil {
		return
	}

	if updateIdentity {
		model.ID = types.StringValue(details.Id)
		model.Name = types.StringValue(details.Name)
	}
	model.SignalFxAWSAccount = types.StringValue(details.SfxAwsAccountArn)
}
