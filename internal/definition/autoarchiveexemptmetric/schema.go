package autoarchiveexemptmetric

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
	"go.uber.org/multierr"
)

var (
	exemptMetricSchema = map[string]*schema.Schema{
		"creator": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ID of the creator of the automated archival setting",
		},
		"last_updated_by": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "ID of user who last updated the automated archival setting",
		},
		"created": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Timestamp of when the automated archival setting was created",
		},
		"last_updated": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Timestamp of when the automated archival setting was last updated",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the metric to be exempted from automated archival",
		},
	}
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"exempt_metrics": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "List of metrics to be exempted from automated archival",
			Elem: &schema.Resource{
				Schema: exemptMetricSchema,
			},
			ForceNew: true,
		},
	}
}

func decodeTerraform(data *schema.ResourceData) (*[]automated_archival.ExemptMetric, error) {
	exempt_metrics := &[]automated_archival.ExemptMetric{}
	if v, ok := data.GetOk("exempt_metrics"); ok {
		for _, item := range v.([]any) {
			if itemMap, ok := item.(map[string]any); ok {
				exempt_metric := automated_archival.ExemptMetric{
					Name: itemMap["name"].(string),
				}

				if creator, ok := itemMap["creator"]; ok && creator != nil {
					str := creator.(string)
					exempt_metric.Creator = &str
				}

				if lastUpdatedBy, ok := itemMap["last_updated_by"]; ok && lastUpdatedBy != nil {
					str := lastUpdatedBy.(string)
					exempt_metric.LastUpdatedBy = &str
				}

				if created, ok := itemMap["created"]; ok && created != nil {
					val := int64(created.(int))
					exempt_metric.Created = &val
				}

				if lastUpdated, ok := itemMap["last_updated"]; ok && lastUpdated != nil {
					val := int64(lastUpdated.(int))
					exempt_metric.LastUpdated = &val
				}

				*exempt_metrics = append(*exempt_metrics, exempt_metric)
			}
		}
	}
	return exempt_metrics, nil
}

func encodeTerraform(exempt_metrics *[]automated_archival.ExemptMetric, data *schema.ResourceData) error {
	if exempt_metrics == nil {
		data.Set("exempt_metrics", nil)
		return nil
	}

	exemptMetricsList := make([]map[string]any, len(*exempt_metrics))
	for i, metric := range *exempt_metrics {
		exemptMetricsList[i] = map[string]any{
			"name": &metric.Name,
		}
		if metric.Creator != nil {
			exemptMetricsList[i]["creator"] = *metric.Creator
		}
		if metric.LastUpdatedBy != nil {
			exemptMetricsList[i]["last_updated_by"] = *metric.LastUpdatedBy
		}
		if metric.Created != nil {
			exemptMetricsList[i]["created"] = *metric.Created
		}
		if metric.LastUpdated != nil {
			exemptMetricsList[i]["last_updated"] = *metric.LastUpdated
		}
	}

	errs := multierr.Combine(
		data.Set("exempt_metrics", exemptMetricsList),
	)

	return errs
}
