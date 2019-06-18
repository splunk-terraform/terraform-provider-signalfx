package signalfx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	detector "github.com/signalfx/signalfx-go/detector"
)

const (
	DETECTOR_API_PATH = "/v2/detector"
	DETECTOR_APP_PATH = "/detector/"
)

func detectorResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the detector",
			},
			"program_text": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Signalflow program text for the detector. More info at \"https://developers.signalfx.com/docs/signalflow-overview\"",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the detector",
			},
			"max_delay": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "How long (in seconds) to wait for late datapoints. Max value 900s (15m)",
				ValidateFunc: validateMaxDelayValue,
			},
			"show_data_markers": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) When true, markers will be drawn for each datapoint within the visualization.",
			},
			"show_event_lines": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) When true, vertical lines will be drawn for each triggered event within the visualization.",
			},
			"disable_sampling": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "(false by default) When false, samples a subset of the output MTS in the visualization.",
			},
			"time_range": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				Description:   "Seconds to display in the visualization. This is a rolling range from the current time. Example: 8600 = `-1h`",
				ConflictsWith: []string{"start_time", "end_time"},
			},
			"start_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"time_range"},
				Description:   "Seconds since epoch. Used for visualization",
			},
			"end_time": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"time_range"},
				Description:   "Seconds since epoch. Used for visualization",
			},
			"tags": &schema.Schema{
				Type:        schema.TypeList,
				Deprecated:  "signalfx_detector.tags is being removed in the next release",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the detector",
			},
			"teams": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Team IDs to associate the detector to",
			},
			"rule": &schema.Schema{
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Set of rules used for alerting",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"severity": &schema.Schema{
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateSeverity,
							Description:  "The severity of the rule, must be one of: Critical, Warning, Major, Minor, Info",
						},
						"detect_label": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							Description: "A detect label which matches a detect label within the program text",
						},
						"description": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Description of the rule",
						},
						"notifications": &schema.Schema{
							Type:        schema.TypeList,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "List of strings specifying where notifications will be sent when an incident occurs. See https://developers.signalfx.com/v2/docs/detector-model#notifications-models for more info",
						},
						"disabled": &schema.Schema{
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "(default: false) When true, notifications and events will not be generated for the detect label",
						},
						"parameterized_body": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Custom notification message body when an alert is triggered. See https://developers.signalfx.com/v2/reference#detector-model for more info",
						},
						"parameterized_subject": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Custom notification message subject when an alert is triggered. See https://d    evelopers.signalfx.com/v2/reference#detector-model for more info",
						},
						"runbook_url": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "URL of page to consult when an alert is triggered",
						},
						"tip": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Plain text suggested first course of action, such as a command to execute.",
						},
					},
				},
				Set: resourceRuleHash,
			},
			"last_updated": &schema.Schema{
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Latest timestamp the resource was updated",
			},
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the detector",
			},
		},

		Create: detectorCreate,
		Read:   detectorRead,
		Update: detectorUpdate,
		Delete: detectorDelete,
	}
}

/*
  Use Resource object to construct json payload in order to create a detector
*/
func getPayloadDetector(d *schema.ResourceData) (*detector.CreateUpdateDetectorRequest, error) {

	tfRules := d.Get("rule").(*schema.Set).List()
	rulesList := make([]detector.Rule, len(tfRules))
	for i, tfRule := range tfRules {
		tfRule := tfRule.(map[string]interface{})
		rule := detector.Rule{
			Description: tfRule["description"].(string),
			DetectLabel: tfRule["detect_label"].(string),
			Disabled:    tfRule["disabled"].(bool),
		}

		tfSev := tfRule["severity"].(string)
		sev := detector.INFO
		switch tfSev {
		case "Critical":
			sev = detector.CRITICAL
		case "Warning":
			sev = detector.WARNING
		case "Major":
			sev = detector.MAJOR
		case "Minor":
			sev = detector.MINOR
		case "Info":
			sev = detector.INFO
		}
		rule.Severity = sev

		if val, ok := tfRule["parameterized_body"]; ok {
			rule.ParameterizedBody = val.(string)
		}

		if val, ok := tfRule["parameterized_subject"]; ok {
			rule.ParameterizedSubject = val.(string)
		}

		if val, ok := tfRule["runbook_url"]; ok {
			rule.RunbookUrl = val.(string)
		}

		if val, ok := tfRule["tip"]; ok {
			rule.Tip = val.(string)
		}

		if notifications, ok := tfRule["notifications"]; ok {
			notify := getNotifications(notifications.([]interface{}))
			rule.Notifications = notify
		}
		rulesList[i] = rule
	}

	cudr := &detector.CreateUpdateDetectorRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProgramText: d.Get("program_text").(string),
		Rules:       rulesList,
	}

	if val, ok := d.GetOk("max_delay"); ok {
		cudr.MaxDelay = int32(val.(int) * 1000)
	}

	cudr.VisualizationOptions = getVisualizationOptionsDetector(d)

	if val, ok := d.GetOk("teams"); ok {
		cudr.Teams = val.([]string)
	}

	if val, ok := d.GetOk("tags"); ok {
		cudr.Tags = val.([]string)
	}
	return cudr, nil
}

func getVisualizationOptionsDetector(d *schema.ResourceData) *detector.Visualization {
	viz := detector.Visualization{}

	if val, ok := d.GetOk("show_data_markers"); ok {
		viz.ShowDataMarkers = val.(bool)
	}
	if val, ok := d.GetOk("show_event_lines"); ok {
		viz.ShowEventLines = val.(bool)
	}
	if val, ok := d.GetOk("disable_sampling"); ok {
		viz.DisableSampling = val.(bool)
	}

	if val, ok := d.GetOk("time_range"); ok {
		tr := &detector.Time{}
		tr.Range = int32(val.(int)) * 1000
		tr.Type = "relative"
		viz.Time = tr
	}
	if val, ok := d.GetOk("start_time"); ok {
		tr := &detector.Time{}
		tr.Type = "absolute"
		tr.Start = val.(int32) * 1000
		if val, ok := d.GetOk("end_time"); ok {
			tr.End = val.(int32) * 1000
		}
		viz.Time = tr
	}

	if (detector.Visualization{}) == viz {
		// Return a nil ptr so we don't serialize nothing
		return nil
	}

	return &viz
}

/*
  Get list of notifications from Resource object (a list of strings), and return a list of notification maps
*/
func getNotifications(tf_notifications []interface{}) []map[string]interface{} {
	notifications_list := make([]map[string]interface{}, len(tf_notifications))
	for i, tf_notification := range tf_notifications {
		vars := strings.Split(tf_notification.(string), ",")
		item := make(map[string]interface{})
		item["type"] = vars[0]

		if vars[0] == "Email" {
			item["email"] = vars[1]
		} else if vars[0] == "PagerDuty" {
			item["credentialId"] = vars[1]
		} else if vars[0] == "Slack" {
			item["credentialId"] = vars[1]
			item["channel"] = vars[2]
		} else if vars[0] == "Webhook" {
			item["secret"] = vars[1]
			item["url"] = vars[2]
		} else if vars[0] == "Team" || vars[0] == "TeamEmail" {
			item["team"] = vars[1]
		} else if vars[0] == "Opsgenie" {
			item["credentialId"] = vars[1]
			item["credentialName"] = vars[2]
			item["responderName"] = vars[3]
			item["responderId"] = vars[4]
			item["responderType"] = vars[5]
		}

		notifications_list[i] = item
	}

	return notifications_list
}

func detectorCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDetector(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	payload2, _ := json.Marshal(payload)
	log.Printf("[DEBUG] Payload: %s", string(payload2))

	detector, err := config.Client.CreateDetector(payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, DETECTOR_APP_PATH+d.Id())
	if err != nil {
		return err
	}
	d.Set("url", appURL)
	d.SetId(detector.Id)
	return nil
}

func detectorRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	detector, err := config.Client.GetDetector(d.Id())
	if err != nil {
		return err
	}

	if err := d.Set("name", detector.Name); err != nil {
		return nil
	}
	if err := d.Set("description", detector.Description); err != nil {
		return nil
	}
	if err := d.Set("program_text", detector.ProgramText); err != nil {
		return nil
	}
	// We divide by 1000 because the API uses millis, but this provider uses
	// seconds
	if err := d.Set("max_delay", detector.MaxDelay/1000); err != nil {
		return nil
	}
	if err := d.Set("teams", detector.Teams); err != nil {
		return nil
	}
	if err := d.Set("tags", detector.Tags); err != nil {
		return nil
	}

	viz := detector.VisualizationOptions
	if viz != nil {
		if err := d.Set("show_data_markers", viz.ShowDataMarkers); err != nil {
			return nil
		}
		if err := d.Set("show_event_lines", viz.ShowEventLines); err != nil {
			return nil
		}
		if err := d.Set("disable_sampling", viz.DisableSampling); err != nil {
			return nil
		}

		tr := viz.Time
		// We divide by 1000 because the API uses millis, but this provider uses
		// seconds
		if err := d.Set("time_range", tr.Range/1000); err != nil {
			return nil
		}
		if err := d.Set("start_time", tr.Start); err != nil {
			return nil
		}
		if err := d.Set("end_time", tr.End); err != nil {
			return nil
		}
		if err := d.Set("type", tr.Type); err != nil {
			return nil
		}
	}

	rules := make([]map[string]interface{}, len(detector.Rules))
	for i, r := range detector.Rules {
		rule := make(map[string]interface{})
		rule["severity"] = r.Severity
		rule["detect_label"] = r.DetectLabel
		rule["description"] = r.Description
		rule["notifications"] = r.Notifications
		rule["disabled"] = r.Disabled
		rule["parameterized_body"] = r.ParameterizedBody
		rule["parameterized_subject"] = r.ParameterizedSubject
		rule["runbook_url"] = r.RunbookUrl
		rule["tip"] = r.Tip
		rules[i] = rule
	}
	d.Set("rule", rules)

	return nil
}

func detectorUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDetector(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	detector, err := config.Client.UpdateDetector(d.Id(), payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, DETECTOR_APP_PATH+d.Id())
	if err != nil {
		return err
	}
	d.Set("url", appURL)
	d.SetId(detector.Id)

	return nil
	// return resourceUpdate(url, config.AuthToken, payload, d)
}

func detectorDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	path := fmt.Sprintf("%s/%s", DETECTOR_API_PATH, d.Id())
	url, err := buildURL(config.APIURL, path, map[string]string{})
	if err != nil {
		return fmt.Errorf("[SignalFx] Error constructing API URL: %s", err.Error())
	}

	return resourceDelete(url, config.AuthToken, d)
}

/*
   Hashing function for rule substructure of the detector resource, used in determining state changes.
*/
func resourceRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["description"]))
	buf.WriteString(fmt.Sprintf("%s-", m["severity"]))
	buf.WriteString(fmt.Sprintf("%s-", m["detect_label"]))
	buf.WriteString(fmt.Sprintf("%s-", m["disabled"]))

	// loop through optional rule attributes
	var optional_rule_keys = []string{"parameterized_body", "parameterized_subject", "runbook_url", "tip"}

	for _, key := range optional_rule_keys {
		if val, ok := m[key]; ok {
			buf.WriteString(fmt.Sprintf("%s-", val))
		}
	}

	// Sort the notifications so that we generate a consistent hash
	if v, ok := m["notifications"]; ok {
		notifications := v.([]interface{})
		s_notifications := make([]string, len(notifications))
		for i, raw := range notifications {
			s_notifications[i] = raw.(string)
		}
		sort.Strings(s_notifications)

		for _, notification := range s_notifications {
			buf.WriteString(fmt.Sprintf("%s-", notification))
		}
	}

	return hashcode.String(buf.String())
}

/*
  Validates the severity field against a list of allowed words.
*/
func validateSeverity(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	allowedWords := []string{"Critical", "Major", "Minor", "Warning", "Info"}
	for _, word := range allowedWords {
		if value == word {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; must be one of: %s", value, strings.Join(allowedWords, ", ")))
	return
}
