package signalfx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/signalfx/signalfx-go/integration"
)

const (
	IntegrationAppPath = "/integration/"
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
				ConflictsWith: []string{"custom_namespace_sync_rules"},
			},
			"custom_namespace_sync_rules": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"custom_cloudwatch_namespaces"},
				Description:   "Each element controls the data collected by SignalFx for the specified namespace. If you specify this property, SignalFx ignores values in the \"custom_cloud_watch_namespaces\" property.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_action": {
							Type:         schema.TypeString,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the SignalFx default behavior for processing data from an AWS namespace. SignalFx ignores this property unless you specify the \"filter\" property in the namespace sync rule. If you do specify a filter, use this property to control how SignalFx treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateFilterAction,
										Description:  "Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
									},
									"source": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Expression that selects the data that SignalFx should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.",
									},
								},
							},
						},
						"namespace": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "An AWS custom namespace having custom AWS metrics that you want to sync with SignalFx. See the AWS documentation on publishing metrics for more information.",
						},
					},
				},
			},
			"namespace_sync_rules": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "Each element in the array is an object that contains an AWS namespace name and a filter that controls the data that SignalFx collects for the namespace. If you specify this property, SignalFx ignores the values in the AWS CloudWatch Integration Model \"services\" property. If you don't specify either property, SignalFx syncs all data in all AWS namespaces.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_action": {
							Type:         schema.TypeString,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the SignalFx default behavior for processing data from an AWS namespace. SignalFx ignores this property unless you specify the \"filter\" property in the namespace sync rule. If you do specify a filter, use this property to control how SignalFx treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter": {
							Type: schema.TypeSet,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action": {
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: validateFilterAction,
										Description:  "Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
									},
									"source": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Expression that selects the data that SignalFx should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.",
									},
								},
							},
						},
						"namespace": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateAwsService,
							Description:  "An AWS custom namespace having custom AWS metrics that you want to sync with SignalFx. See the AWS documentation on publishing metrics for more information.",
						},
					},
				},
			},
			"enable_aws_usage": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag that controls how SignalFx imports usage metrics from AWS to use with AWS Cost Optimizer. If `true`, SignalFx imports the metrics.",
			},
			"enable_check_large_volume": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag that controls how SignalFx checks for large amounts of data for this AWS integration. If `true`, SignalFx checks to see if the integration is returning a large amount of data.",
			},
			"import_cloud_watch": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag that controls how SignalFx imports Cloud Watch metrics. If true, SignalFx imports Cloud Watch metrics from AWS.",
			},
			"is_large_volume": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If true, this property indicates that SignalFx is receiving a large volume of data and tags from AWS.",
			},
			"key": {
				Type:          schema.TypeString,
				Required:      true,
				Description:   "If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the key.",
				ConflictsWith: []string{"token"},
			},
			"regions": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Array of AWS regions that SignalFx should monitor.",
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
				Description: "Array of AWS services that you want SignalFx to monitor. Each element is a string designating an AWS service.",
			},
			"token": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the token.",
				ConflictsWith: []string{"key"},
			},
			"poll_rate": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "AWS poll rate (in seconds)",
				ValidateFunc: validateAwsPollRate,
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
		if strings.Contains(err.Error(), "Bad status 404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func integrationAWSRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	det, err := config.Client.GetAWSCloudWatchIntegration(d.Id())
	if err != nil {
		if strings.HasPrefix(err.Error(), "Bad status 404") {
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

	return awsIntegrationAPIToTF(d, det)
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
	if err := d.Set("enable_check_large_volume", aws.EnableCheckLargeVolume); err != nil {
		return err
	}
	if err := d.Set("import_cloud_watch", aws.ImportCloudWatch); err != nil {
		return err
	}
	if err := d.Set("is_large_volume", aws.IsLargeVolume); err != nil {
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
	if len(aws.Services) > 0 {
		services := make([]interface{}, len(aws.Services))
		for i, v := range aws.Services {
			services[i] = string(v)
		}
		if err := d.Set("services", services); err != nil {
			return err
		}
	}
	if aws.CustomCloudWatchNamespaces != "" {
		nesses := strings.Split(aws.CustomCloudWatchNamespaces, ",")
		if err := d.Set("custom_cloudwatch_namespaces", flattenStringSliceToSet(nesses)); err != nil {
			return err
		}
	}
	if len(aws.CustomNamespaceSyncRules) > 0 {
		rules := make([]map[string]interface{}, len(aws.CustomNamespaceSyncRules))
		for i, v := range aws.CustomNamespaceSyncRules {
			rules[i] = map[string]interface{}{
				"default_action": string(v.DefaultAction),
				"filter": map[string]interface{}{
					"action": v.Filter.Action,
					"source": v.Filter.Source,
				},
				"namespace": v.Namespace,
			}
		}
		if err := d.Set("custom_namespace_sync_rules", rules); err != nil {
			return err
		}
	}
	if len(aws.NamespaceSyncRules) > 0 {
		rules := make([]map[string]interface{}, len(aws.NamespaceSyncRules))
		for i, v := range aws.NamespaceSyncRules {
			rules[i] = map[string]interface{}{
				"default_action": string(v.DefaultAction),
				"filter": map[string]interface{}{
					"action": v.Filter.Action,
					"source": v.Filter.Source,
				},
				"namespace": string(v.Namespace),
			}
		}
		if err := d.Set("namespace_sync_rules", rules); err != nil {
			return err
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
		EnableCheckLargeVolume: d.Get("enable_check_large_volume").(bool),
		ImportCloudWatch:       d.Get("import_cloud_watch").(bool),
		IsLargeVolume:          d.Get("is_large_volume").(bool),
		Key:                    d.Get("key").(string),
		RoleArn:                d.Get("role_arn").(string),
		Token:                  d.Get("token").(string),
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
		for _, ns := range val.([]interface{}) {
			cwns = append(cwns, ns.(string))
		}
		aws.CustomCloudWatchNamespaces = strings.Join(cwns, ",")
	}

	if val, ok := d.GetOk("custom_namespace_sync_rules"); ok {
		val := val.([]map[string]interface{})
		aws.CustomNamespaceSyncRules = getCustomNamespaceRules(val)
	}

	if val, ok := d.GetOk("namespace_sync_rules"); ok {
		val := val.([]map[string]interface{})
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

func getCustomNamespaceRules(tfRules []map[string]interface{}) []*integration.AwsCustomNameSpaceSyncRule {
	rules := make([]*integration.AwsCustomNameSpaceSyncRule, len(tfRules))
	for i, r := range tfRules {
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
		if f, fok := r["filter"]; fok {
			f := f.(map[string]interface{})
			filter = &integration.AwsSyncRuleFilter{
				Action: integration.AwsSyncRuleFilterAction(f["action"].(string)),
				Source: f["source"].(string),
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

func getNamespaceRules(tfRules []map[string]interface{}) []*integration.AwsNameSpaceSyncRule {
	rules := make([]*integration.AwsNameSpaceSyncRule, len(tfRules))
	for i, r := range tfRules {
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
		if f, fok := r["filter"]; fok {
			f := f.(map[string]interface{})
			filter = &integration.AwsSyncRuleFilter{
				Action: integration.AwsSyncRuleFilterAction(f["action"].(string)),
				Source: f["source"].(string),
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
	// Since things worked, set the URL and move on
	appURL, err := buildAppURL(config.CustomAppURL, IntegrationAppPath+int.Id)
	if err != nil {
		return err
	}
	if err := d.Set("url", appURL); err != nil {
		return err
	}
	d.SetId(int.Id)

	return awsIntegrationAPIToTF(d, int)
}

func integrationAWSUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
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
