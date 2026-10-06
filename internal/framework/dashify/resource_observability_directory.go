// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go"
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

const (
	observabilityDirectoryUnoccupiedSummary      = "Unoccupied directory"
	observabilityDirectoryCreateUnoccupiedDetail = "Set pinned to true or provide at least one Template. Terraform cannot manage a new unpinned Directory without Templates because the API represents it the same as an unoccupied path."
)

var (
	_ resource.Resource                = (*observabilityDirectoryResource)(nil)
	_ resource.ResourceWithConfigure   = (*observabilityDirectoryResource)(nil)
	_ resource.ResourceWithImportState = (*observabilityDirectoryResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*observabilityDirectoryResource)(nil)
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
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^(?:|[^/]+(?:/[^/]+)*)$`),
						"use a decoded logical Directory path without leading, trailing, or consecutive slashes",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"templates": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Description: "Complete ordered list of Template IDs assigned to this Directory. Updating this attribute replaces the entire backend list; concurrent UI or API changes are last-write-wins.",
				Validators:  observabilityTemplateIDListValidators(),
			},
			"pinned": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the Directory entry is pinned. Defaults to true so an empty Directory remains occupied.",
			},
		},
	}
}

func (r *observabilityDirectoryResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model observabilityDirectoryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !model.Path.IsNull() && !model.Path.IsUnknown() {
		if reserved := observabilityReservedDirectoryPath(model.Path.ValueString()); reserved != "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("path"),
				"Reserved directory path",
				fmt.Sprintf("%q is reserved by the dashboard service and cannot be managed by this resource", reserved),
			)
		}
	}
}

func (r *observabilityDirectoryResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if !req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var pinned types.Bool
	var templates types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("pinned"), &pinned)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("templates"), &templates)...)
	if resp.Diagnostics.HasError() || pinned.IsUnknown() || templates.IsUnknown() {
		return
	}
	if observabilityDirectoryPlanUnoccupied(pinned, templates) {
		resp.Diagnostics.AddAttributeError(path.Root("pinned"), observabilityDirectoryUnoccupiedSummary, observabilityDirectoryCreateUnoccupiedDetail)
	}
}

func (r *observabilityDirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model observabilityDirectoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if reserved := observabilityReservedDirectoryPath(model.Path.ValueString()); reserved != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("path"),
			"Reserved directory path",
			fmt.Sprintf("%q is reserved by the dashboard service and cannot be managed by this resource", reserved),
		)
		return
	}
	// PATCH is an upsert and replaces the complete Template membership list.
	// A GET can also return a synthetic unoccupied entry for an absent path.
	// Refuse to claim an occupied entry; users can import it instead.
	existing, diags := r.fetchDirectoryEntry(ctx, resp.State, model.Path.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if observabilityDirectoryEntryOccupied(existing) {
		resp.Diagnostics.AddAttributeError(
			path.Root("path"),
			"Directory already exists",
			fmt.Sprintf("Directory %q already exists. Import it to manage the existing entry.", model.Path.ValueString()),
		)
		return
	}
	if observabilityDirectoryPlanUnoccupied(model.Pinned, model.Templates) {
		resp.Diagnostics.AddAttributeError(path.Root("pinned"), observabilityDirectoryUnoccupiedSummary, observabilityDirectoryCreateUnoccupiedDetail)
		return
	}

	patch, diags := observabilityDirectoryPatch(ctx, model.Pinned, model.Templates)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, err := r.Details().Client.PatchDirectoryEntry(ctx, model.Path.ValueString(), patch)
	if resp.Diagnostics.Append(observabilityDirectoryRequestError(ctx, resp.State, "creating", err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if entry == nil || entry.Data == nil {
		resp.Diagnostics.AddError("Error creating directory", "Directory API returned no directory entry")
		return
	}
	next, diags := observabilityDirectoryModelFromEntry(ctx, model.Path.ValueString(), entry.Data)
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
	if resp.Diagnostics.Append(observabilityDirectoryRequestError(ctx, resp.State, "reading", err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if result == nil || result.Data == nil {
		resp.Diagnostics.AddError("Error reading directory", "Directory API returned no directory entry")
		return
	}
	next, diags := observabilityDirectoryModelFromEntry(ctx, state.Path.ValueString(), result.Data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !observabilityDirectoryEntryOccupied(result.Data) {
		resp.State.RemoveResource(ctx)
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
	if observabilityDirectoryPlanUnoccupied(model.Pinned, model.Templates) {
		current, fetchDiags := r.fetchDirectoryEntry(ctx, resp.State, prior.Path.ValueString())
		resp.Diagnostics.Append(fetchDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(current.Children) == 0 && !current.Identity && !current.Canonical {
			resp.Diagnostics.AddAttributeError(path.Root("pinned"), "Unoccupied directory", "Set pinned to true or provide at least one Template or child directory. An unpinned Directory without Templates or children is removed by the service.")
			return
		}
	}

	result, err := r.Details().Client.PatchDirectoryEntry(ctx, prior.Path.ValueString(), patch)
	if resp.Diagnostics.Append(observabilityDirectoryRequestError(ctx, resp.State, "updating", err)...); resp.Diagnostics.HasError() || err != nil {
		return
	}
	if result == nil || result.Data == nil {
		resp.Diagnostics.AddError("Error updating directory", "Directory API returned no directory entry")
		return
	}
	next, diags := observabilityDirectoryModelFromEntry(ctx, prior.Path.ValueString(), result.Data)
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
	if resp.Diagnostics.Append(observabilityDirectoryRequestError(ctx, resp.State, "checking", err)...); resp.Diagnostics.HasError() || err != nil {
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
	if !observabilityDirectoryEntryOccupied(entry) {
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

	err = r.Details().Client.DeleteDirectoryEntry(ctx, entry.Path)
	resp.Diagnostics.Append(observabilityDirectoryRequestError(ctx, resp.State, "deleting", err)...)
}

// fetchDirectoryEntry fetches the live entry for path and validates it was
// returned successfully and for the expected logical path.
func (r *observabilityDirectoryResource) fetchDirectoryEntry(ctx context.Context, state tfsdk.State, path string) (*directory.Entry, diag.Diagnostics) {
	var diags diag.Diagnostics
	result, err := r.Details().Client.GetDirectoryEntry(ctx, path)
	if diags.Append(observabilityDirectoryRequestError(ctx, state, "checking", err)...); diags.HasError() || err != nil {
		return nil, diags
	}
	if result == nil || result.Data == nil {
		diags.AddError("Error checking directory", "Directory API returned no directory entry")
		return nil, diags
	}
	if result.Data.Path != path {
		diags.AddError("Error checking directory", "Directory API returned a different logical path than the requested path")
		return nil, diags
	}
	return result.Data, diags
}

func observabilityDirectoryRequestError(ctx context.Context, state tfsdk.State, action string, err error) diag.Diagnostics {
	if responseError, ok := signalfx.AsResponseError(err); ok && responseError.Code() == http.StatusNotFound {
		var diags diag.Diagnostics
		detail := fmt.Sprintf("Directory API returned HTTP 404 for %q.", responseError.Route())
		switch action {
		case "checking", "reading":
			detail += " Empty Directory paths normally return an entry, so Terraform cannot conclude the entry is absent."
		default:
			detail += " The entry was confirmed to exist moments earlier; retry after checking the entry's current state."
		}
		if responseDetails := strings.TrimSpace(responseError.Details()); responseDetails != "" {
			detail += " API response: " + responseDetails
		}
		diags.AddError("Error "+action+" directory", detail)
		return diags
	}
	return fwerr.ErrorHandler(ctx, state, err)
}

func (r *observabilityDirectoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
}

func observabilityDirectoryPatch(ctx context.Context, pinned types.Bool, templates types.List) (*directory.PatchDirectoryEntryRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	value := true
	if !pinned.IsNull() && !pinned.IsUnknown() {
		value = pinned.ValueBool()
	}
	patch := &directory.PatchDirectoryEntryRequest{Pinned: &value}

	if templates.IsNull() || templates.IsUnknown() {
		return patch, diags
	}
	var ids []string
	diags.Append(templates.ElementsAs(ctx, &ids, false)...)
	if diags.HasError() {
		return nil, diags
	}
	references := make([]string, len(ids))
	for i, id := range ids {
		if !observabilityTemplateIDValid(id) {
			diags.AddAttributeError(path.Root("templates").AtListIndex(i), "Invalid Template ID", "Use a Template ID without an API path or slashes.")
			return nil, diags
		}
		references[i] = observabilityTemplateReference(id)
	}
	patch.Templates = references
	return patch, diags
}

func observabilityDirectoryPlanUnoccupied(pinned types.Bool, templates types.List) bool {
	if pinned.IsNull() || pinned.IsUnknown() || pinned.ValueBool() || templates.IsUnknown() {
		return false
	}
	return templates.IsNull() || len(templates.Elements()) == 0
}

func observabilityDirectoryEntryOccupied(entry *directory.Entry) bool {
	return entry.Pinned || len(entry.Templates) > 0 || len(entry.Children) > 0 || entry.Identity || entry.Canonical
}

func observabilityDirectoryModelFromEntry(ctx context.Context, expectedPath string, entry *directory.Entry) (observabilityDirectoryModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if entry.Path != expectedPath {
		diags.AddError("Unexpected Directory path", fmt.Sprintf("The Directory API returned path %q for %q; Terraform state was not changed.", entry.Path, expectedPath))
		return observabilityDirectoryModel{}, diags
	}
	ids := make([]string, len(entry.Templates))
	for i, reference := range entry.Templates {
		id, ok := observabilityTemplateIDFromReference(reference)
		if !ok {
			diags.AddError("Invalid Directory Template reference", fmt.Sprintf("The Directory API returned an unsupported Template reference at position %d.", i+1))
			return observabilityDirectoryModel{}, diags
		}
		ids[i] = id
	}
	templates, valueDiags := types.ListValueFrom(ctx, types.StringType, ids)
	diags.Append(valueDiags...)
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
