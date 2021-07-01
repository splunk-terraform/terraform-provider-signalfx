package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/signalfx/signalfx-go/integration"
)

func integrationAWSResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"integration_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of this integration",
				ForceNew:    true,
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Removed:     "Please specify the name in `signalfx_aws_external_integration` or `signalfx_aws_integration_token`",
				Description: "Name of the integration",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"auth_method": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Removed:     "Use one of `signalfx_aws_external_integration` or `signalfx_aws_token_integration`",
				Description: "The mechanism used to authenticate with AWS.",
			},
			"custom_cloudwatch_namespaces": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:      true,
				Description:   "List of custom AWS CloudWatch namespaces to monitor. Custom namespaces contain custom metrics that you define in AWS; SignalFx imports the metrics so you can monitor them.",
				ConflictsWith: []string{"custom_namespace_sync_rule"},
			},
			"custom_namespace_sync_rule": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"custom_cloudwatch_namespaces"},
				Description:   "Each element controls the data collected by SignalFx for the specified namespace. If you specify this property, SignalFx ignores values in the \"custom_cloudwatch_namespaces\" property.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the SignalFx default behavior for processing data from an AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_source": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Expression that selects the data that SignalFx should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "An AWS custom namespace having custom AWS metrics that you want to sync with SignalFx. See the AWS documentation on publishing metrics for more information.",
						},
					},
				},
			},
			"namespace_sync_rule": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"services"},
				Description:   "Each element in the array is an object that contains an AWS namespace name and a filter that controls the data that SignalFx collects for the namespace. If you specify this property, SignalFx ignores the values in the AWS CloudWatch Integration Model \"services\" property. If you don't specify either property, SignalFx syncs all data in all AWS namespaces.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the SignalFx default behavior for processing data from an AWS namespace. SignalFx ignores this property unless you specify the \"filter\" property in the namespace sync rule. If you do specify a filter, use this property to control how SignalFx treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_source": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Expression that selects the data that SignalFx should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.",
						},
						"namespace": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsService,
							Description:  "An AWS namespace having custom AWS metrics that you want to sync with SignalFx. See the AWS documentation on publishing metrics for more information.",
						},
					},
				},
			},
			"enable_aws_usage": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag that controls how SignalFx imports usage metrics from AWS to use with AWS Cost Optimizer. If `true`, SignalFx imports the metrics.",
			},
			"import_cloud_watch": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag that controls how SignalFx imports Cloud Watch metrics. If true, SignalFx imports Cloud Watch metrics from AWS.",
			},
			"key": {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"role_arn", "external_id"},
				Description:   "Used with `signalfx_aws_token_integration`. Use this property to specify the token.",
			},
			"regions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of AWS regions that SignalFx should monitor.",
			},
			"role_arn": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"token", "key"},
				Description:   "Used with `signalfx_aws_external_integration`. Use this property to specify the AIM role ARN.",
			},
			"services": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateAwsService,
				},
				Description: "List of AWS services that you want SignalFx to monitor. Each element is a string designating an AWS service.",
			},
			"token": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"role_arn", "external_id"},
				Description:   "Used with `signalfx_aws_token_integration`. Use this property to specify the token.",
			},
			"poll_rate": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "AWS poll rate (in seconds). Between `60` and `600`.",
				ValidateFunc: validation.IntBetween(60, 600),
			},
			"external_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"token", "key"},
				Description:   "Used with `signalfx_aws_external_integration`. Use this property to specify the external id.",
			},
			"use_get_metric_data_method": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enables the use of Amazon's GetMetricData API. Defaults to `false`.",
			},
			"named_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A named token to use for ingest",
			},
			"enable_check_large_volume": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Controls how SignalFx checks for large amounts of data for this AWS integration. If true, SignalFx monitors the amount of data coming in from the integration.",
			},
		},

		Create: integrationAWSCreate,
		Read:   integrationAWSRead,
		Update: integrationAWSUpdate,
		Delete: integrationAWSDelete,
		Exists: integrationAWSExists,
	}
}

func integrationAWSExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Get("integration_id").(string))
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func integrationAWSRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Get("integration_id").(string))
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	if int.AuthMethod == integration.EXTERNAL_ID {
		if int.ExternalId != "" {
			if err := d.Set("external_id", int.ExternalId); err != nil {
				return err
			}
		}
	}
	if err := d.Set("name", int.Name); err != nil {
		return err
	}
	if err := d.Set("auth_method", int.AuthMethod); err != nil {
		return err
	}

	return awsIntegrationAPIToTF(d, int)
}

func awsIntegrationAPIToTF(d *schema.ResourceData, aws *integration.AwsCloudWatchIntegration) error {
	debugOutput, _ := json.Marshal(aws)
	log.Printf("[DEBUG] SignalFx: Got AWS Integration to enState: %s", string(debugOutput))

	if err := d.Set("integration_id", aws.Id); err != nil {
		return err
	}
	if err := d.Set("name", aws.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", aws.Enabled); err != nil {
		return err
	}
	if err := d.Set("enable_aws_usage", aws.EnableAwsUsage); err != nil {
		return err
	}
	if err := d.Set("import_cloud_watch", aws.ImportCloudWatch); err != nil {
		return err
	}
	if err := d.Set("poll_rate", aws.PollRate/1000); err != nil {
		return err
	}
	if err := d.Set("use_get_metric_data_method", aws.UseGetMetricDataMethod); err != nil {
		return err
	}
	if err := d.Set("enable_check_large_volume", aws.EnableCheckLargeVolume); err != nil {
		return err
	}
	if err := d.Set("named_token", aws.NamedToken); err != nil {
		return err
	}

	if aws.Token != "" {
		if err := d.Set("token", aws.Token); err != nil {
			return err
		}
	}
	if aws.Key != "" {
		if err := d.Set("key", aws.Key); err != nil {
			return err
		}
	}
	if len(aws.Regions) > 0 {
		if err := d.Set("regions", flattenStringSliceToSet(aws.Regions)); err != nil {
			return err
		}
	}
	if len(aws.CustomNamespaceSyncRules) > 0 {
		var rules []map[string]interface{}
		for _, v := range aws.CustomNamespaceSyncRules {
			if v.Filter != nil {
				rules = append(rules, map[string]interface{}{
					"default_action": string(v.DefaultAction),
					"filter_action":  v.Filter.Action,
					"filter_source":  v.Filter.Source,
					"namespace":      v.Namespace,
				})
			}
		}
		if len(rules) > 0 {
			if err := d.Set("custom_namespace_sync_rule", rules); err != nil {
				return err
			}
		}
	} else {
		// Don't look at this unless they aren't using CustomNamespaceSyncRules
		if aws.CustomCloudWatchNamespaces != "" {
			nesses := strings.Split(aws.CustomCloudWatchNamespaces, ",")
			if err := d.Set("custom_cloudwatch_namespaces", flattenStringSliceToSet(nesses)); err != nil {
				return err
			}
		}
	}
	if _, ok := d.GetOk("services"); ok {
		// The SFx API unhelpfully "upgrades" services entries into
		// `namespace_sync_rule`s with a bunch of null fields. As such we'll ignore
		// `NamespaceSyncRules` if we have services
		if len(aws.Services) > 0 {
			services := make([]interface{}, len(aws.Services))
			for i, v := range aws.Services {
				services[i] = string(v)
			}
			if err := d.Set("services", services); err != nil {
				return err
			}
		}
	} else {
		if len(aws.NamespaceSyncRules) > 0 {
			var rules []map[string]interface{}
			for _, v := range aws.NamespaceSyncRules {
				if v.Filter != nil {
					rules = append(rules, map[string]interface{}{
						"default_action": string(v.DefaultAction),
						"filter_action":  v.Filter.Action,
						"filter_source":  v.Filter.Source,
						"namespace":      string(v.Namespace),
					})
				} else {
					// Sometimes the rules come back with just a namespace and no
					// filters.
					rules = append(rules, map[string]interface{}{
						"default_action": string(v.DefaultAction),
						"namespace":      string(v.Namespace),
					})
				}
			}
			if len(rules) > 0 {
				if err := d.Set("namespace_sync_rule", rules); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func getPayloadAWSIntegration(d *schema.ResourceData) (*integration.AwsCloudWatchIntegration, error) {

	aws := &integration.AwsCloudWatchIntegration{
		Name:                   d.Get("name").(string),
		Type:                   "AWSCloudWatch",
		Enabled:                d.Get("enabled").(bool),
		EnableAwsUsage:         d.Get("enable_aws_usage").(bool),
		ImportCloudWatch:       d.Get("import_cloud_watch").(bool),
		UseGetMetricDataMethod: d.Get("use_get_metric_data_method").(bool),
		EnableCheckLargeVolume: d.Get("enable_check_large_volume").(bool),
	}

	if d.Get("external_id").(string) != "" {
		aws.AuthMethod = integration.EXTERNAL_ID
		aws.ExternalId = d.Get("external_id").(string)
		aws.RoleArn = d.Get("role_arn").(string)
	} else if d.Get("token").(string) != "" {
		aws.AuthMethod = integration.SECURITY_TOKEN
		aws.Token = d.Get("token").(string)
		aws.Key = d.Get("key").(string)
	} else {
		return nil, fmt.Errorf("Please specify one of `external_id` or `token` and `key`")
	}

	if val, ok := d.GetOk("custom_cloudwatch_namespaces"); ok {
		var cwns []string
		for _, ns := range val.(*schema.Set).List() {
			cwns = append(cwns, ns.(string))
		}
		aws.CustomCloudWatchNamespaces = strings.Join(cwns, ",")
	}

	if val, ok := d.GetOk("custom_namespace_sync_rule"); ok {
		val := val.(*schema.Set).List()
		aws.CustomNamespaceSyncRules = getCustomNamespaceRules(val)
	}

	if val, ok := d.GetOk("namespace_sync_rule"); ok {
		val := val.(*schema.Set).List()
		aws.NamespaceSyncRules = getNamespaceRules(val)
	}

	if val, ok := d.GetOk("poll_rate"); ok {
		aws.PollRate = int64(val.(int)) * 1000
	}

	if val, ok := d.GetOk("regions"); ok {
		rs := val.(*schema.Set).List()
		if len(rs) > 0 {
			regions := make([]string, len(rs))
			for i, v := range rs {
				v := v.(string)
				regions[i] = v
			}
			aws.Regions = regions
		}
	}

	if val, ok := d.GetOk("services"); ok {
		esses := val.(*schema.Set).List()
		if len(esses) > 0 {
			services := make([]integration.AwsService, len(esses))
			for i, v := range esses {
				v := integration.AwsService(v.(string))
				services[i] = v
			}
			aws.Services = services
		}
	}

	if val, ok := d.GetOk("named_token"); ok {
		aws.NamedToken = val.(string)
	}

	return aws, nil
}

func getCustomNamespaceRules(tfRules []interface{}) []*integration.AwsCustomNameSpaceSyncRule {
	rules := make([]*integration.AwsCustomNameSpaceSyncRule, len(tfRules))
	for i, r := range tfRules {
		r := r.(map[string]interface{})
		defaultAction := integration.AwsSyncRuleFilterAction("")
		if da, ok := r["default_action"]; ok {
			da := da.(string)
			if da == string(integration.INCLUDE) {
				defaultAction = integration.INCLUDE
			} else if da == string(integration.EXCLUDE) {
				defaultAction = integration.EXCLUDE
			}
		}

		var filter *integration.AwsSyncRuleFilter
		if f, fok := r["filter_action"]; fok && (f.(string) != "") {
			filter = &integration.AwsSyncRuleFilter{
				Action: integration.AwsSyncRuleFilterAction(f.(string)),
				Source: r["filter_source"].(string),
			}
		}

		rules[i] = &integration.AwsCustomNameSpaceSyncRule{
			DefaultAction: defaultAction,
			Filter:        filter,
			Namespace:     r["namespace"].(string),
		}
	}
	return rules
}

func getNamespaceRules(tfRules []interface{}) []*integration.AwsNameSpaceSyncRule {
	rules := make([]*integration.AwsNameSpaceSyncRule, len(tfRules))
	for i, r := range tfRules {
		r := r.(map[string]interface{})
		defaultAction := integration.AwsSyncRuleFilterAction("")
		if da, ok := r["default_action"]; ok {
			da := da.(string)
			if da == string(integration.INCLUDE) {
				defaultAction = integration.INCLUDE
			} else if da == string(integration.EXCLUDE) {
				defaultAction = integration.EXCLUDE
			}
		}

		var filter *integration.AwsSyncRuleFilter
		if f, fok := r["filter_action"]; fok && (f.(string) != "") {
			filter = &integration.AwsSyncRuleFilter{
				Action: integration.AwsSyncRuleFilterAction(f.(string)),
				Source: r["filter_source"].(string),
			}
		}

		rules[i] = &integration.AwsNameSpaceSyncRule{
			DefaultAction: defaultAction,
			Filter:        filter,
			Namespace:     integration.AwsService(r["namespace"].(string)),
		}
	}
	return rules
}

func integrationAWSCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	preInt, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Get("integration_id").(string))
	if err != nil {
		return fmt.Errorf("Error fetching existing integration for integration %s, %s", d.Get("integration_id").(string), err.Error())
	}
	if preInt.AuthMethod == integration.EXTERNAL_ID {
		if err := d.Set("external_id", preInt.ExternalId); err != nil {
			return err
		}
	}
	if err := d.Set("name", preInt.Name); err != nil {
		return err
	}
	payload, err := getPayloadAWSIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create (Update) AWS Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateAWSCloudWatchIntegration(context.TODO(), d.Get("integration_id").(string), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return awsIntegrationAPIToTF(d, int)
}

func integrationAWSUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	payload, err := getPayloadAWSIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update AWS Integration Payload: %s", string(debugOutput))

	int, err := config.Client.UpdateAWSCloudWatchIntegration(context.TODO(), d.Id(), payload)
	if err != nil {
		if strings.Contains(err.Error(), "40") {
			err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
		}
		return err
	}
	d.SetId(int.Id)

	return awsIntegrationAPIToTF(d, int)
}

func integrationAWSDelete(d *schema.ResourceData, meta interface{}) error {
	// Do nothing, let the aws_(external|token)_integration do the deletion
	return nil
}

func validateFilterAction(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	if value != string(integration.EXCLUDE) && value != string(integration.INCLUDE) {
		errors = append(errors, fmt.Errorf("%s not allowed; filter action must be one of %s or %s", value, integration.EXCLUDE, integration.INCLUDE))
	}
	return
}

func validateAwsService(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	for key, _ := range integration.AWSServiceNames {
		if key == value {
			return
		}
	}
	errors = append(errors, fmt.Errorf("%s not allowed; consult the documentation for a list of valid AWS Service names", value))
	return
}
