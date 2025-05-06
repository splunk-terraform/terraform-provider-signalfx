// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package organization

import (
	"context"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const DataSourceName = "signalfx_organization_members"

func NewDataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Allows for members to be queried and used as part of other resources. Requires the supplied token to have Admin priviledges.",
		SchemaFunc:  newSchema,
		ReadContext: datasourceRead,
	}
}

func datasourceRead(ctx context.Context, rd *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	var (
		hasher = fnv.New64()
		users  []any
		limit  = 1000
	)

	for _, email := range convert.SliceAll(rd.Get("emails").([]any), convert.ToString) {
		_, _ = fmt.Fprint(hasher, email)
		for offset := 0; ; offset += limit {
			results, err := sfx.GetOrganizationMembers(ctx, limit, fmt.Sprintf("email:%s", email), offset, "-sf_timestamp")
			if err != nil {
				return tfext.AsErrorDiagnostics(err)
			}

			for _, u := range results.Results {
				tflog.Debug(ctx, "Retrieved user details", tfext.NewLogFields().JSON("user", u))
				users = append(users, u.UserId)
			}

			if offset >= int(results.Count) {
				break
			}
		}
	}
	rd.SetId(strconv.FormatUint(hasher.Sum64(), 36))

	return tfext.AsErrorDiagnostics(rd.Set("users", users))
}
