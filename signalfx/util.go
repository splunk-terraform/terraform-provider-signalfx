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
)

const (
	// Workaround for Splunk Observability Cloud bug related to post processing and lastUpdatedTime
	OFFSET         = 10000.0
	CHART_API_PATH = "/v2/chart"
	CHART_APP_PATH = "/chart/"
)

type chartColor struct {
	name string
	hex  string
}

var ChartColorsSlice = []chartColor{
	{"gray", "#999999"},
	{"blue", "#0077c2"},
	{"light_blue", "#00b9ff"},
	{"navy", "#6CA2B7"},
	{"dark_orange", "#b04600"},
	{"orange", "#f47e00"},
	{"dark_yellow", "#e5b312"},
	{"magenta", "#bd468d"},
	{"cerise", "#e9008a"},
	{"pink", "#ff8dd1"},
	{"violet", "#876ff3"},
	{"purple", "#a747ff"},
	{"gray_blue", "#ab99bc"},
	{"dark_green", "#007c1d"},
	{"green", "#05ce00"},
	{"aquamarine", "#0dba8f"},
	{"red", "#ea1849"},
	{"yellow", "#ea1849"},
	{"vivid_yellow", "#ea1849"},
	{"light_green", "#acef7f"},
	{"lime_green", "#6bd37e"},
}

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

func getNameFromPaletteColorsByIndex(index int) (string, error) {
	for k, v := range PaletteColors {
		if v == index {
			return k, nil
		}
	}
	return "", fmt.Errorf("Unknown color index %d", index)
}

func getNameFromFullPaletteColorsByIndex(index int) (string, error) {
	for k, v := range FullPaletteColors {
		if v == index {
			return k, nil
		}
	}
	return "", fmt.Errorf("Unknown color index %d", index)
}

func getNameFromChartColorsByIndex(index int) (string, error) {
	for i, v := range ChartColorsSlice {
		if i == index {
			return v.name, nil
		}
	}
	return "", fmt.Errorf("Unknown color index %d", index)
}

/*
Get Color Scale Options
*/
func getColorScaleOptions(d *schema.ResourceData) []*chart.SecondaryVisualization {
	colorScale := d.Get("color_scale").(*schema.Set).List()
	return getColorScaleOptionsFromSlice(colorScale)
}

func getValueUsingMaxFloatAsDefault(v float64) *float64 {
	if v >= math.MaxFloat32 || v <= -math.MaxFloat32 {
		return nil
	}
	vf := float64(v)
	return &vf
}

func getColorScaleOptionsFromSlice(colorScale []interface{}) []*chart.SecondaryVisualization {
	item := make([]*chart.SecondaryVisualization, len(colorScale))
	if len(colorScale) == 0 {
		return item
	}
	for i := range colorScale {
		options := &chart.SecondaryVisualization{}
		scale := colorScale[i].(map[string]interface{})
		options.Gt = getValueUsingMaxFloatAsDefault(scale["gt"].(float64))
		options.Gte = getValueUsingMaxFloatAsDefault(scale["gte"].(float64))
		options.Lt = getValueUsingMaxFloatAsDefault(scale["lt"].(float64))
		options.Lte = getValueUsingMaxFloatAsDefault(scale["lte"].(float64))

		var paletteIndex *int32
		for index, thing := range ChartColorsSlice {
			if scale["color"] == thing.name {
				i := int32(index)
				paletteIndex = &i
				break
			}
		}
		if paletteIndex != nil {
			options.PaletteIndex = paletteIndex
		}
		item[i] = options
	}
	return item
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

/*
Validates the color field against a list of allowed words.
*/
func validatePerSignalColor(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	if _, ok := PaletteColors[value]; !ok {
		keys := make([]string, 0, len(PaletteColors))
		for k := range PaletteColors {
			keys = append(keys, k)
		}
		joinedColors := strings.Join(keys, ",")
		errors = append(errors, fmt.Errorf("%s not allowed; must be either %s", value, joinedColors))
	}
	return
}

func validateFullPaletteColors(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	if _, ok := FullPaletteColors[value]; !ok {
		keys := make([]string, 0, len(FullPaletteColors))
		for k := range FullPaletteColors {
			keys = append(keys, k)
		}
		joinedColors := strings.Join(keys, ",")
		errors = append(errors, fmt.Errorf("%s not allowed; must be either %s", value, joinedColors))
	}
	return
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
