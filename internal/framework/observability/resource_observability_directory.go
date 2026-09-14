// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/directory"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type observabilityDirectoryModel struct {
	ID        types.String `tfsdk:"id"`
	Path      types.String `tfsdk:"path"`
	Templates types.List   `tfsdk:"templates"`
	Pinned    types.Bool   `tfsdk:"pinned"`
}

type observabilityDirectoryResource struct {
	fwembed.ResourceData
}

var (
	_ resource.Resource                = (*observabilityDirectoryResource)(nil)
	_ resource.ResourceWithConfigure   = (*observabilityDirectoryResource)(nil)
	_ resource.ResourceWithImportState = (*observabilityDirectoryResource)(nil)
)

func NewResourceObservabilityDirectory() resource.Resource {
	return &observabilityDirectoryResource{}
}

func (r *observabilityDirectoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_observability_directory"
}

func (r *observabilityDirectoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.ResourceData.Configure(ctx, req, resp)
}

func (r *observabilityDirectoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Observability Directory entry and its complete ordered Template membership list.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Decoded logical Directory path.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"templates": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "Complete ordered list of Template API references assigned to this Directory, such as /v2/template/<id>. Updating this attribute replaces the entire backend list; concurrent UI or API changes are last-write-wins.",
				Validators:  append(nonEmptyStringListValidators(), listvalidator.UniqueValues()),
			},
			"pinned": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the Directory entry is pinned.",
			},
		},
	}
}

func (r *observabilityDirectoryResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model observabilityDirectoryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() || model.Path.IsNull() || model.Path.IsUnknown() {
		return
	}
	if reserved := observabilityReservedDirectoryPath(model.Path.ValueString()); reserved != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("path"),
			"Reserved directory path",
			fmt.Sprintf("%q is reserved by the dashboard service and cannot be managed by this resource", reserved),
		)
	}
}

func (r *observabilityDirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model observabilityDirectoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch, diags := observabilityDirectoryPatch(ctx, model.Pinned, model.Templates)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, err := r.Details().Client.PatchDirectoryEntry(ctx, model.Path.ValueString(), patch)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if entry == nil || entry.Data == nil {
		resp.Diagnostics.AddError("Error creating directory", "Directory API returned no directory entry")
		return
	}
	next, diags := observabilityDirectoryModelFromEntry(ctx, entry.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, next)...)
}

func (r *observabilityDirectoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state observabilityDirectoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.Details().Client.GetDirectoryEntry(ctx, state.Path.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if result == nil || result.Data == nil {
		resp.Diagnostics.AddError("Error reading directory", "Directory API returned no directory entry")
		return
	}
	next, diags := observabilityDirectoryModelFromEntry(ctx, result.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, next)...)
}

func (r *observabilityDirectoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model observabilityDirectoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var prior observabilityDirectoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patch, diags := observabilityDirectoryPatch(ctx, model.Pinned, model.Templates)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.Details().Client.PatchDirectoryEntry(ctx, prior.Path.ValueString(), patch)
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if result == nil || result.Data == nil {
		resp.Diagnostics.AddError("Error updating directory", "Directory API returned no directory entry")
		return
	}
	next, diags := observabilityDirectoryModelFromEntry(ctx, result.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, next)...)
}

func (r *observabilityDirectoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state observabilityDirectoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.Details().Client.GetDirectoryEntry(ctx, state.Path.ValueString())
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if result == nil || result.Data == nil {
		resp.Diagnostics.AddError("Error deleting directory", "Directory API returned no directory entry")
		return
	}
	entry := result.Data
	if entry.Path != state.Path.ValueString() {
		resp.Diagnostics.AddError("Refusing to delete directory", "The Directory API returned a different logical path than the provider state; no entry was deleted.")
		return
	}
	if reserved := observabilityReservedDirectoryPath(entry.Path); reserved != "" {
		resp.Diagnostics.AddError("Refusing to delete directory", fmt.Sprintf("%q is a reserved Directory path.", reserved))
		return
	}
	if len(entry.Children) > 0 || entry.Identity || entry.Canonical {
		resp.Diagnostics.AddError(
			"Refusing to delete directory",
			"The Directory entry is managed by the service or contains child directories (children, identity, or canonical metadata are present). Remove those dependencies and retry.",
		)
		return
	}
	if len(entry.Templates) > 0 {
		resp.Diagnostics.AddWarning(
			"Deleting directory with Template memberships",
			fmt.Sprintf("The Directory entry still references %d Template(s); deleting it removes those membership links but does not delete the referenced Template records.", len(entry.Templates)),
		)
	}

	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, r.Details().Client.DeleteDirectoryEntry(ctx, entry.Path))...)
}

func (r *observabilityDirectoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
}

func observabilityDirectoryPatch(ctx context.Context, pinned types.Bool, templates types.List) (*directory.Patch, diag.Diagnostics) {
	var diags diag.Diagnostics
	value := false
	if !pinned.IsNull() && !pinned.IsUnknown() {
		value = pinned.ValueBool()
	}
	patch := &directory.Patch{Pinned: &value}

	if templates.IsNull() || templates.IsUnknown() {
		return patch, diags
	}
	var references []string
	diags.Append(templates.ElementsAs(ctx, &references, false)...)
	if diags.HasError() {
		return nil, diags
	}
	patch.Templates = &references
	return patch, diags
}

func observabilityDirectoryModelFromEntry(ctx context.Context, entry *directory.Entry) (observabilityDirectoryModel, diag.Diagnostics) {
	references := entry.Templates
	if references == nil {
		references = []string{}
	}
	templates, diags := types.ListValueFrom(ctx, types.StringType, references)
	return observabilityDirectoryModel{
		ID:        types.StringValue(entry.Path),
		Path:      types.StringValue(entry.Path),
		Templates: templates,
		Pinned:    types.BoolValue(entry.Pinned),
	}, diags
}

func observabilityReservedDirectoryPath(value string) string {
	segments := strings.Split(value, "/")
	for _, base := range []string{"~demo", "~local", "~signalview", "~templates"} {
		if segments[0] == base {
			return value
		}
	}
	for _, reserved := range []string{"~favorites", "~observability", "~organization", "~recents", "~users"} {
		if segments[len(segments)-1] == reserved {
			return value
		}
	}
	if len(segments) == 2 && segments[0] == "~users" {
		return value
	}
	if len(segments) >= 2 && strings.Join(segments[len(segments)-2:], "/") == "~observability/homepage" {
		return value
	}
	return ""
}
