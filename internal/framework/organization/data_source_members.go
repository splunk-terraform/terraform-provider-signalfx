// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fworganization

import (
	"context"
	"fmt"
	"hash/fnv"
	"net/mail"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	fwembed "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/embed"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/fwerr"
	fwshared "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/shared"
)

const organizationMembersPageSize = 1000

type DataSourceMembers struct {
	fwembed.DatasourceData
}

type dataSourceMembersModel struct {
	ID     types.String `tfsdk:"id"`
	Emails types.List   `tfsdk:"emails"`
	Users  types.List   `tfsdk:"users"`
}

type emailListValidator struct{}

var (
	_ datasource.DataSource              = (*DataSourceMembers)(nil)
	_ datasource.DataSourceWithConfigure = (*DataSourceMembers)(nil)
	_ validator.List                     = emailListValidator{}
)

func NewDataSourceMembers() datasource.DataSource {
	return &DataSourceMembers{}
}

func (members *DataSourceMembers) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_members"
}

func (members *DataSourceMembers) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Queries organization members by email address for use in other resources. The provider token must have administrator privileges.",
		Attributes: map[string]schema.Attribute{
			"id": fwshared.ResourceIDAttribute(),
			"emails": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Email addresses of organization members to query.",
				Validators:  []validator.List{emailListValidator{}},
			},
			"users": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Organization member IDs matching the configured email addresses.",
			},
		},
	}
}

func (members *DataSourceMembers) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model dataSourceMembersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var emails []string
	resp.Diagnostics.Append(model.Emails.ElementsAs(ctx, &emails, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users := make([]string, 0)
	for _, email := range emails {
		for offset := 0; ; offset += organizationMembersPageSize {
			results, err := members.Details().Client.GetOrganizationMembers(
				ctx,
				organizationMembersPageSize,
				fmt.Sprintf("email:%s", email),
				offset,
				"-sf_timestamp",
			)
			if err != nil {
				resp.Diagnostics.Append(fwerr.ErrorHandler(ctx, resp.State, err)...)
				return
			}

			for _, member := range results.Results {
				if member != nil {
					users = append(users, member.Id)
				}
			}
			if len(results.Results) == 0 || offset+len(results.Results) >= int(results.Count) {
				break
			}
		}
	}

	model.ID = types.StringValue(organizationMembersID(emails))
	usersValue, valueDiags := types.ListValueFrom(ctx, types.StringType, users)
	resp.Diagnostics.Append(valueDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Users = usersValue
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func organizationMembersID(emails []string) string {
	hasher := fnv.New64()
	for _, email := range emails {
		_, _ = fmt.Fprint(hasher, email)
	}
	return strconv.FormatUint(hasher.Sum64(), 36)
}

func (emailListValidator) Description(context.Context) string {
	return "each value must be a valid email address"
}

func (v emailListValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (emailListValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var emails []string
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &emails, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for index, email := range emails {
		if _, err := mail.ParseAddress(email); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(index),
				"Invalid email address",
				err.Error(),
			)
		}
	}
}
