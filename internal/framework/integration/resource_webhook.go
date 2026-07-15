// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/integration"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
)

type ResourceWebhook struct {
	fwembed.ResourceData
	fwembed.ResourceIDImporter
}

type resourceWebhookModel struct {
	integrationModel
	URL             types.String `tfsdk:"url"`
	SharedSecret    types.String `tfsdk:"shared_secret"`
	Headers         types.Set    `tfsdk:"headers"`
	Method          types.String `tfsdk:"method"`
	PayloadTemplate types.String `tfsdk:"payload_template"`
}

type webhookHeaderModel struct {
	Key   types.String `tfsdk:"header_key"`
	Value types.String `tfsdk:"header_value"`
}

var (
	_ resource.Resource                = (*ResourceWebhook)(nil)
	_ resource.ResourceWithConfigure   = (*ResourceWebhook)(nil)
	_ resource.ResourceWithImportState = (*ResourceWebhook)(nil)

	webhookHeaderAttributeTypes = map[string]attr.Type{
		"header_key":   types.StringType,
		"header_value": types.StringType,
	}
	webhookHeaderObjectType = types.ObjectType{AttrTypes: webhookHeaderAttributeTypes}
)

func NewResourceWebhook() resource.Resource { return &ResourceWebhook{} }

func (webhook *ResourceWebhook) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_integration"
}

func (webhook *ResourceWebhook) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := integrationAttributes()
	attributes["url"] = schema.StringAttribute{
		Optional:    true,
		Description: "URL that receives webhook requests.",
	}
	attributes["shared_secret"] = schema.StringAttribute{
		Optional:    true,
		Sensitive:   true,
		Description: "Secret used to sign webhook requests.",
	}
	attributes["method"] = schema.StringAttribute{
		Optional:    true,
		Description: "HTTP method used for webhook requests, such as GET, POST, or PUT.",
	}
	attributes["payload_template"] = schema.StringAttribute{
		Optional:    true,
		Description: "JSON template used for the webhook request payload.",
	}
	resp.Schema = schema.Schema{
		Description: "Manages a webhook integration in Splunk Observability Cloud.",
		Attributes:  attributes,
		Blocks: map[string]schema.Block{
			"headers": schema.SetNestedBlock{
				Description: "HTTP headers included with webhook requests.",
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"header_key": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "HTTP header name.",
					},
					"header_value": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "HTTP header value.",
					},
				}},
			},
		},
	}
}

func (webhook *ResourceWebhook) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceWebhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := model.webhookIntegration(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := webhook.Details().Client.CreateWebhookIntegration(ctx, payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(details, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (webhook *ResourceWebhook) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model resourceWebhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := webhook.Details().Client.GetWebhookIntegration(ctx, model.ID.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(details, false)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (webhook *ResourceWebhook) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceWebhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload, diags := model.webhookIntegration(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	details, err := webhook.Details().Client.UpdateWebhookIntegration(ctx, model.ID.ValueString(), payload)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, withAdminTokenHelp(err))...); resp.Diagnostics.HasError() {
		return
	}
	if details == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(model.updateFromAPI(details, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (webhook *ResourceWebhook) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceWebhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(fwerr.ErrorHandler(
		ctx, resp.State, webhook.Details().Client.DeleteWebhookIntegration(ctx, model.ID.ValueString()),
	)...)
}

func (model resourceWebhookModel) webhookIntegration(ctx context.Context) (*integration.WebhookIntegration, diag.Diagnostics) {
	payload := &integration.WebhookIntegration{
		Type:            integration.WEBHOOK,
		Name:            model.Name.ValueString(),
		Enabled:         model.Enabled.ValueBool(),
		Url:             model.URL.ValueString(),
		SharedSecret:    model.SharedSecret.ValueString(),
		Method:          model.Method.ValueString(),
		PayloadTemplate: model.PayloadTemplate.ValueString(),
	}
	if model.Headers.IsNull() || model.Headers.IsUnknown() {
		return payload, nil
	}
	var headers []webhookHeaderModel
	diags := model.Headers.ElementsAs(ctx, &headers, false)
	if diags.HasError() {
		return nil, diags
	}
	sort.Slice(headers, func(i, j int) bool {
		if headers[i].Key.ValueString() == headers[j].Key.ValueString() {
			return headers[i].Value.ValueString() < headers[j].Value.ValueString()
		}
		return headers[i].Key.ValueString() < headers[j].Key.ValueString()
	})
	if len(headers) > 0 {
		payload.Headers = make(map[string]any, len(headers))
		for _, header := range headers {
			payload.Headers[header.Key.ValueString()] = header.Value.ValueString()
		}
	}
	return payload, diags
}

func (model *resourceWebhookModel) updateFromAPI(details *integration.WebhookIntegration, updateID bool) diag.Diagnostics {
	if details == nil {
		return nil
	}
	if updateID {
		model.updateWithID(details.Id, details.Name, details.Enabled)
	} else {
		model.update(details.Name, details.Enabled)
	}
	updateOptionalString(&model.URL, details.Url)
	updateOptionalString(&model.SharedSecret, details.SharedSecret)
	updateOptionalString(&model.Method, details.Method)
	updateOptionalString(&model.PayloadTemplate, details.PayloadTemplate)
	if len(details.Headers) == 0 {
		return nil
	}

	keys := make([]string, 0, len(details.Headers))
	for key := range details.Headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	elements := make([]attr.Value, 0, len(keys))
	var diags diag.Diagnostics
	for _, key := range keys {
		value, ok := details.Headers[key].(string)
		if !ok {
			diags.AddError("Invalid webhook header", fmt.Sprintf("Header %q has non-string value %T", key, details.Headers[key]))
			continue
		}
		object, objectDiags := types.ObjectValue(webhookHeaderAttributeTypes, map[string]attr.Value{
			"header_key": types.StringValue(key), "header_value": types.StringValue(value),
		})
		diags.Append(objectDiags...)
		elements = append(elements, object)
	}
	if diags.HasError() {
		return diags
	}
	set, setDiags := types.SetValue(webhookHeaderObjectType, elements)
	diags.Append(setDiags...)
	if !diags.HasError() {
		model.Headers = set
	}
	return diags
}

func updateOptionalString(target *types.String, value string) {
	if value != "" {
		*target = types.StringValue(value)
	}
}
