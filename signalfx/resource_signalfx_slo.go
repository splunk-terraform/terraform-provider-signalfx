// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
    "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/slo"
)

const (
	nameLabel                   = "name"
	descriptionLabel            = "description"
	typeLabel                   = "type"
	inputLabel                  = "input"
	programTextLabel            = "program_text"
	goodEventsLabel             = "good_events_label"
	totalEventsLabel            = "total_events_label"
	targetLabel                 = "target"
	sloLabel                    = "slo"
	compliancePeriodLabel       = "compliance_period"
	cycleTypeLabel              = "cycle_type"
	cycleStartLabel             = "cycle_start"
	alertRuleLabel              = "alert_rule"
	ruleLabel                   = "rule"
	parametersLabel             = "parameters"
	fireLastingLabel            = "fire_lasting"
	percentOfLastingLabel       = "percent_of_lasting"
	percentErrorBudgetLeftLabel = "percent_error_budget_left"
	shortWindow1Label           = "short_window_1"
	longWindow1Label            = "long_window_1"
	shortWindow2Label           = "short_window_2"
	longWindow2Label            = "long_window_2"
	burnRateThreshold1Label     = "burn_rate_threshold_1"
	burnRateThreshold2Label     = "burn_rate_threshold_2"
)

func sloResource() *schema.Resource {

	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			nameLabel: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name of the SLO",
				ValidateFunc: validation.StringLenBetween(0, 256),
			},
			descriptionLabel: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Description of the SLO",
				ValidateFunc: validation.StringLenBetween(0, 1024),
			},
			typeLabel: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Type of the SLO. Currently only RequestBased SLO is supported",
				ValidateFunc: validation.StringInSlice([]string{slo.RequestBased}, false),
			},
			inputLabel: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "SignalFlow program and arguments text strings that define the streams used as successful event count and total event count",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						programTextLabel: {
							Type:     schema.TypeString,
							Required: true,
							Description: "Signalflow program text for the SLO. More info at \"https://dev.splunk.com/observability/docs/signalflow\". " +
								"We require this Signalflow program text to contain at least 2 data blocks - one for the total stream and one for the good stream, whose labels are specified by goodEventsLabel and totalEventsLabel",
							ValidateFunc: validation.StringLenBetween(18, 50000),
						},
						goodEventsLabel: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Label used in `program_text` that refers to the data block which contains the stream of successful events",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						totalEventsLabel: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "Label used in `program_text` that refers to the data block which contains the stream of total events",
							ValidateFunc: validation.StringIsNotEmpty,
						},
					},
				},
			},
			targetLabel: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Define target value of the service level indicator in the appropriate time period.",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						typeLabel: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "SLO target type can be the following type: `RollingWindow`",
							ValidateFunc: validation.StringInSlice([]string{slo.RollingWindowTarget, slo.CalendarWindowTarget}, false),
						},
						sloLabel: {
							Type:         schema.TypeFloat,
							Required:     true,
							Description:  "Target value in the form of a percentage",
							ValidateFunc: validation.FloatBetween(0, 100.0),
						},
						compliancePeriodLabel: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "(Required for `RollingWindow` type) Compliance period of this SLO. This value must be within the range of 1d (1 days) to 30d (30 days), inclusive.",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						cycleTypeLabel: {
							Type:         schema.TypeString,
							Optional:     true,
							Description:  "(Required for `CalendarWindow` type) The cycle type of the calendar window, e.g. week, month.",
							ValidateFunc: validation.StringInSlice([]string{"week", "month"}, false),
						},
						cycleStartLabel: {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							Description:  "(Optional for `CalendarWindow` type)  It can be used to change the cycle start time. For example, you can specify sunday as the start of the week (instead of the default monday)",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						alertRuleLabel: {
							Type:        schema.TypeList,
							Required:    true,
							Description: "SLO alert rules",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									typeLabel: {
										Type:         schema.TypeString,
										Required:     true,
										Description:  "SLO alert rule type",
										ValidateFunc: validation.StringInSlice([]string{slo.BreachRule, slo.ErrorBudgetLeftRule, slo.BurnRateRule}, false),
									},
									ruleLabel: {
										Type:        schema.TypeList,
										Required:    true,
										Description: "Set of rules used for alerting",
										Elem: &schema.Resource{
											SchemaFunc: func() map[string]*schema.Schema {
												// The alert rules for SLO are very similar to those in Detector with 2 exceptions:
												// 1. We don't expect detect_label. The user can send it - but we will ignore it - so we remove it from the TF schema here
												// 2. There is an additional field called parameters, which the user can use to parameterize the program text of the SLO
												ruleSchema := make(map[string]*schema.Schema)

												for k, v := range detectorRuleSchema {
													if k != "detect_label" {
														ruleSchema[k] = v
													}
												}

												ruleSchema[parametersLabel] = &schema.Schema{
													Type:        schema.TypeList,
													Optional:    true,
													Description: "Parameters for the SLO alert rule. Each SLO alert rule type accepts different parameters. If not specified, default parameters are used.",
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															fireLastingLabel: {
																Type:        schema.TypeString,
																Optional:    true,
																Computed:    true,
																Description: "Duration that indicates how long the alert condition is met before the alert is triggered. The value must be positive and smaller than the compliance period of the SLO target. Note: BREACH and ERROR_BUDGET_LEFT alert rules use the fire_lasting parameter",
															},
															percentOfLastingLabel: {
																Type:        schema.TypeFloat,
																Optional:    true,
																Computed:    true,
																Description: "Percentage of the fire_lasting duration that the alert condition is met before the alert is triggered. Note: BREACH and ERROR_BUDGET_LEFT alert rules use the percent_of_lasting parameter",
															},
															percentErrorBudgetLeftLabel: {
																Type:        schema.TypeFloat,
																Optional:    true,
																Computed:    true,
																Description: "Error budget must be equal to or smaller than this percentage for the alert to be triggered. Note: ERROR_BUDGET_LEFT alert rules use the percent_error_budget_left parameter.",
															},
															shortWindow1Label: {
																Type:        schema.TypeString,
																Optional:    true,
																Computed:    true,
																Description: "Short window 1 used in burn rate alert calculation. This value must be longer than 1/30 of long_window_1. Note: BURN_RATE alert rules use the short_window_1 parameter.",
															},
															shortWindow2Label: {
																Type:        schema.TypeString,
																Optional:    true,
																Computed:    true,
																Description: "Short window 2 used in burn rate alert calculation. This value must be longer than 1/30 of long_window_2. Note: BURN_RATE alert rules use the short_window_2 parameter.",
															},
															longWindow1Label: {
																Type:        schema.TypeString,
																Optional:    true,
																Computed:    true,
																Description: "Long window 1 used in burn rate alert calculation. This value must be longer than short_window_1` and shorter than 90 days. Note: BURN_RATE alert rules use the long_window_1 parameter. ",
															},
															longWindow2Label: {
																Type:        schema.TypeString,
																Optional:    true,
																Computed:    true,
																Description: "Long window 2 used in burn rate alert calculation. This value must be longer than short_window_2` and shorter than 90 days. Note: BURN_RATE alert rules use the long_window_2 parameter. ",
															},
															burnRateThreshold1Label: {
																Type:        schema.TypeFloat,
																Optional:    true,
																Computed:    true,
																Description: "Burn rate threshold 1 used in burn rate alert calculation. This value must be between 0 and 100/(100-SLO target). Note: BURN_RATE alert rules use the burn_rate_threshold_1 parameter.",
															},
															burnRateThreshold2Label: {
																Type:        schema.TypeFloat,
																Optional:    true,
																Computed:    true,
																Description: "Burn rate threshold 2 used in burn rate alert calculation. This value must be between 0 and 100/(100-SLO target). Note: BURN_RATE alert rules use the burn_rate_threshold_2 parameter.",
															},
														},
													},
												}

												return ruleSchema
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},

		SchemaVersion: 1,

		CustomizeDiff: sloValidate,
		CreateContext: sloCreate,
		ReadContext:   sloRead,
		UpdateContext: sloUpdate,
		DeleteContext: sloDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func sloValidate(ctx context.Context, sloObject *schema.ResourceDiff, config interface{}) error {
	payloadSlo, err := getPayloadSlo(sloObject)

	// we are checking the uniqueness of the slo name, so we need some randomness here if it's an SLO update(resource ID exists)
	if sloObject.Id() != "" {
		payloadSlo.Name = payloadSlo.Name + time.Now().String()
	}

	if err != nil {
		return err
	}

	err = config.(*signalfxConfig).Client.ValidateSlo(ctx, payloadSlo)
	if err != nil {
		return err
	}

	return nil
}

func sloCreate(ctx context.Context, sloResource *schema.ResourceData, config interface{}) diag.Diagnostics {
	client := config.(*signalfxConfig).Client
	payload, err := getPayloadSlo(sloResource)
	if err != nil {
		return diag.Errorf("Failed creating SLO json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] Create SLO Payload: %s", string(debugOutput))

	createdSlo, err := client.CreateSlo(ctx, payload)
    if err != nil {
        return tfext.AsErrorDiagnostics(err)
    }

	id := createdSlo.Id
	sloResource.SetId(id)

    err = sloAPIToTF(sloResource, createdSlo)
    return tfext.AsErrorDiagnostics(err)
}

func sloUpdate(ctx context.Context, sloResource *schema.ResourceData, config interface{}) diag.Diagnostics {
	client := config.(*signalfxConfig).Client
	payload, err := getPayloadSlo(sloResource)
	if err != nil {
		return diag.Errorf("Failed creating SLO json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] Update SLO Payload: %s", string(debugOutput))

	updatedSlo, err := client.UpdateSlo(ctx, sloResource.Id(), payload)
    if err != nil {
        return tfext.AsErrorDiagnostics(err)
    }

    err = sloAPIToTF(sloResource, updatedSlo)
    return tfext.AsErrorDiagnostics(err)
}

func sloRead(ctx context.Context, d *schema.ResourceData, config interface{}) diag.Diagnostics {
	client := config.(*signalfxConfig).Client

	returnedSlo, err := client.GetSlo(ctx, d.Id())
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
		}
        return tfext.AsErrorDiagnostics(err)
	}

    err = sloAPIToTF(d, returnedSlo)
    return tfext.AsErrorDiagnostics(err)
}

func sloDelete(ctx context.Context, d *schema.ResourceData, config interface{}) diag.Diagnostics {
	client := config.(*signalfxConfig).Client

	err := client.DeleteSlo(ctx, d.Id())

    if err != nil {
        return tfext.AsErrorDiagnostics(err)
    }

	return nil
}

// getPayloadSlo is called with objects: *schema.ResourceDiff or *schema.ResourceData - that's why we need this interface to have only one method
type Resource interface {
	Get(key string) interface{}
}

func getPayloadSlo(sloTfResource Resource) (*slo.SloObject, error) {
	sloApiObject := &slo.SloObject{
		BaseSlo: slo.BaseSlo{
			Name:        sloTfResource.Get(nameLabel).(string),
			Description: sloTfResource.Get(descriptionLabel).(string),
			Type:        sloTfResource.Get(typeLabel).(string),
		},
	}

	targets, err := getApiTargets(sloTfResource.Get(targetLabel).([]interface{}))
	if err != nil {
		return nil, err
	}
	sloApiObject.Targets = targets

	return setSloInput(sloTfResource, sloApiObject)
}

func getApiTargets(tfTargets []interface{}) ([]slo.SloTarget, error) {
	apiTargets := make([]slo.SloTarget, len(tfTargets))

	for ind, rawTfTarget := range tfTargets {
		tfTarget := rawTfTarget.(map[string]interface{})

		apiTarget := slo.SloTarget{
			BaseSloTarget: slo.BaseSloTarget{
				Slo:  tfTarget[sloLabel].(float64),
				Type: tfTarget[typeLabel].(string),
			},
		}

		switch apiTarget.Type {
		case slo.RollingWindowTarget:
			apiTarget.RollingWindowSloTarget = &slo.RollingWindowSloTarget{
				CompliancePeriod: tfTarget[compliancePeriodLabel].(string),
			}
		case slo.CalendarWindowTarget:
			apiTarget.CalendarWindowSloTarget = &slo.CalendarWindowSloTarget{
				CycleStart: tfTarget[cycleStartLabel].(string),
				CycleType:  tfTarget[cycleTypeLabel].(string),
			}
		default:
			return nil, fmt.Errorf("unsupported SLO target type: %s", apiTarget.Type)
		}

		alertRules, err := getApiAlertRules(tfTarget[alertRuleLabel].([]interface{}))
		if err != nil {
			return nil, err
		}

		apiTarget.SloAlertRules = alertRules
		apiTargets[ind] = apiTarget
	}

	return apiTargets, nil

}

func getApiAlertRules(tfAlertRules []interface{}) ([]slo.SloAlertRule, error) {
	apiAlertRules := make([]slo.SloAlertRule, len(tfAlertRules))

	for ind, rawTfAlertRule := range tfAlertRules {
		tfAlertRule := rawTfAlertRule.(map[string]interface{})

		alertType := tfAlertRule[typeLabel].(string)
		switch alertType {
		case slo.BreachRule:
			detectorRules, err := getApiDetectorRules[slo.BreachDetectorRule](
				tfAlertRule[ruleLabel].([]interface{}),
				func(rule *detector.Rule) *slo.BreachDetectorRule {
					return &slo.BreachDetectorRule{
						Rule: *rule,
					}
				},
				func(rule *slo.BreachDetectorRule, parameters map[string]interface{}) {
					rule.Parameters = &slo.BreachDetectorParameters{
						FireLasting:      parameters[fireLastingLabel].(string),
						PercentOfLasting: parameters[percentOfLastingLabel].(float64),
					}
				},
			)
			if err != nil {
				return nil, err
			}

			apiAlertRules[ind].BreachSloAlertRule = &slo.BreachSloAlertRule{
				Rules: detectorRules,
			}
			apiAlertRules[ind].Type = alertType

		case slo.ErrorBudgetLeftRule:
			detectorRules, err := getApiDetectorRules[slo.ErrorBudgetLeftDetectorRule](
				tfAlertRule[ruleLabel].([]interface{}),
				func(rule *detector.Rule) *slo.ErrorBudgetLeftDetectorRule {
					return &slo.ErrorBudgetLeftDetectorRule{
						Rule: *rule,
					}
				},
				func(rule *slo.ErrorBudgetLeftDetectorRule, parameters map[string]interface{}) {
					rule.Parameters = &slo.ErrorBudgetLeftDetectorParameters{
						FireLasting:            parameters[fireLastingLabel].(string),
						PercentOfLasting:       parameters[percentOfLastingLabel].(float64),
						PercentErrorBudgetLeft: parameters[percentErrorBudgetLeftLabel].(float64),
					}
				},
			)
			if err != nil {
				return nil, err
			}

			apiAlertRules[ind].ErrorBudgetLeftSloAlertRule = &slo.ErrorBudgetLeftSloAlertRule{
				Rules: detectorRules,
			}
			apiAlertRules[ind].Type = alertType
		case slo.BurnRateRule:
			detectorRules, err := getApiDetectorRules[slo.BurnRateDetectorRule](
				tfAlertRule[ruleLabel].([]interface{}),
				func(rule *detector.Rule) *slo.BurnRateDetectorRule {
					return &slo.BurnRateDetectorRule{
						Rule: *rule,
					}
				},
				func(rule *slo.BurnRateDetectorRule, parameters map[string]interface{}) {
					rule.Parameters = &slo.BurnRateDetectorParameters{
						ShortWindow1:       parameters[shortWindow1Label].(string),
						LongWindow1:        parameters[longWindow1Label].(string),
						ShortWindow2:       parameters[shortWindow2Label].(string),
						LongWindow2:        parameters[longWindow2Label].(string),
						BurnRateThreshold1: parameters[burnRateThreshold1Label].(float64),
						BurnRateThreshold2: parameters[burnRateThreshold2Label].(float64),
					}
				},
			)
			if err != nil {
				return nil, err
			}

			apiAlertRules[ind].BurnRateSloAlertRule = &slo.BurnRateSloAlertRule{
				Rules: detectorRules,
			}
			apiAlertRules[ind].Type = alertType
		default:
			return nil, fmt.Errorf("unsupported SLO alert rule type: %s", alertType)
		}
	}

	return apiAlertRules, nil
}

type DetectorRuleType interface {
	slo.BreachDetectorRule | slo.ErrorBudgetLeftDetectorRule | slo.BurnRateDetectorRule
}

func getApiDetectorRules[DetectorRule DetectorRuleType](
	tfRules []interface{},
	newSloDetectorRule func(*detector.Rule) *DetectorRule,
	setSloDetectorParameters func(rule *DetectorRule, source map[string]interface{})) ([]*DetectorRule, error) {

	apiDetectorRules := make([]*DetectorRule, len(tfRules))

	for ind, tfRule := range tfRules {
		detectorRule, err := getDetectorRule(tfRule.(map[string]interface{}))
		if err != nil {
			return nil, err
		}

		apiDetectorRules[ind] = newSloDetectorRule(detectorRule)

		parameters, err := getParametersFromRule(tfRule.(map[string]interface{}))
		if err != nil {
			return nil, err
		}

		if parameters != nil {
			setSloDetectorParameters(apiDetectorRules[ind], parameters)
		}
	}
	return apiDetectorRules, nil
}

func getParametersFromRule(tfRule map[string]interface{}) (map[string]interface{}, error) {
	parameters := tfRule[parametersLabel].([]interface{})

	switch len(parameters) {
	case 0:
		return nil, nil
	case 1:
		if parameters[0] != nil {
			return parameters[0].(map[string]interface{}), nil
		} else {
			return nil, nil
		}
	default:
		return nil, fmt.Errorf("expecting at most one parameter to be present")
	}
}

func setSloInput(sloTfResource Resource, sloApiObject *slo.SloObject) (*slo.SloObject, error) {
	switch sloApiObject.Type {
	case slo.RequestBased:
		requestBasedInput, err := getRequestBasedApiInput(sloTfResource)
		if err != nil {
			return nil, err
		}
		sloApiObject.RequestBasedSlo = &slo.RequestBasedSlo{requestBasedInput}
	default:
		return nil, fmt.Errorf("unsupported SLO type: %s", sloApiObject.Type)
	}
	return sloApiObject, nil
}

func getRequestBasedApiInput(sloTfResource Resource) (*slo.RequestBasedSloInput, error) {
	inputs := sloTfResource.Get(inputLabel).([]interface{})

	if len(inputs) != 1 {
		return nil, fmt.Errorf("expecting exactly one input to be present")
	}

	tfInput := inputs[0].(map[string]interface{})
	requestBasedInput := &slo.RequestBasedSloInput{
		ProgramText:      tfInput[programTextLabel].(string),
		GoodEventsLabel:  tfInput[goodEventsLabel].(string),
		TotalEventsLabel: tfInput[totalEventsLabel].(string),
	}
	return requestBasedInput, nil
}

func sloAPIToTF(sloTfResource *schema.ResourceData, sloApiObject *slo.SloObject) error {
	debugOutput, _ := json.Marshal(sloApiObject)
	log.Printf("[DEBUG] Convert SLO to TF State: %s", string(debugOutput))

	if errSet := sloTfResource.Set(nameLabel, sloApiObject.Name); errSet != nil {
		return errSet
	}
	if errSet := sloTfResource.Set(descriptionLabel, sloApiObject.Description); errSet != nil {
		return errSet
	}
	if errSet := sloTfResource.Set(typeLabel, sloApiObject.Type); errSet != nil {
		return errSet
	}

	tfSloInput, err := getTfSloInput(sloApiObject)
	if err != nil {
		return err
	}

	if errSet := sloTfResource.Set(inputLabel, []map[string]interface{}{tfSloInput}); errSet != nil {
		return errSet
	}

	tfTargets, err := getTfTargets(sloApiObject)
	if err != nil {
		return err
	}

	if errSet := sloTfResource.Set(targetLabel, tfTargets); errSet != nil {
		return errSet
	}

	return nil
}

func getTfSloInput(sloApiObject *slo.SloObject) (map[string]interface{}, error) {
	switch sloApiObject.Type {
	case slo.RequestBased:
		tfInput := getRequestBasedTerraformInput(sloApiObject.RequestBasedSlo.Inputs)
		return tfInput, nil
	default:
		return nil, fmt.Errorf("Unsupported SLO type: %q", sloApiObject.Type)
	}
}

func getRequestBasedTerraformInput(sloInput *slo.RequestBasedSloInput) map[string]interface{} {
	tfInput := make(map[string]interface{})
	tfInput[programTextLabel] = sloInput.ProgramText
	tfInput[goodEventsLabel] = sloInput.GoodEventsLabel
	tfInput[totalEventsLabel] = sloInput.TotalEventsLabel
	return tfInput
}

func getTfTargets(sloApiObject *slo.SloObject) ([]map[string]interface{}, error) {
	tfTargets := make([]map[string]interface{}, len(sloApiObject.Targets))
	for ind, apiTarget := range sloApiObject.Targets {
		tfTarget := make(map[string]interface{})
		tfTarget[sloLabel] = apiTarget.Slo
		tfTarget[typeLabel] = apiTarget.Type

		switch apiTarget.Type {
		case slo.RollingWindowTarget:
			tfTarget[compliancePeriodLabel] = apiTarget.RollingWindowSloTarget.CompliancePeriod
		case slo.CalendarWindowTarget:
			tfTarget[cycleTypeLabel] = apiTarget.CalendarWindowSloTarget.CycleType
			tfTarget[cycleStartLabel] = apiTarget.CalendarWindowSloTarget.CycleStart
		default:
			return nil, fmt.Errorf("unsupported SLO target type: %s", apiTarget.Type)
		}

		tfAlertRules, err := getTfAlertRules(apiTarget.SloAlertRules)
		if err != nil {
			return nil, err
		}
		tfTarget[alertRuleLabel] = tfAlertRules
		tfTargets[ind] = tfTarget
	}
	return tfTargets, nil
}

func getTfAlertRules(apiAlertRules []slo.SloAlertRule) (interface{}, error) {
	tfAlertRules := make([]map[string]interface{}, len(apiAlertRules))

	// Since the API can return alert rules in any order, we need to sort here to avoid TF wanting to update a resource because the order has changed.
	sort.SliceStable(apiAlertRules, func(i, j int) bool {
		return apiAlertRules[i].Type < apiAlertRules[j].Type
	})

	for ind, apiAlertRule := range apiAlertRules {
		tfAlertRule := make(map[string]interface{})
		tfAlertRule[typeLabel] = apiAlertRule.Type

		switch apiAlertRule.Type {
		case slo.BreachRule:

			tfDetectorRules, err := getTfDetectorRules[slo.BreachDetectorRule](
				apiAlertRule.BreachSloAlertRule.Rules,
				func(apiRule slo.BreachDetectorRule) *detector.Rule {
					return &apiRule.Rule
				},
				func(apiRule slo.BreachDetectorRule) []map[string]interface{} {
					if parameters := apiRule.Parameters; parameters != nil {
						return []map[string]interface{}{{
							percentOfLastingLabel: parameters.PercentOfLasting,
							fireLastingLabel:      parameters.FireLasting,
						},
						}
					}

					return nil
				},
			)

			if err != nil {
				return nil, err
			}
			tfAlertRule[ruleLabel] = tfDetectorRules

		case slo.ErrorBudgetLeftRule:
			tfDetectorRules, err := getTfDetectorRules[slo.ErrorBudgetLeftDetectorRule](
				apiAlertRule.ErrorBudgetLeftSloAlertRule.Rules,
				func(apiRule slo.ErrorBudgetLeftDetectorRule) *detector.Rule {
					return &apiRule.Rule
				},
				func(apiRule slo.ErrorBudgetLeftDetectorRule) []map[string]interface{} {
					if parameters := apiRule.Parameters; parameters != nil {
						return []map[string]interface{}{{
							percentOfLastingLabel:       parameters.PercentOfLasting,
							fireLastingLabel:            parameters.FireLasting,
							percentErrorBudgetLeftLabel: parameters.PercentErrorBudgetLeft,
						},
						}
					}

					return nil
				},
			)

			if err != nil {
				return nil, err
			}
			tfAlertRule[ruleLabel] = tfDetectorRules

		case slo.BurnRateRule:
			tfDetectorRules, err := getTfDetectorRules[slo.BurnRateDetectorRule](
				apiAlertRule.BurnRateSloAlertRule.Rules,
				func(apiRule slo.BurnRateDetectorRule) *detector.Rule {
					return &apiRule.Rule
				},
				func(apiRule slo.BurnRateDetectorRule) []map[string]interface{} {
					if parameters := apiRule.Parameters; parameters != nil {
						return []map[string]interface{}{{
							shortWindow1Label:       parameters.ShortWindow1,
							longWindow1Label:        parameters.LongWindow1,
							shortWindow2Label:       parameters.ShortWindow2,
							longWindow2Label:        parameters.LongWindow2,
							burnRateThreshold1Label: parameters.BurnRateThreshold1,
							burnRateThreshold2Label: parameters.BurnRateThreshold2,
						},
						}
					}

					return nil
				},
			)

			if err != nil {
				return nil, err
			}
			tfAlertRule[ruleLabel] = tfDetectorRules
		default:
			return nil, fmt.Errorf("unsupported SLO alert rule type: %s", apiAlertRule.Type)

		}

		tfAlertRules[ind] = tfAlertRule
	}

	return tfAlertRules, nil
}

type DetectorRuleProvider[Rule DetectorRuleType] func(rule Rule) (detectorRule *detector.Rule)

type RuleParametersProvider[Rule DetectorRuleType] func(rule Rule) []map[string]interface{}

func getTfDetectorRules[Rule DetectorRuleType](alertRules []*Rule,
	detectorRuleProvider DetectorRuleProvider[Rule],
	ruleParametersProvider RuleParametersProvider[Rule]) ([]map[string]interface{}, error) {

	tfDetectorRules := make([]map[string]interface{}, len(alertRules))

	for ind, apiRule := range alertRules {
		tfDetectorRule, err := getTfDetectorRule(detectorRuleProvider(*apiRule))
		delete(tfDetectorRule, "detect_label") // We don't expect detect_label. The user can send it - but we will ignore it - so we remove it from the TF schema here

		if err != nil {
			return nil, err
		}

		tfDetectorRule[parametersLabel] = ruleParametersProvider(*apiRule)
		tfDetectorRules[ind] = tfDetectorRule

	}
	return tfDetectorRules, nil
}
