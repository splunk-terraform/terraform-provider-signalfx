// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwobservability

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go/directory"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

type observabilityDirectoryModel struct {
	ID     types.String `tfsdk:"id"`
	Path   types.String `tfsdk:"path"`
	Pinned types.Bool   `tfsdk:"pinned"`
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
		Description: "Manages a safe, path-only Observability Directory entry. Membership is intentionally not managed by this resource.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Decoded logical Directory path.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
			fmt.Sprintf("%q is reserved by Dashify and cannot be managed by this resource", reserved),
		)
	}
}

func (r *observabilityDirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model observabilityDirectoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, err := r.Details().Client.PatchDirectoryEntry(ctx, model.Path.ValueString(), observabilityDirectoryPatch(model.Pinned))
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if entry == nil || entry.Data == nil {
		resp.Diagnostics.AddError("Error creating directory", "Directory API returned no directory entry")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, observabilityDirectoryModelFromEntry(entry.Data))...)
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
	resp.Diagnostics.Append(resp.State.Set(ctx, observabilityDirectoryModelFromEntry(result.Data))...)
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

	result, err := r.Details().Client.PatchDirectoryEntry(ctx, prior.Path.ValueString(), observabilityDirectoryPatch(model.Pinned))
	if resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if result == nil || result.Data == nil {
		resp.Diagnostics.AddError("Error updating directory", "Directory API returned no directory entry")
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, observabilityDirectoryModelFromEntry(result.Data))...)
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
	if len(entry.Templates) > 0 || len(entry.Children) > 0 || entry.Identity || entry.Canonical {
		resp.Diagnostics.AddError(
			"Refusing to delete directory",
			"The Directory entry is occupied or managed by the service (templates, children, identity, or canonical metadata are present). Remove those dependencies and retry.",
		)
		return
	}

	resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, r.Details().Client.DeleteDirectoryEntry(ctx, entry.Path))...)
}

func (r *observabilityDirectoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
}

func observabilityDirectoryPatch(pinned types.Bool) *directory.Patch {
	value := false
	if !pinned.IsNull() && !pinned.IsUnknown() {
		value = pinned.ValueBool()
	}
	return &directory.Patch{Pinned: &value}
}

func observabilityDirectoryModelFromEntry(entry *directory.Entry) observabilityDirectoryModel {
	return observabilityDirectoryModel{
		ID:     types.StringValue(entry.Path),
		Path:   types.StringValue(entry.Path),
		Pinned: types.BoolValue(entry.Pinned),
	}
}

func observabilityReservedDirectoryPath(value string) string {
	segments := strings.Split(value, "/")
	if len(segments) == 0 {
		return ""
	}
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
