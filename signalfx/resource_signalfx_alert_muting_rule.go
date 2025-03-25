// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/alertmuting"
)

const alertMutingDetectorIdProperty = "sf_detectorId"

func alertMutingRuleResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "description of the rule",
			},
			"detectors": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "detectors to which this muting rule applies",
			},
			"filter": {
				Type:         schema.TypeSet,
				Optional:     true,
				AtLeastOneOf: []string{"detectors"},
				Description:  "list of alert muting filters for this rule",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"property": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringNotInSlice([]string{alertMutingDetectorIdProperty}, false),
							Description:  "the property to filter by",
						},
						"property_value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "the value of the property to filter by",
						},
						"negated": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "(false by default) whether this filter should be a \"not\" filter",
						},
					},
				},
			},
			"recurrence": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"unit": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"d", "w"}, false),
							Description:  "unit of the period. Can be days (d) or weeks (w)",
						},
						"value": {
							Type:         schema.TypeInt,
							Required:     true,
							Description:  "amount of time, expressed as an integer applicable to the unit",
							ValidateFunc: validation.IntAtLeast(1),
						},
					},
				},
			},
			"start_time": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "starting time of an alert muting rule as a Unix timestamp, in seconds",
				ForceNew:    true,
			},
			"stop_time": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "stop time of an alert muting rule as a Unix timestamp, in seconds",
			},
			// Because the API returns a different start time from that
			// defined in the config file, we need another place to store
			// that. Note that we won't be doing seconds conversion on
			// this field since it is *not* user-facing.
			"effective_start_time": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
		Create: alertMutingRuleCreate,
		Read:   alertMutingRuleRead,
		Update: alertMutingRuleUpdate,
		Delete: alertMutingRuleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func getPayloadAlertMutingRule(d *schema.ResourceData) (*alertmuting.CreateUpdateAlertMutingRuleRequest, error) {
	var filterList []*alertmuting.AlertMutingRuleFilter

	if filters, ok := d.GetOk("filter"); ok {
		tfFilters := filters.(*schema.Set).List()
		for _, tfFilter := range tfFilters {
			tfFilter := tfFilter.(map[string]interface{})
			filter := &alertmuting.AlertMutingRuleFilter{
				Property:      tfFilter["property"].(string),
				PropertyValue: alertmuting.StringOrArray{Values: []string{tfFilter["property_value"].(string)}},
				NOT:           tfFilter["negated"].(bool),
			}
			filterList = append(filterList, filter)
		}
	}

	// Detectors is a convenience property that allows
	// the user a way to specific the detectors to which
	// this rule will apply without having to know the details
	// of how that happens.
	if val, ok := d.GetOk("detectors"); ok {
		for _, d := range val.([]interface{}) {
			filterList = append(filterList, &alertmuting.AlertMutingRuleFilter{
				Property:      alertMutingDetectorIdProperty,
				PropertyValue: alertmuting.StringOrArray{Values: []string{d.(string)}},
				NOT:           false,
			})
		}
	}

	cuamrr := &alertmuting.CreateUpdateAlertMutingRuleRequest{
		Description: d.Get("description").(string),
		Filters:     filterList,
		StartTime:   int64(d.Get("start_time").(int) * 1000),
		StopTime:    int64(d.Get("stop_time").(int) * 1000),
	}

	if recurrence, ok := d.GetOk("recurrence"); ok {
		tfRecurrences := recurrence.(*schema.Set).List()
		if len(tfRecurrences) > 0 {
			recurrence := tfRecurrences[0].(map[string]interface{})
			cuamrr.Recurrence = &alertmuting.AlertMutingRuleRecurrence{
				Unit:  recurrence["unit"].(string),
				Value: int32(recurrence["value"].(int)),
			}
		}
	}

	return cuamrr, nil
}

func alertMutingRuleCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadAlertMutingRule(d)
	if err != nil {
		return fmt.Errorf("failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Alert Muting Rule Payload: %s", string(debugOutput))

	amr, err := config.Client.CreateAlertMutingRule(context.TODO(), payload)
	if err != nil {
		return err
	}
	d.SetId(amr.Id)

	return alertMutingRuleAPIToTF(d, amr)
}

func alertMutingRuleRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	amr, err := config.Client.GetAlertMutingRule(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	return alertMutingRuleAPIToTF(d, amr)
}

func alertMutingRuleAPIToTF(d *schema.ResourceData, amr *alertmuting.AlertMutingRule) error {
	debugOutput, _ := json.Marshal(amr)
	log.Printf("[DEBUG] SignalFx: Got Alert Muting Rule to enState: %s", string(debugOutput))

	if err := d.Set("description", amr.Description); err != nil {
		return err
	}

	if amr.Filters != nil && len(amr.Filters) > 0 {
		var filters []map[string]interface{}
		var detectors []string
		for _, f := range amr.Filters {

			val := ""
			if len(f.PropertyValue.Values) == 1 {
				val = f.PropertyValue.Values[0]
			} else if len(f.PropertyValue.Values) > 1 {
				return errors.New("terraform provider does not support arrays in alert muting rule filter values")
			}

			switch f.Property {
			// The API does not differentiate, but we do to make things
			// easier for the user, so separate detectors out into their
			// own property.
			case alertMutingDetectorIdProperty:
				detectors = append(detectors, val)
			default:
				filters = append(filters, map[string]interface{}{
					"property":       f.Property,
					"property_value": val,
					"negated":        f.NOT,
				})
			}
		}
		if filters != nil {
			if err := d.Set("filter", filters); err != nil {
				return err
			}
		}
		if detectors != nil {
			if err := d.Set("detectors", detectors); err != nil {
				return err
			}
		}
		// The API changes `startTime` to be >= the current
		// timestamp at the time of the API call. This means
		// it will pretty much never agree with what the user specified.
		// To accommodate this we will store the "effective" start time
		// as a computed attribute, then…
		if err := d.Set("effective_start_time", amr.StartTime); err != nil {
			return err
		}
		// We will ignore the start time because it doesn't matter.
		// See above.
		if err := d.Set("stop_time", amr.StopTime/1000); err != nil {
			return err
		}
	}

	if amr.Recurrence != nil {
		d.Set("recurrence", []interface{}{
			map[string]interface{}{
				"unit":  amr.Recurrence.Unit,
				"value": amr.Recurrence.Value,
			},
		})
	} else {
		d.Set("recurrence", nil)
	}

	return nil
}

func alertMutingRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadAlertMutingRule(d)
	if err != nil {
		return fmt.Errorf("failed creating json payload: %s", err.Error())
	}

	// If we have an effective start time…
	if val, ok := d.GetOk("effective_start_time"); ok {
		est := val.(int)
		st := d.Get("start_time").(int)
		// and if the start time is in the past…
		if int64(st) <= time.Now().Unix() {
			// then replace the start time with the effective start
			// time. This papers over the fact that the API basically
			// ignores our start times unless they are in the future.
			payload.StartTime = int64(est)
			log.Printf("[DEBUG] SignalFx: Replaced start time with effective time")
		} else {
			log.Printf("[DEBUG] SignalFx: Using specified start time")
			payload.StartTime = int64(d.Get("start_time").(int)) * 1000
		}
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Alert Muting Rule Payload: %s", string(debugOutput))

	det, err := config.Client.UpdateAlertMutingRule(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Alert Muting Rule Response: %v", det)

	d.SetId(det.Id)
	return alertMutingRuleAPIToTF(d, det)
}

func alertMutingRuleDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	err := config.Client.DeleteAlertMutingRule(context.TODO(), d.Id())
	// Silently ignore muting in the past for there is nothing the client could do with them and attempt to destroy
	// results in invalid terraform state.
	// 400 : Cannot delete alert muting in the past
	if err != nil && strings.Contains(err.Error(), "400") {
		log.Print("[DEBUG] SignalFx: Ignoring Delete Alert Muting Rule error 400 for alert muting in the past")
		return nil
	}
	return err
}
