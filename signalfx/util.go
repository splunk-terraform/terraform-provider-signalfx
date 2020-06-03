package signalfx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	chart "github.com/signalfx/signalfx-go/chart"
)

const (
	// Workaround for Signalfx bug related to post processing and lastUpdatedTime
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

func buildURL(apiURL string, path string, params map[string]string) (string, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}

	u.Path = path

	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
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

func chartExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetChart(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

/*
  Utility function that wraps http calls to SignalFx
*/
func sendRequest(method string, url string, token string, payload []byte) (int, []byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-SF-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return -1, nil, fmt.Errorf("Failed sending %s request to Signalfx: %s", method, err.Error())
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("Failed reading response body from %s request: %s", method, err.Error())
	}

	return resp.StatusCode, body, nil
}

func expandStringSetToSlice(set *schema.Set) []string {
	result := make([]string, set.Len(), set.Len())
	for i, s := range set.List() {
		result[i] = s.(string)
	}
	return result
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
  Send a GET to get the current state of the resource. It just checks if the lastUpdated timestamp is
  later than the timestamp saved in the resource. If so, the resource has been modified in some way
  in the UI, and should be recreated. This is signaled by setting synced to false, meaning if synced is set to
  true in the tf configuration, it will update the resource to achieve the desired state.
*/
func resourceRead(url string, sfxToken string, d *schema.ResourceData) error {
	status_code, resp_body, err := sendRequest("GET", url, sfxToken, nil)
	if status_code == 200 {
		mapped_resp := map[string]interface{}{}
		err = json.Unmarshal(resp_body, &mapped_resp)
		if err != nil {
			return fmt.Errorf("Failed unmarshaling for the resource %s during read: %s", d.Get("name"), err.Error())
		}
		// This implies the resource was modified in the Signalfx UI and therefore it is not synced.
		last_updated := mapped_resp["lastUpdated"].(float64)
		if last_updated > (d.Get("last_updated").(float64) + OFFSET) {
			d.Set("synced", false)
			d.Set("last_updated", last_updated)
		}
	} else {
		if status_code == 404 && strings.Contains(string(resp_body), " not found") {
			// This implies that the resouce was deleted in the Signalfx UI and therefore we need to recreate it
			d.SetId("")
		} else {
			return fmt.Errorf("For the resource '%s' SignalFx returned status %d: \n%s", d.Get("name"), status_code, resp_body)
		}
	}

	return nil
}

/*
  Fetches payload specified in terraform configuration and creates a resource
*/
func resourceCreate(url string, sfxToken string, payload []byte, d *schema.ResourceData) error {
	status_code, resp_body, err := sendRequest("POST", url, sfxToken, payload)
	if status_code == 200 {
		mapped_resp := map[string]interface{}{}
		err = json.Unmarshal(resp_body, &mapped_resp)
		if err != nil {
			return fmt.Errorf("Failed unmarshaling for the resource %s during creation: %s", d.Get("name"), err.Error())
		}
		d.SetId(fmt.Sprintf("%s", mapped_resp["id"].(string)))
		d.Set("last_updated", mapped_resp["lastUpdated"].(float64))
		d.Set("synced", true)
	} else {
		return fmt.Errorf("For the resource %s SignalFx returned status %d: \n%s", d.Get("name"), status_code, resp_body)
	}
	return nil
}

/*
  Fetches payload specified in terraform configuration and creates chart
*/
func resourceUpdate(url string, sfxToken string, payload []byte, d *schema.ResourceData) error {
	status_code, resp_body, err := sendRequest("PUT", url, sfxToken, payload)
	if status_code == 200 {
		mapped_resp := map[string]interface{}{}
		err = json.Unmarshal(resp_body, &mapped_resp)
		if err != nil {
			return fmt.Errorf("Failed unmarshaling for the resource %s during creation: %s", d.Get("name"), err.Error())
		}
		// If the resource was updated successfully with configs, it is now synced with Signalfx
		d.Set("synced", true)
		d.Set("last_updated", mapped_resp["lastUpdated"].(float64))
	} else {
		return fmt.Errorf("For the resource '%s' SignalFx returned status %d: \n%s", d.Get("name"), status_code, resp_body)
	}
	return nil
}

/*
  Deletes a resource.  If the resource does not exist, it will receive a 404, and carry on as usual.
*/
func resourceDelete(url string, sfxToken string, d *schema.ResourceData) error {
	status_code, resp_body, err := sendRequest("DELETE", url, sfxToken, nil)
	if err != nil {
		return fmt.Errorf("Failed deleting resource  %s: %s", d.Get("name"), err.Error())
	}
	if status_code < 400 || status_code == 404 {
		d.SetId("")
	} else {
		return fmt.Errorf("For the resource  %s SignalFx returned status %d: \n%s", d.Get("name"), status_code, resp_body)
	}
	return nil
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
	Util method to validate SignalFx specific string format.
*/
func validateSignalfxRelativeTime(v interface{}, k string) (we []string, errors []error) {
	ts := v.(string)

	r, _ := regexp.Compile("-([0-9]+)[mhdw]")
	if !r.MatchString(ts) {
		errors = append(errors, fmt.Errorf("%s not allowed. Please use milliseconds from epoch or SignalFx time syntax (e.g. -5m, -1h)", ts))
	}
	return
}

/*
*  Util method to convert from Signalfx string format to milliseconds
 */
func fromRangeToMilliSeconds(timeRange string) (int, error) {
	r := regexp.MustCompile("-([0-9]+)([mhdw])")
	ss := r.FindStringSubmatch(timeRange)
	var c int
	switch ss[2] {
	case "m":
		c = 60 * 1000
	case "h":
		c = 60 * 60 * 1000
	case "d":
		c = 24 * 60 * 60 * 1000
	case "w":
		c = 7 * 24 * 60 * 60 * 1000
	default:
		c = 1
	}
	val, err := strconv.Atoi(ss[1])
	if err != nil {
		return -1, err
	}
	return val * c, nil
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
