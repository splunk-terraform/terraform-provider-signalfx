package autoarchiveexemptmetric

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

const (
	ResourceName = "signalfx_automated_archival_exempt_metric"
)

func NewResource() *schema.Resource {
	return &schema.Resource{
		SchemaFunc:    newSchema,
		ReadContext:   resourceRead,
		CreateContext: resourceCreate,
		DeleteContext: resourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	exempt_metrics, err := sfx.GetExemptMetrics(ctx)
	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	return tfext.AsErrorDiagnostics(encodeTerraform(exempt_metrics, data))
}

func resourceCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	details, err := decodeTerraform(data)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	exempt_metrics, err := sfx.CreateExemptMetrics(ctx, details)
	if err != nil {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
	}

	// Set the ID of the resource to the string representation of the ids of all the exempt metrics
	if len(*exempt_metrics) == 0 {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, errors.New("no exempt metrics found after creation"), data))
	}
	// Join the IDs of the exempt metrics into a single string, separated by commas
	ids := make([]string, len(*exempt_metrics))
	for i, metric := range *exempt_metrics {
		ids[i] = *metric.Id
	}
	data.SetId(strings.Join(ids, ","))

	return tfext.AsErrorDiagnostics(encodeTerraform(exempt_metrics, data))
}

func resourceDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	sfx, err := pmeta.LoadClient(ctx, meta)
	if err != nil {
		return tfext.AsErrorDiagnostics(err)
	}

	if len(data.State().ID) == 0 {
		return tfext.AsErrorDiagnostics(common.HandleError(ctx, errors.New("no id found for the resource"), data))
	}
	ids := strings.Split(data.State().ID, ",")

	exemptMetricDeleteRequest := &automated_archival.ExemptMetricDeleteRequest{
		Ids: ids,
	}
	err = sfx.DeleteExemptMetrics(ctx, exemptMetricDeleteRequest)
	if err == nil {
		data.SetId("") // Clear the ID to indicate deletion
	}
	return tfext.AsErrorDiagnostics(common.HandleError(ctx, err, data))
}
