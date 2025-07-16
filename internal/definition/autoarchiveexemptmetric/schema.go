// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package autoarchiveexemptmetric

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"exempt_metrics": {
			Type:        schema.TypeList,
			Required:    true,
			Description: "List of metrics to be exempted from automated archival",
			Elem: &schema.Resource{
				Schema: getExemptMetricSchema(),
			},
			ForceNew: true,
			MinItems: 1,
		},
	}
}

func getExemptMetricSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
}

func decodeTerraform(data *schema.ResourceData) (*[]automated_archival.ExemptMetric, error) {
	exempt_metrics := &[]automated_archival.ExemptMetric{}
	v, ok := data.GetOk("exempt_metrics")
	if !ok {
		return exempt_metrics, nil
	}

	for _, item := range v.([]any) {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		exempt_metric := automated_archival.ExemptMetric{
			Name: itemMap["name"].(string),
		}

		// Only set if field exists and contains a value
		if creator, ok := itemMap["creator"].(string); ok && creator != "" {
			exempt_metric.Creator = &creator
		}

		if lastUpdatedBy, ok := itemMap["last_updated_by"].(string); ok && lastUpdatedBy != "" {
			exempt_metric.LastUpdatedBy = &lastUpdatedBy
		}

		if created, ok := itemMap["created"].(int); ok && created != 0 {
			val := int64(created)
			exempt_metric.Created = &val
		}

		if lastUpdated, ok := itemMap["last_updated"].(int); ok && lastUpdated != 0 {
			val := int64(lastUpdated)
			exempt_metric.LastUpdated = &val
		}

		*exempt_metrics = append(*exempt_metrics, exempt_metric)
	}
	return exempt_metrics, nil
}

func encodeTerraform(exempt_metrics *[]automated_archival.ExemptMetric, data *schema.ResourceData) error {
	if exempt_metrics == nil || len(*exempt_metrics) == 0 {
		err := data.Set("exempt_metrics", nil)
		if err != nil {
			return fmt.Errorf("failed to set exempt_metrics: %w", err)
		}
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

	return data.Set("exempt_metrics", exemptMetricsList)
}
