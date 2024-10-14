// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dimension

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const (
	DataSourceName = "signalfx_dimension_values"
	// PageLimit is the default number of dimensions
	// returned within one response
	PageLimit = 1000
)

func NewDataSource() *schema.Resource {
	return &schema.Resource{
		SchemaFunc:    newSchema,
		ReadContext:   readDimensions,
		SchemaVersion: 1,
	}
}

func readDimensions(ctx context.Context, rd *schema.ResourceData, meta any) (issues diag.Diagnostics) {
	client, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	var (
		query   = rd.Get("query").(string)
		orderby = rd.Get("order_by").(string)
		limit   = slices.Min([]int{rd.Get("limit").(int), PageLimit})

		count   int
		results = make([]string, 0, limit)
	)

	for offset := 0; offset < limit; offset += PageLimit {
		tflog.Debug(ctx, "Performing dimension search operation", tfext.NewLogFields().
			Field("limit", limit).
			Field("offset", offset).
			Field("order_by", orderby),
		)

		resp, err := client.SearchDimension(ctx, query, orderby, limit, offset)
		if err != nil {
			return tfext.AsErrorDiagnostics(err)
		}

		count = int(resp.Count)
		for _, dim := range resp.Results {
			results = append(results, dim.Value)
		}

	}

	rd.SetId(query)
	if err := rd.Set("values", results); err != nil {
		issues = tfext.AppendDiagnostics(issues, tfext.AsErrorDiagnostics(err)...)
	}

	if count > limit {
		issues = tfext.AppendDiagnostics(issues, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Number of matched results exceeds allowed returned limit, values truncated",
			Detail:   "Adjust the query to be more selective or increase the limit to avoid this issue",
		})
	}

	return issues
}
