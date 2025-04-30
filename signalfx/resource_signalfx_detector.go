// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

const (
	DetectorAppPath = "/detector/"
)

var (
	detectorRuleSchema = map[string]*schema.Schema{
		"severity": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validateSeverity,
			Description:  "The severity of the rule, must be one of: Critical, Warning, Major, Minor, Info",
		},
		"detect_label": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "A detect label which matches a detect label within the program text",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Description of the rule",
		},
		"notifications": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of strings specifying where notifications will be sent when an incident occurs. See https://developers.signalfx.com/v2/docs/detector-model#notifications-models for more info",
		},
		"disabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "(default: false) When true, notifications and events will not be generated for the detect label",
		},
		"parameterized_body": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Custom notification message body when an alert is triggered. See https://developers.signalfx.com/v2/reference#detector-model for more info",
		},
		"parameterized_subject": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Custom notification message subject when an alert is triggered. See https://developers.signalfx.com/v2/reference#detector-model for more info",
		},
		"runbook_url": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "URL of page to consult when an alert is triggered",
		},
		"tip": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Plain text suggested first course of action, such as a command to execute.",
		},
		"reminder_notification": {
			Optional:    true,
			Description: "Some description about reminder",
			Type:        schema.TypeList,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"interval": {
						Type:        schema.TypeInt,
						Required:    true,
						Description: "Interval in milliseconds.",
					},
					"timeout": {
						Type:        schema.TypeInt,
						Optional:    true,
						Description: "Timeout in milliseconds.",
					},
					"type": {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validateReminderType,
						Description:  "Type of the reminder notification",
					},
				},
			},
		},
	}
)

func detectorResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the detector",
			},
			"program_text": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Signalflow program text for the detector. More info at \"https://developers.signalfx.com/docs/signalflow-overview\"",
				ValidateFunc: validation.StringLenBetween(1, 50000),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the detector",
			},
			"timezone": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "UTC",
				Description: "The property value is a string that denotes the geographic region associated with the time zone, (e.g. Australia/Sydney)",
			},
			"max_delay": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				Description:  "Maximum time (in seconds) to wait for late datapoints. Max value is 900 (15m)",
				ValidateFunc: validation.IntBetween(0, 900),
			},
			"min_delay": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				Description:  "Minimum time (in seconds) for the computation to wait even if the datapoints are arriving in a timely fashion. Max value is 900 (15m)",
				ValidateFunc: validation.IntBetween(0, 900),
			},
			"show_data_markers": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "(true by default) When true, markers will be drawn for each datapoint within the visualization.",
			},
			"show_event_lines": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "(false by default) When true, vertical lines will be drawn for each triggered event within the visualization.",
			},
			"disable_sampling": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "(false by default) When false, samples a subset of the output MTS in the visualization.",
			},
			"time_range": {
				Type:          schema.TypeInt,
				Optional:      true,
				Default:       3600,
				Description:   "Seconds to display in the visualization. This is a rolling range from the current time. Example: 3600 = `-1h`. Defaults to 3600",
				ConflictsWith: []string{"start_time", "end_time"},
			},
			"start_time": {
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"time_range"},
				Description:   "Seconds since epoch. Used for visualization",
				ValidateFunc:  validation.IntAtLeast(0),
			},
			"end_time": {
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"time_range"},
				Description:   "Seconds since epoch. Used for visualization",
				ValidateFunc:  validation.IntAtLeast(0),
			},
			"tags": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the detector",
			},
			"teams": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Team IDs to associate the detector to",
			},
			"rule": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Set of rules used for alerting",
				Elem: &schema.Resource{
					Schema: detectorRuleSchema,
				},
				Set: resourceRuleHash,
			},
			"authorized_writer_teams": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Team IDs that have write access to this dashboard",
			},
			"authorized_writer_users": {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "User IDs that have write access to this dashboard",
			},
			"viz_options": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Plot-level customization options, associated with a publish statement",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The label used in the publish statement that displays the plot (metric time series data) you want to customize",
						},
						"color": {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Color to use",
							ValidateFunc: validatePerSignalColor,
						},
						"display_name": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Specifies an alternate value for the Plot Name column of the Data Table associated with the chart.",
						},
						"value_unit": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateUnitTimeChart,
							Description:  "A unit to attach to this plot. Units support automatic scaling (eg thousands of bytes will be displayed as kilobytes)",
						},
						"value_prefix": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An arbitrary prefix to display with the value of this plot",
						},
						"value_suffix": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "An arbitrary suffix to display with the value of this plot",
						},
					},
				},
			},
			"label_resolutions": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Resolutions of the detector alerts in milliseconds that indicate how often data is analyzed to determine if an alert should be triggered",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL of the detector",
			},
			"detector_origin": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "Standard",
				Description:  "Indicates how a detector was created",
				ValidateFunc: validation.StringInSlice([]string{"Standard", "AutoDetectCustomization"}, false),
			},
			"parent_detector_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ID of the parent AutoDetect detector from which this detector is customized and created. This property is required for detectors with detector_origin of type AutoDetectCustomization.",
			},
		},

		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    timeRangeV0().CoreConfigSchema().ImpliedType(),
				Upgrade: timeRangeStateUpgradeV0,
				Version: 0,
			},
		},

		CustomizeDiff: customdiff.If(validateProgramTextCondition, validateProgramText),

		Create: detectorCreate,
		Read:   detectorRead,
		Update: detectorUpdate,
		Delete: detectorDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func timeRangeV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"time_range": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func timeRangeStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {

	log.Printf("[DEBUG] SignalFx: Upgrading Detector State %v", rawState["time_range"])
	if tr, ok := rawState["time_range"].(string); ok {
		millis, err := common.FromTimeRangeToMilliseconds(tr)
		if err != nil {
			return rawState, err
		}
		rawState["time_range"] = millis / 1000
	}

	return rawState, nil
}

/*
Use Resource object to construct json payload in order to create a detector
*/
func getPayloadDetector(d *schema.ResourceData) (*detector.CreateUpdateDetectorRequest, error) {

	tfRules := d.Get("rule").(*schema.Set).List()
	rulesList := make([]*detector.Rule, len(tfRules))
	for i, tfRule := range tfRules {
		tfRule := tfRule.(map[string]interface{})
		rule, err := getDetectorRule(tfRule)
		if err != nil {
			return nil, err
		}
		rulesList[i] = rule
	}

	maxDelay := int32(d.Get("max_delay").(int) * 1000)
	minDelay := int32(d.Get("min_delay").(int) * 1000)

	tags := []string{}
	if val, ok := d.GetOk("tags"); ok {
		for _, tag := range val.(*schema.Set).List() {
			tags = append(tags, tag.(string))
		}
		log.Printf("[DEBUG] Splunk Observability Cloud the following tags will be set: %s", tags)
	}

	cudr := &detector.CreateUpdateDetectorRequest{
		Name:              d.Get("name").(string),
		Description:       d.Get("description").(string),
		TimeZone:          d.Get("timezone").(string),
		MaxDelay:          &maxDelay,
		MinDelay:          &minDelay,
		ProgramText:       d.Get("program_text").(string),
		Rules:             rulesList,
		AuthorizedWriters: &detector.AuthorizedWriters{},
		Tags:              tags,
		DetectorOrigin:    d.Get("detector_origin").(string),
		ParentDetectorId:  d.Get("parent_detector_id").(string),
	}

	authorizedTeams := []string{}
	if val, ok := d.GetOk("authorized_writer_teams"); ok {
		tfValues := val.(*schema.Set).List()
		for _, v := range tfValues {
			authorizedTeams = append(authorizedTeams, v.(string))
		}
	}
	cudr.AuthorizedWriters.Teams = authorizedTeams

	authorizedUsers := []string{}
	if val, ok := d.GetOk("authorized_writer_users"); ok {
		tfValues := val.(*schema.Set).List()
		for _, v := range tfValues {
			authorizedUsers = append(authorizedUsers, v.(string))
		}
	}
	cudr.AuthorizedWriters.Users = authorizedUsers

	cudr.VisualizationOptions = getVisualizationOptionsDetector(d)

	teams := []string{}
	if val, ok := d.GetOk("teams"); ok {
		for _, t := range val.(*schema.Set).List() {
			teams = append(teams, t.(string))
		}
	}
	cudr.Teams = teams

	return cudr, nil
}

func getDetectorRule(tfRule map[string]interface{}) (*detector.Rule, error) {
	rule := &detector.Rule{
		Description: tfRule["description"].(string),
		Disabled:    tfRule["disabled"].(bool),
	}

	if detectLabel, ok := tfRule["detect_label"]; ok {
		rule.DetectLabel = detectLabel.(string)
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
		notify, err := common.NewNotificationList(notifications.([]any))
		if err != nil {
			return nil, err
		}
		rule.Notifications = notify
	}
	return rule, nil
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
		r := int64(val.(int)) * 1000
		tr.Range = &r
		tr.Type = "relative"
		viz.Time = tr
	}
	if val, ok := d.GetOk("start_time"); ok {
		tr := &detector.Time{}
		tr.Type = "absolute"
		start := val.(int64) * 1000
		tr.Start = &start
		if val, ok := d.GetOk("end_time"); ok {
			end := val.(int64) * 1000
			tr.End = &end
		}
		viz.Time = tr
	}

	if vizOptions := getPerSignalDetectorVizOptions(d); len(vizOptions) > 0 {
		viz.PublishLabelOptions = vizOptions
	}

	return &viz
}

func detectorPublishLabelOptionsToMap(options *detector.PublishLabelOptions) (map[string]interface{}, error) {
	color := ""
	if options.PaletteIndex != nil {
		// We might not have a color, so tread lightly
		c, err := getNameFromPaletteColorsByIndex(int(*options.PaletteIndex))
		if err != nil {
			return map[string]interface{}{}, err
		}
		// Ok, we can set the color now
		color = c
	}

	return map[string]interface{}{
		"label":        options.Label,
		"display_name": options.DisplayName,
		"color":        color,
		"value_unit":   options.ValueUnit,
		"value_suffix": options.ValueSuffix,
		"value_prefix": options.ValuePrefix,
	}, nil
}

func detectorCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDetector(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	payload.Tags = common.Unique(
		pmeta.LoadProviderTags(context.Background(), meta),
		payload.Tags,
	)

	payload.Teams = pmeta.MergeProviderTeams(
		context.TODO(),
		meta,
		payload.Tags,
	)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Detector Payload: %s", string(debugOutput))

	det, err := config.Client.CreateDetector(context.TODO(), payload)
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

	return detectorRead(d, meta)
}

func detectorRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	det, err := config.Client.GetDetector(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	appURL, err := buildAppURL(config.CustomAppURL, DetectorAppPath+det.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
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
	if err := d.Set("timezone", det.TimeZone); err != nil {
		return err
	}
	// We divide by 1000 because the API uses millis, but this provider uses
	// seconds
	if det.MaxDelay != nil {
		if err := d.Set("max_delay", *det.MaxDelay/1000); err != nil {
			return err
		}
	}
	if det.MinDelay != nil {
		if err := d.Set("min_delay", *det.MinDelay/1000); err != nil {
			return err
		}
	}
	if err := d.Set("label_resolutions", det.LabelResolutions); err != nil {
		return err
	}
	if err := d.Set("tags", det.Tags); err != nil {
		return err
	}
	if err := d.Set("teams", det.Teams); err != nil {
		return err
	}
	if err := d.Set("detector_origin", det.DetectorOrigin); err != nil {
		return err
	}
	if err := d.Set("parent_detector_id", det.ParentDetectorId); err != nil {
		return err
	}

	if det.AuthorizedWriters != nil {
		aw := det.AuthorizedWriters
		if aw.Teams != nil && len(aw.Teams) > 0 {
			teams := make([]interface{}, len(aw.Teams))
			for i, v := range aw.Teams {
				teams[i] = v
			}
			if err := d.Set("authorized_writer_teams", schema.NewSet(schema.HashString, teams)); err != nil {
				return err
			}
		}
		if aw.Users != nil && len(aw.Users) > 0 {
			users := make([]interface{}, len(aw.Users))
			for i, v := range aw.Users {
				users[i] = v
			}
			if err := d.Set("authorized_writer_users", schema.NewSet(schema.HashString, users)); err != nil {
				return err
			}
		}
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
		if tr != nil {
			// We divide by 1000 because the API uses millis, but this provider uses
			// seconds
			if tr.Range != nil {
				if err := d.Set("time_range", *tr.Range/1000); err != nil {
					return err
				}
			} else {
				// Only set start/end if we didn't have a range
				if err := d.Set("start_time", tr.Start); err != nil {
					return err
				}
				if err := d.Set("end_time", tr.End); err != nil {
					return err
				}
			}
		}

		if len(viz.PublishLabelOptions) > 0 {
			plos := make([]map[string]interface{}, len(viz.PublishLabelOptions))
			for i, plo := range viz.PublishLabelOptions {
				no, err := detectorPublishLabelOptionsToMap(plo)
				if err != nil {
					return err
				}
				plos[i] = no
			}
			if err := d.Set("viz_options", plos); err != nil {
				return err
			}
		}
	}

	rules := make([]map[string]interface{}, len(det.Rules))
	for i, r := range det.Rules {
		rule, err := getTfDetectorRule(r)
		if err != nil {
			return err
		}
		rules[i] = rule
	}
	if err := d.Set("rule", rules); err != nil {
		return err
	}

	return nil
}

func getTfDetectorRule(r *detector.Rule) (map[string]interface{}, error) {
	rule := make(map[string]interface{})
	rule["severity"] = r.Severity
	rule["detect_label"] = r.DetectLabel
	rule["description"] = r.Description

	notifications := make([]string, len(r.Notifications))
	for i, not := range r.Notifications {
		tfNot, err := common.NewNotificationStringFromAPI(not)
		if err != nil {
			return nil, err
		}
		notifications[i] = tfNot
	}
	rule["notifications"] = notifications
	rule["disabled"] = r.Disabled
	rule["parameterized_body"] = r.ParameterizedBody
	rule["parameterized_subject"] = r.ParameterizedSubject
	rule["runbook_url"] = r.RunbookUrl
	rule["tip"] = r.Tip
	return rule, nil
}

func detectorUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadDetector(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	payload.Tags = common.Unique(
		pmeta.LoadProviderTags(context.Background(), meta),
		payload.Tags,
	)

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Detector Payload: %s", string(debugOutput))

	det, err := config.Client.UpdateDetector(context.TODO(), d.Id(), payload)
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

	return config.Client.DeleteDetector(context.TODO(), d.Id())
}

func getPerSignalDetectorVizOptions(d *schema.ResourceData) []*detector.PublishLabelOptions {
	viz := d.Get("viz_options").(*schema.Set).List()
	vizList := make([]*detector.PublishLabelOptions, len(viz))
	for i, v := range viz {
		v := v.(map[string]interface{})
		item := &detector.PublishLabelOptions{
			Label: v["label"].(string),
		}
		if val, ok := v["display_name"].(string); ok && val != "" {
			item.DisplayName = val
		}
		if val, ok := v["color"].(string); ok {
			if elem, ok := PaletteColors[val]; ok {
				i := int32(elem)
				item.PaletteIndex = &i
			}
		}
		if val, ok := v["value_unit"].(string); ok && val != "" {
			item.ValueUnit = val
		}
		if val, ok := v["value_suffix"].(string); ok && val != "" {
			item.ValueSuffix = val
		}
		if val, ok := v["value_prefix"].(string); ok && val != "" {
			item.ValuePrefix = val
		}

		vizList[i] = item
	}
	return vizList
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

	serializeReminderToString := func(reminder map[string]interface{}) string {
		var _buf bytes.Buffer

		// Sort keys to ensure consistent hash generation
		keys := make([]string, 0, len(reminder))
		for key := range reminder {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Serialize each key-value pair
		for _, key := range keys {
			value := reminder[key]
			_buf.WriteString(fmt.Sprintf("%s-%v-", key, value))
		}

		return _buf.String()
	}

	// check optional reminder notification
	if reminders, ok := m["reminder_notification"]; ok && reminders != nil {
		for _, reminder := range reminders.([]interface{}) {
			if reminder != nil {
				serializedReminder := serializeReminderToString(reminder.(map[string]interface{}))
				buf.WriteString(fmt.Sprintf("%s", serializedReminder))
			}
		}
	}

	// Sort the notifications so that we generate a consistent hash
	if v, ok := m["notifications"]; ok {
		notifications := v.([]interface{})
		s_notifications := make([]string, len(notifications))
		for i, raw := range notifications {
			if raw != nil {
				s_notifications[i] = raw.(string)
			}
		}
		sort.Strings(s_notifications)

		for _, notification := range s_notifications {
			buf.WriteString(fmt.Sprintf("%s-", notification))
		}
	}

	return HashCodeString(buf.String())
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

func validateReminderType(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	allowedTypes := []string{"TIMEOUT"}
	for _, allowedType := range allowedTypes {
		if value == allowedType {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; must be one of: %s", value, strings.Join(allowedTypes, ", ")))
	return
}

/*
Validates the condition to be fulfilled for checking ProgramText.
*/
func validateProgramTextCondition(ctx context.Context, d *schema.ResourceDiff, meta interface{}) bool {

	if _, ok := d.GetOkExists("program_text"); !ok {
		return false
	}
	if _, ok := d.GetOkExists("rule"); !ok {
		return false
	}

	return true
}

/*
Validates the ProgramText and the list of rules.
*/
func validateProgramText(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {

	tfRules := d.Get("rule").(*schema.Set).List()
	rulesList := make([]*detector.Rule, len(tfRules))
	for i, tfRule := range tfRules {
		tfRule := tfRule.(map[string]interface{})
		rule := &detector.Rule{
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
		rulesList[i] = rule
	}

	config := meta.(*signalfxConfig)

	err := config.Client.ValidateDetector(context.Background(), &detector.ValidateDetectorRequestModel{
		Name:             d.Get("name").(string),
		ProgramText:      d.Get("program_text").(string),
		Rules:            rulesList,
		DetectorOrigin:   d.Get("detector_origin").(string),
		ParentDetectorId: d.Get("parent_detector_id").(string),
	})
	if err != nil {
		return err
	}

	return nil
}

// String hashes a string to a unique hashcode.
//
// crc32 returns a uint32, but for our use we need
// and non negative integer. Here we cast to an integer
// and invert it if the result is negative.
func HashCodeString(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	// v == MinInt
	return 0
}
