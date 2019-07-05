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
	DetectorAppPath = "/detector/"
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
		Exists: detectorExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
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
		teams := []string{}
		for _, t := range val.([]interface{}) {
			teams = append(teams, t.(string))
		}
		cudr.Teams = teams
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

		switch vars[0] {
		case "Email":
			item["email"] = vars[1]
		case "PagerDuty", "BigPanda", "Office365", "ServiceNow", "XMatters":
			item["credentialId"] = vars[1]
		case "Slack":
			item["credentialId"] = vars[1]
			item["channel"] = vars[2]
		case "Webhook":
			item["secret"] = vars[1]
			item["url"] = vars[2]
		case "Team", "TeamEmail":
			item["team"] = vars[1]
		case "Opsgenie":
			item["credentialId"] = vars[1]
			item["credentialName"] = vars[2]
			item["responderName"] = vars[3]
			item["responderId"] = vars[4]
			item["responderType"] = vars[5]
		case "VictorOps":
			item["credentialId"] = vars[1]
			item["routingKey"] = vars[2]
		}

		notifications_list[i] = item
	}

	return notifications_list
}

func getNotifyStringFromAPI(notification map[string]interface{}) (string, error) {
	nt, ok := notification["type"].(string)
	if !ok {
		return "", fmt.Errorf("Missing type field in notification body")
	}
	switch nt {
	case "Email":
		email, ok := notification["email"].(string)
		if !ok {
			return "", fmt.Errorf("Missing email field from Email body")
		}
		return fmt.Sprintf("%s,%s", nt, email), nil
	case "Opsgenie":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		credName, ok := notification["credentialName"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialName field from notification body")
		}
		respName, ok := notification["responderName"].(string)
		if !ok {
			return "", fmt.Errorf("Missing responderName field from notification body")
		}
		respId, ok := notification["responderId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing responderId field from notification body")
		}
		respType, ok := notification["responderType"].(string)
		if !ok {
			return "", fmt.Errorf("Missing responderType field from notification body")
		}
		return fmt.Sprintf("%s,%s,%s,%s,%s,%s", nt, cred, credName, respName, respId, respType), nil

	case "PagerDuty", "BigPanda", "Office365", "ServiceNow", "XMatters":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		return fmt.Sprintf("%s,%s", nt, cred), nil
	case "Slack":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		channel, ok := notification["channel"].(string)
		if !ok {
			return "", fmt.Errorf("Missing channel field from notification body")
		}
		return fmt.Sprintf("%s,%s,%s", nt, cred, channel), nil
	case "Team", "TeamEmail":
		team, ok := notification["team"].(string)
		if !ok {
			return "", fmt.Errorf("Missing team field from notification body")
		}
		return fmt.Sprintf("%s,%s", nt, team), nil
	case "VictorOps":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		routing, ok := notification["routingKey"].(string)
		if !ok {
			return "", fmt.Errorf("Missing routing key from notification body")
		}
		return fmt.Sprintf("%s,%s,%s", nt, cred, routing), nil
	case "Webhook":
		cred, ok := notification["credentialId"].(string)
		if !ok {
			return "", fmt.Errorf("Missing credentialId field from notification body")
		}
		secret, ok := notification["secret"].(string)
		if !ok {
			return "", fmt.Errorf("Missing secret field from notification body")
		}
		url, ok := notification["url"].(string)
		if !ok {
			return "", fmt.Errorf("Missing url field from notification body")
		}
		return fmt.Sprintf("%s,%s,%s,%s", nt, cred, secret, url), nil
	}

	return "", nil
}

func detectorCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDetector(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Detector Payload: %s", string(debugOutput))

	det, err := config.Client.CreateDetector(payload)
	if err != nil {
		return err
	}
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, DetectorAppPath+det.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(det.Id)

	return detectorAPIToTF(d, det)
}

func detectorExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetDetector(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "Bad status 404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func detectorRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	det, err := config.Client.GetDetector(d.Id())
	if err != nil {
		if strings.HasPrefix(err.Error(), "Bad status 404") {
			d.SetId("")
		}
		return err
	}
	return detectorAPIToTF(d, det)
}

func detectorAPIToTF(d *schema.ResourceData, det *detector.Detector) error {
	debugOutput, _ := json.Marshal(det)
	log.Printf("[DEBUG] SignalFx: Got Detector to enState: %s", string(debugOutput))

	if err := d.Set("name", det.Name); err != nil {
		return err
	}
	if err := d.Set("description", det.Description); err != nil {
		return err
	}
	if err := d.Set("program_text", det.ProgramText); err != nil {
		return err
	}
	// We divide by 1000 because the API uses millis, but this provider uses
	// seconds
	if err := d.Set("max_delay", det.MaxDelay/1000); err != nil {
		return err
	}
	if err := d.Set("teams", det.Teams); err != nil {
		return err
	}
	viz := det.VisualizationOptions
	if viz != nil {
		if err := d.Set("show_data_markers", viz.ShowDataMarkers); err != nil {
			return err
		}
		if err := d.Set("show_event_lines", viz.ShowEventLines); err != nil {
			return err
		}
		if err := d.Set("disable_sampling", viz.DisableSampling); err != nil {
			return err
		}

		tr := viz.Time
		// We divide by 1000 because the API uses millis, but this provider uses
		// seconds
		if err := d.Set("time_range", tr.Range/1000); err != nil {
			return err
		}
		if err := d.Set("start_time", tr.Start); err != nil {
			return err
		}
		if err := d.Set("end_time", tr.End); err != nil {
			return err
		}
	}

	rules := make([]map[string]interface{}, len(det.Rules))
	for i, r := range det.Rules {
		rule := make(map[string]interface{})
		rule["severity"] = r.Severity
		rule["detect_label"] = r.DetectLabel
		rule["description"] = r.Description

		notifications := make([]string, len(r.Notifications))
		for i, not := range r.Notifications {
			tfNot, err := getNotifyStringFromAPI(not)
			if err != nil {
				return err
			}
			notifications[i] = tfNot
		}
		rule["notifications"] = notifications
		rule["disabled"] = r.Disabled
		rule["parameterized_body"] = r.ParameterizedBody
		rule["parameterized_subject"] = r.ParameterizedSubject
		rule["runbook_url"] = r.RunbookUrl
		rule["tip"] = r.Tip
		rules[i] = rule
	}
	if err := d.Set("rule", rules); err != nil {
		return err
	}

	return nil
}

func detectorUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDetector(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Detector Payload: %s", string(debugOutput))

	det, err := config.Client.UpdateDetector(d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Detector Response: %v", det)
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, DetectorAppPath+det.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(det.Id)
	return detectorAPIToTF(d, det)
}

func detectorDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteDetector(d.Id())
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
