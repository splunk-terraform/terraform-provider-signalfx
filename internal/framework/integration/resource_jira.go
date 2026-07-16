// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
)

const (
	jiraAuthUsernamePassword = "UsernameAndPassword"
	jiraAuthEmailToken       = "EmailAndToken"
)

type ResourceJira struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceJiraModel struct {
	integrationModel
	APIToken            types.String `tfsdk:"api_token"`
	UserEmail           types.String `tfsdk:"user_email"`
	Username            types.String `tfsdk:"username"`
	Password            types.String `tfsdk:"password"`
	AuthMethod          types.String `tfsdk:"auth_method"`
	BaseURL             types.String `tfsdk:"base_url"`
	IssueType           types.String `tfsdk:"issue_type"`
	ProjectKey          types.String `tfsdk:"project_key"`
	AssigneeName        types.String `tfsdk:"assignee_name"`
	AssigneeDisplayName types.String `tfsdk:"assignee_display_name"`
}

var (
	_ resource.Resource                = (*ResourceJira)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceJira)(nil)
	_ resource.ResourceWithImportState = (*ResourceJira)(nil)
)

func NewResourceJira() resource.Resource { return &ResourceJira{} }

func (jira *ResourceJira) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jira_integration"
}

func (jira *ResourceJira) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["api_token"] = schema.StringAttribute{
		Optional:    true,
		Sensitive:   true,
		Description: "API token used with the `EmailAndToken` authentication method.",
		Validators: []validator.String{stringvalidator.ConflictsWith(
			path.MatchRoot("username"), path.MatchRoot("password"),
		)},
	}
	attributes["user_email"] = schema.StringAttribute{
		Optional:    true,
		Description: "Email address used with the `EmailAndToken` authentication method.",
		Validators: []validator.String{stringvalidator.ConflictsWith(
			path.MatchRoot("username"), path.MatchRoot("password"),
		)},
	}
	attributes["username"] = schema.StringAttribute{
		Optional:    true,
		Description: "User name used with the `UsernameAndPassword` authentication method.",
		Validators: []validator.String{stringvalidator.ConflictsWith(
			path.MatchRoot("user_email"), path.MatchRoot("api_token"),
		)},
	}
	attributes["password"] = schema.StringAttribute{
		Optional:    true,
		Sensitive:   true,
		Description: "Password used with the `UsernameAndPassword` authentication method.",
		Validators: []validator.String{stringvalidator.ConflictsWith(
			path.MatchRoot("user_email"), path.MatchRoot("api_token"),
		)},
	}
	attributes["auth_method"] = schema.StringAttribute{
		Required:    true,
		Description: "Authentication method. Allowed values are `UsernameAndPassword` and `EmailAndToken`.",
		Validators: []validator.String{
			stringvalidator.OneOf(jiraAuthUsernamePassword, jiraAuthEmailToken),
		},
	}
	attributes["base_url"] = schema.StringAttribute{
		Required:    true,
		Description: "Base URL of the Jira instance.",
	}
	attributes["issue_type"] = schema.StringAttribute{
		Required:    true,
		Description: "Jira issue type used for tickets created from detector notifications.",
	}
	attributes["project_key"] = schema.StringAttribute{
		Required:    true,
		Description: "Key of the Jira project that receives tickets.",
	}
	attributes["assignee_name"] = schema.StringAttribute{
		Required:    true,
		Description: "Jira user name for the ticket assignee.",
	}
	attributes["assignee_display_name"] = schema.StringAttribute{
		Optional:    true,
		Description: "Jira display name for the ticket assignee.",
	}
	resp.Schema = schema.Schema{
		Description: "Manages a Jira integration in Splunk Observability Cloud.",
		Attributes:  attributes,
	}
}

func (jira *ResourceJira) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceJiraModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := jira.Details().Client.CreateJiraIntegration(ctx, model.jiraIntegration())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	model.updateFromAPI(details, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (jira *ResourceJira) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceJiraModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := jira.Details().Client.GetJiraIntegration(ctx, model.ID.ValueString())
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

func (jira *ResourceJira) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceJiraModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := jira.Details().Client.UpdateJiraIntegration(ctx, model.ID.ValueString(), model.jiraIntegration())
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

func (jira *ResourceJira) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceJiraModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(
		ctx, resp.State, jira.Details().Client.DeleteJiraIntegration(ctx, model.ID.ValueString()),
	)...)
}

func (model resourceJiraModel) jiraIntegration() *integration.JiraIntegration {
	details := &integration.JiraIntegration{
		Type:       integration.Type("Jira"),
		Name:       model.Name.ValueString(),
		Enabled:    model.Enabled.ValueBool(),
		AuthMethod: model.AuthMethod.ValueString(),
		BaseURL:    model.BaseURL.ValueString(),
		IssueType:  model.IssueType.ValueString(),
		ProjectKey: model.ProjectKey.ValueString(),
		Assignee: &integration.JiraAssignee{
			Name:        model.AssigneeName.ValueString(),
			DisplayName: model.AssigneeDisplayName.ValueString(),
		},
	}
	if details.AuthMethod == jiraAuthUsernamePassword {
		details.Username = model.Username.ValueString()
		details.Password = model.Password.ValueString()
	} else {
		details.UserEmail = model.UserEmail.ValueString()
		details.APIToken = model.APIToken.ValueString()
	}
	return details
}

func (model *resourceJiraModel) updateFromAPI(details *integration.JiraIntegration, updateID bool) {
	if details == nil {
		return
	}
	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	model.AuthMethod = types.StringValue(details.AuthMethod)
	model.BaseURL = types.StringValue(details.BaseURL)
	model.IssueType = types.StringValue(details.IssueType)
	model.ProjectKey = types.StringValue(details.ProjectKey)
	if details.AuthMethod == jiraAuthUsernamePassword {
		updateOptionalString(&model.Username, details.Username)
		updateOptionalString(&model.Password, details.Password)
		model.UserEmail = types.StringNull()
		model.APIToken = types.StringNull()
	} else {
		updateOptionalString(&model.UserEmail, details.UserEmail)
		updateOptionalString(&model.APIToken, details.APIToken)
		model.Username = types.StringNull()
		model.Password = types.StringNull()
	}
	if details.Assignee != nil {
		model.AssigneeName = types.StringValue(details.Assignee.Name)
		updateOptionalString(&model.AssigneeDisplayName, details.Assignee.DisplayName)
	}
}
