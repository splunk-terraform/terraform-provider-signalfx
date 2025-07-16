// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	chart "github.com/signalfx/signalfx-go/chart"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
)

const (
	// Workaround for Splunk Observability Cloud bug related to post processing and lastUpdatedTime
	OFFSET         = 10000.0
	CHART_API_PATH = "/v2/chart"
	CHART_APP_PATH = "/chart/"
)

func buildAppURL(appURL string, fragment string) (string, error) {
	// Include a trailing slash, as without this Go doesn't add one for the fragment and that seems to be a required part of the url
	u, err := url.Parse(appURL + "/")
	if err != nil {
		return "", err
	}
	// The URL is actually a fragment, so use that instead of Path
	u.Fragment = fragment
	return u.String(), nil
}

func flattenStringSliceToSet(slice []string) *schema.Set {
	if len(slice) < 1 {
		return nil
	}
	var values []interface{}
	for _, v := range slice {
		if v != "" { // Ignore empty strings
			values = append(values, v)
		}
	}
	return schema.NewSet(schema.HashString, values)
}

/*
Validates that sort_by field start with either + or -.
*/
func validateSortBy(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	if !strings.HasPrefix(value, "+") && !strings.HasPrefix(value, "-") {
		errors = append(errors, fmt.Errorf("%s not allowed; must start either with + or - (ascending or descending)", value))
	}
	return
}

// Deprecated: Use `convert.SchemaListAll(d.Get("color_scale"), convert.ToChartSecondaryVisualization)` instead
func getColorScaleOptions(d *schema.ResourceData) []*chart.SecondaryVisualization {
	return convert.SchemaListAll(d.Get("color_scale"), convert.ToChartSecondaryVisualization)
}

func getValueUsingMaxFloatAsDefault(v float64) *float64 {
	if v >= math.MaxFloat32 || v <= -math.MaxFloat32 {
		return nil
	}
	vf := float64(v)
	return &vf
}

/*
Util method to get Legend Chart Options.
*/
func getLegendOptions(d *schema.ResourceData) *chart.DataTableOptions {
	var options *chart.DataTableOptions
	if properties, ok := d.GetOk("legend_fields_to_hide"); ok {
		properties := properties.(*schema.Set).List()

		propertiesOpts := make([]*chart.DataTableOptionsFields, len(properties))
		for i, property := range properties {
			property := property.(string)
			if property == "metric" {
				property = "sf_originatingMetric"
			} else if property == "plot_label" || property == "Plot Label" {
				property = "sf_metric"
			}
			item := &chart.DataTableOptionsFields{
				Property: property,
				Enabled:  false,
			}
			propertiesOpts[i] = item
		}
		if len(propertiesOpts) > 0 {
			options = &chart.DataTableOptions{
				Fields: propertiesOpts,
			}
		}
	}
	return options
}

/*
Util method to get Legend Chart Options for fields
*/
func getLegendFieldOptions(d *schema.ResourceData) *chart.DataTableOptions {
	if fields, ok := d.GetOk("legend_options_fields"); ok {
		fields := fields.([]interface{})
		if len(fields) > 0 {
			legendOptions := make([]*chart.DataTableOptionsFields, len(fields))
			for i, f := range fields {
				f := f.(map[string]interface{})
				legendOptions[i] = &chart.DataTableOptionsFields{
					Property: f["property"].(string),
					Enabled:  f["enabled"].(bool),
				}
			}
			return &chart.DataTableOptions{
				Fields: legendOptions,
			}
		}
	}
	return nil
}

func validateSecondaryVisualization(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	allowedWords := []string{"", "None", "Radial", "Linear", "Sparkline"}
	for _, word := range allowedWords {
		if value == word {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; must be one of: %s", value, strings.Join(allowedWords, ", ")))
	return
}
