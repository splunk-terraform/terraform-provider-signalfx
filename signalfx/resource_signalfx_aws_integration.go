package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

func integrationAWSResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the integration",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"auth_method": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The mechanism used to authenticate with AWS. The allowed values are \"ExternalID\" or \"SecurityToken\"",
				ValidateFunc: validateAuthMethod,
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
							Required:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the SignalFx default behavior for processing data from an AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_action": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_source": {
							Type:        schema.TypeString,
							Required:    true,
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
							Required:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the SignalFx default behavior for processing data from an AWS namespace. SignalFx ignores this property unless you specify the \"filter\" property in the namespace sync rule. If you do specify a filter, use this property to control how SignalFx treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_action": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_source": {
							Type:        schema.TypeString,
							Required:    true,
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
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the key.",
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
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Role ARN that you add to an existing AWS integration object",
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
				Type:        schema.TypeString,
				Optional:    true,
				Description: "If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the token.",
			},
			"poll_rate": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "AWS poll rate (in seconds). One of `60` or `300`.",
				ValidateFunc: validateAwsPollRate,
			},
			"external_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "If you specify `authMethod = \"ExternalId\"` in your request to create an AWS integration object, the response object contains a value for `externalId`. Use this value and the ARN value you get from AWS to update the integration object. SignalFx can then connect to AWS using the integration object.",
			},
		},

		Create: integrationAWSCreate,
		Read:   integrationAWSRead,
		Update: integrationAWSUpdate,
		Delete: integrationAWSDelete,
		Exists: integrationAWSExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func integrationAWSExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetAWSCloudWatchIntegration(d.Id())
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
	int, err := config.Client.GetAWSCloudWatchIntegration(d.Id())
	if err != nil {
		if strings.HasPrefix(err.Error(), "404") {
			d.SetId("")
		}
		return err
	}

	if err := d.Set("external_id", int.ExternalId); err != nil {
		return err
	}

	return awsIntegrationAPIToTF(d, int)
}

func awsIntegrationAPIToTF(d *schema.ResourceData, aws *integration.AwsCloudWatchIntegration) error {
	debugOutput, _ := json.Marshal(aws)
	log.Printf("[DEBUG] SignalFx: Got AWS Integration to enState: %s", string(debugOutput))

	if err := d.Set("name", aws.Name); err != nil {
		return err
	}
	if err := d.Set("enabled", aws.Enabled); err != nil {
		return err
	}
	if err := d.Set("auth_method", aws.AuthMethod); err != nil {
		return err
	}
	if err := d.Set("enable_aws_usage", aws.EnableAwsUsage); err != nil {
		return err
	}
	if err := d.Set("import_cloud_watch", aws.ImportCloudWatch); err != nil {
		return err
	}
	if err := d.Set("poll_rate", *aws.PollRate/1000); err != nil {
		return err
	}
	if err := d.Set("token", aws.Token); err != nil {
		return err
	}
	if err := d.Set("key", aws.Key); err != nil {
		return err
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
	if len(aws.NamespaceSyncRules) > 0 {
		var rules []map[string]interface{}
		for _, v := range aws.NamespaceSyncRules {
			// Sometimes the rules come back with just a namespace and no
			// filters. If that's the case we'll ignore it and leverage
			// that it also comes in the `services` field.
			if v.Filter != nil {
				rules = append(rules, map[string]interface{}{
					"default_action": string(v.DefaultAction),
					"filter_action":  v.Filter.Action,
					"filter_source":  v.Filter.Source,
					"namespace":      string(v.Namespace),
				})
			}
		}
		if len(rules) > 0 {
			if err := d.Set("namespace_sync_rule", rules); err != nil {
				return err
			}
		}
	} else {
		// Only look for services if we don't have NamespaceSyncRules
		if len(aws.Services) > 0 {
			services := make([]interface{}, len(aws.Services))
			for i, v := range aws.Services {
				services[i] = string(v)
			}
			if err := d.Set("services", services); err != nil {
				return err
			}
		}
	}

	return nil
}

func getPayloadAWSIntegration(d *schema.ResourceData) (*integration.AwsCloudWatchIntegration, error) {

	aws := &integration.AwsCloudWatchIntegration{
		Name:             d.Get("name").(string),
		Type:             "AWSCloudWatch",
		Enabled:          d.Get("enabled").(bool),
		EnableAwsUsage:   d.Get("enable_aws_usage").(bool),
		ImportCloudWatch: d.Get("import_cloud_watch").(bool),
		Key:              d.Get("key").(string),
		RoleArn:          d.Get("role_arn").(string),
		Token:            d.Get("token").(string),
		ExternalId:       d.Get("external_id").(string),
	}

	if val, ok := d.GetOk("auth_method"); ok {
		authMethod := integration.EXTERNAL_ID
		if val == string(integration.SECURITY_TOKEN) {
			authMethod = integration.SECURITY_TOKEN
		}
		aws.AuthMethod = authMethod
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
		val := val.(int)
		if val != 0 {
			pollRate := integration.OneMinutely
			if val == 300 {
				pollRate = integration.FiveMinutely
			}
			aws.PollRate = &pollRate
		}
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
			} else {
				defaultAction = integration.EXCLUDE
			}
		}

		var filter *integration.AwsSyncRuleFilter
		if f, fok := r["filter_action"]; fok {
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
			} else {
				defaultAction = integration.EXCLUDE
			}
		}

		var filter *integration.AwsSyncRuleFilter
		if f, fok := r["filter_action"]; fok {
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
	payload, err := getPayloadAWSIntegration(d)
	if err != nil {
		return fmt.Errorf("Failed creating json payload: %s", err.Error())
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create AWS Integration Payload: %s", string(debugOutput))

	int, err := config.Client.CreateAWSCloudWatchIntegration(payload)
	if err != nil {
		return err
	}
	d.SetId(int.Id)
	if err := d.Set("external_id", int.ExternalId); err != nil {
		return err
	}

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

	int, err := config.Client.UpdateAWSCloudWatchIntegration(d.Id(), payload)
	if err != nil {
		return err
	}
	d.SetId(int.Id)
	if err := d.Set("external_id", int.ExternalId); err != nil {
		return err
	}

	return awsIntegrationAPIToTF(d, int)
}

func integrationAWSDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteAWSCloudWatchIntegration(d.Id())
}

func validateAuthMethod(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	if value != string(integration.EXTERNAL_ID) && value != string(integration.SECURITY_TOKEN) {
		errors = append(errors, fmt.Errorf("%s not allowed; auth method must be one of %s or %s", value, integration.EXTERNAL_ID, integration.SECURITY_TOKEN))
	}
	return
}

func validateFilterAction(v interface{}, k string) (we []string, errors []error) {
	value := v.(string)
	if value != string(integration.EXCLUDE) && value != string(integration.INCLUDE) {
		errors = append(errors, fmt.Errorf("%s not allowed; filter action must be one of %s or %s", value, integration.EXCLUDE, integration.INCLUDE))
	}
	return
}

func validateAwsPollRate(v interface{}, k string) (we []string, errors []error) {
	value := v.(int)
	if value != 60 && value != 300 {
		errors = append(errors, fmt.Errorf("%d not allowed; Use one of 60 or 300.", value))
		return
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
