// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/integration"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
	"go.uber.org/multierr"
)

type stateSupplierFunc = func(*integration.AwsCloudWatchIntegration) string

func metricStreamsStateSupplier(int *integration.AwsCloudWatchIntegration) string {
	return int.MetricStreamsSyncState
}

func logsSyncStateSupplier(int *integration.AwsCloudWatchIntegration) string {
	return int.LogsSyncState
}

func integrationAWSResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"integration_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of this integration",
				ForceNew:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the integration. Please specify the name in `signalfx_aws_external_integration` or `signalfx_aws_integration_token`",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the integration is enabled or not",
			},
			"auth_method": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The mechanism used to authenticate with AWS. Use one of `signalfx_aws_external_integration` or `signalfx_aws_token_integration` to define this",
			},
			"custom_cloudwatch_namespaces": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:      true,
				Description:   "List of custom AWS CloudWatch namespaces to monitor. Custom namespaces contain custom metrics that you define in AWS; Splunk Observability imports the metrics so you can monitor them.",
				ConflictsWith: []string{"custom_namespace_sync_rule"},
			},
			"custom_namespace_sync_rule": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"custom_cloudwatch_namespaces"},
				Description:   "Each element controls the data collected by Splunk Observability for the specified namespace. If you specify this property, Splunk Observability ignores values in the \"custom_cloudwatch_namespaces\" property.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the Splunk Observability default behavior for processing data from an AWS namespace. Splunk Observability ignores this property unless you specify the `filter_action` and `filter_source` properties. If you do specify them, use this property to control how Splunk Observability treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls how Splunk Observability processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_source": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Expression that selects the data that Splunk Observability should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "An AWS custom namespace having custom AWS metrics that you want to sync with Splunk Observability. See the AWS documentation on publishing metrics for more information.",
						},
					},
				},
			},
			"namespace_sync_rule": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"services"},
				Description:   "Each element in the array is an object that contains an AWS namespace name and a filter that controls the data that Splunk Observability collects for the namespace. If you specify this property, Splunk Observability ignores the values in the AWS CloudWatch Integration Model \"services\" property. If you don't specify either property, Splunk Observability syncs all data in all AWS namespaces.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"default_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls the Splunk Observability default behavior for processing data from an AWS namespace. Splunk Observability ignores this property unless you specify the `filter_action` and `filter_source` properties. If you do specify them, use this property to control how Splunk Observability treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_action": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validateFilterAction,
							Description:  "Controls how Splunk Observability processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
						},
						"filter_source": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Expression that selects the data that Splunk Observability should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.",
						},
						"namespace": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "An AWS namespace having custom AWS metrics that you want to sync with Splunk Observability. See the AWS documentation on publishing metrics for more information.",
						},
					},
				},
			},
			"enable_aws_usage": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag that controls how Splunk Observability imports usage metrics from AWS to use with AWS Cost Optimizer. If `true`, Splunk Observability imports the metrics.",
			},
			"import_cloud_watch": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag that controls how Splunk Observability imports Cloud Watch metrics. If true, Splunk Observability imports Cloud Watch metrics from AWS.",
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
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of AWS regions that Splunk Observability should monitor.",
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
					Type: schema.TypeString,
				},
				Description: "List of AWS services that you want Splunk Observability to monitor. Each element is a string designating an AWS service.",
			},
			"token": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"role_arn", "external_id"},
				Description:   "Used with `signalfx_aws_token_integration`. Use this property to specify the token.",
			},
			"poll_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      300,
				Description:  "AWS poll rate (in seconds). Between `60` and `600`.",
				ValidateFunc: validation.IntBetween(60, 600),
			},
			"cold_poll_rate": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "AWS cold poll rate (in seconds). Between `60` and `3600`",
				ValidateFunc: validation.IntBetween(60, 3600),
			},
			"external_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"token", "key"},
				Description:   "Used with `signalfx_aws_external_integration`. Use this property to specify the external id.",
			},
			"use_metric_streams_sync": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enables the use of Cloudwatch Metric Streams for metrics synchronization.",
			},
			"enable_logs_sync": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enables AWS logs synchronization.",
			},
			"named_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A named token to use for ingest",
			},
			"enable_check_large_volume": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Controls how Splunk Observability checks for large amounts of data for this AWS integration. If true, Splunk Observability monitors the amount of data coming in from the integration.",
			},
			"metric_stats_to_sync": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Each element in the array is an object that contains an AWS namespace name, AWS metric name and a list of statistics that Splunk Observability collects for this metric. If you specify this property, Splunk Observability retrieves only specified AWS statistics. If you don't specify this property, Splunk Observability retrieves the AWS standard set of statistics.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"namespace": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "An AWS namespace having AWS metric that you want to pick statistics for",
						},
						"metric": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "AWS metric that you want to pick statistics for",
						},
						"stats": {
							Type:        schema.TypeSet,
							Required:    true,
							Description: "AWS statistics you want to collect",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"sync_custom_namespaces_only": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Indicates that Splunk Observability should sync metrics and metadata from custom AWS namespaces only (see the `custom_namespace_sync_rule` field for details). Defaults to `false`.",
			},
			"collect_only_recommended_stats": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Indicates that Splunk Observability should only sync recommended statistics",
			},
			"metric_streams_managed_externally": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				Description: "If set to true, Splunk Observability Cloud accepts data from Metric Streams managed from the AWS console. " +
					"The AWS account sending the Metric Streams and the AWS account in the Splunk Observability Cloud integration have to match." +
					"Requires `use_metric_streams_sync` set to true to work.",
			},
		},

		Create: integrationAWSCreate,
		Read:   integrationAWSRead,
		Update: integrationAWSUpdate,
		Delete: integrationAWSDelete,
	}
}

func integrationAWSRead(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)

	int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Get("integration_id").(string))
	if err != nil {
		if isNotFoundError(err) {
			d.SetId("")
			return nil
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

	err := multierr.Combine(
		d.Set("integration_id", aws.Id),
		d.Set("name", aws.Name),
		d.Set("enabled", aws.Enabled),
		d.Set("enable_aws_usage", aws.EnableAwsUsage),
		d.Set("import_cloud_watch", aws.ImportCloudWatch),
		d.Set("poll_rate", aws.PollRate/1000),
		d.Set("use_metric_streams_sync", aws.MetricStreamsSyncState == "ENABLED"),
		d.Set("enable_logs_sync", aws.LogsSyncState == "ENABLED"),
		d.Set("enable_check_large_volume", aws.EnableCheckLargeVolume),
		d.Set("sync_custom_namespaces_only", aws.SyncCustomNamespacesOnly),
		d.Set("named_token", aws.NamedToken),
		d.Set("collect_only_recommended_stats", aws.CollectOnlyRecommendedStats),
		d.Set("metric_streams_managed_externally", aws.MetricStreamsManagedExternally),
	)

	if err != nil {
		return err
	}

	if aws.ColdPollRate != 0 {
		if err := d.Set("cold_poll_rate", aws.ColdPollRate/1000); err != nil {
			return err
		}
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
		var rules []map[string]any
		for _, v := range aws.CustomNamespaceSyncRules {
			rule := map[string]any{
				"default_action": string(v.DefaultAction),
				"namespace":      v.Namespace,
			}
			if v.Filter != nil {
				rule["filter_action"] = v.Filter.Action
				rule["filter_source"] = v.Filter.Source
			}
			rules = append(rules, rule)
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
			services := make([]any, len(aws.Services))
			for i, v := range aws.Services {
				services[i] = string(v)
			}
			if err := d.Set("services", services); err != nil {
				return err
			}
		}
	} else {
		if len(aws.NamespaceSyncRules) > 0 {
			var rules []map[string]any
			for _, v := range aws.NamespaceSyncRules {
				if v.Filter != nil {
					rules = append(rules, map[string]any{
						"default_action": string(v.DefaultAction),
						"filter_action":  v.Filter.Action,
						"filter_source":  v.Filter.Source,
						"namespace":      string(v.Namespace),
					})
				} else {
					// Sometimes the rules come back with just a namespace and no
					// filters.
					rules = append(rules, map[string]any{
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

	if len(aws.MetricStatsToSync) > 0 {
		var metricStatsToSync []map[string]any
		for namespace, v := range aws.MetricStatsToSync {
			for metric, stats := range v {
				metricStatsToSync = append(metricStatsToSync, map[string]any{
					"namespace": namespace,
					"metric":    metric,
					"stats":     stats,
				})
			}
		}
		if len(metricStatsToSync) > 0 {
			if err := d.Set("metric_stats_to_sync", metricStatsToSync); err != nil {
				return err
			}
		}
	}

	return nil
}

func getPayloadAWSIntegration(d *schema.ResourceData) (*integration.AwsCloudWatchIntegration, error) {

	aws := &integration.AwsCloudWatchIntegration{
		Name:                           d.Get("name").(string),
		Type:                           "AWSCloudWatch",
		Enabled:                        d.Get("enabled").(bool),
		EnableAwsUsage:                 d.Get("enable_aws_usage").(bool),
		ImportCloudWatch:               d.Get("import_cloud_watch").(bool),
		EnableCheckLargeVolume:         d.Get("enable_check_large_volume").(bool),
		SyncCustomNamespacesOnly:       d.Get("sync_custom_namespaces_only").(bool),
		CollectOnlyRecommendedStats:    d.Get("collect_only_recommended_stats").(bool),
		MetricStreamsManagedExternally: d.Get("metric_streams_managed_externally").(bool),
	}

	if d.Get("use_metric_streams_sync").(bool) {
		aws.MetricStreamsSyncState = "ENABLED"
	} else if d.HasChange("use_metric_streams_sync") {
		aws.MetricStreamsSyncState = "CANCELLING" // use_metric_streams_sync is false, and it has changed, meaning it was ENABLED before
	}

	if d.Get("enable_logs_sync").(bool) {
		aws.LogsSyncState = "ENABLED"
	} else if d.HasChange("enable_logs_sync") {
		aws.LogsSyncState = "CANCELLING" // enable_logs_sync is false, and it has changed, meaning it was ENABLED before
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

	if val, ok := d.GetOk("cold_poll_rate"); ok {
		aws.ColdPollRate = int64(val.(int)) * 1000
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
	} else {
		return nil, fmt.Errorf("regions should be defined explicitly, see https://docs.splunk.com/Observability/gdi/get-data-in/connect/aws/aws-prereqs.html#supported-aws-regions")
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

	if val, ok := d.GetOk("metric_stats_to_sync"); ok {
		val := val.(*schema.Set).List()
		if len(val) > 0 {
			metricStatsToSync := map[string]map[string][]string{}
			for _, v := range val {
				v := v.(map[string]any)
				namespace := v["namespace"].(string)
				metric := v["metric"].(string)
				stats := convert.SchemaListAll(v["stats"], convert.ToString)

				if metricStatsToSync[namespace] == nil {
					metricStatsToSync[namespace] = map[string][]string{}
				}
				metricStatsToSync[namespace][metric] = stats
			}
			aws.MetricStatsToSync = metricStatsToSync
		}
	}

	return aws, nil
}

func getCustomNamespaceRules(tfRules []any) []*integration.AwsCustomNameSpaceSyncRule {
	rules := make([]*integration.AwsCustomNameSpaceSyncRule, len(tfRules))
	for i, r := range tfRules {
		r := r.(map[string]any)
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

func getNamespaceRules(tfRules []any) []*integration.AwsNameSpaceSyncRule {
	rules := make([]*integration.AwsNameSpaceSyncRule, len(tfRules))
	for i, r := range tfRules {
		r := r.(map[string]any)
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

func integrationAWSCreate(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)

	preInt, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Get("integration_id").(string))
	if err != nil {
		return fmt.Errorf("Error fetching existing integration %s, %s", d.Get("integration_id").(string), err.Error())
	}
	if preInt.AuthMethod == integration.EXTERNAL_ID {
		if err := d.Set("external_id", preInt.ExternalId); err != nil {
			return err
		}
	}
	if err := d.Set("name", preInt.Name); err != nil {
		return err
	}
	d.SetId(preInt.Id)

	return integrationAWSUpdate(d, meta)
}

func integrationAWSUpdate(d *schema.ResourceData, meta any) error {
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

	if d.HasChange("use_metric_streams_sync") {
		if int, err = waitForIntegrationStateToSettle(d, config, int.Id, "use_metric_streams_sync", metricStreamsStateSupplier); err != nil {
			return err
		}
	}
	if d.HasChange("enable_logs_sync") {
		if int, err = waitForIntegrationStateToSettle(d, config, int.Id, "enable_logs_sync", logsSyncStateSupplier); err != nil {
			return err
		}
	}

	return awsIntegrationAPIToTF(d, int)
}

func DoIntegrationAWSDelete(d *schema.ResourceData, meta any) error {
	config := meta.(*signalfxConfig)

	// Retrieve current integration state
	int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), d.Id())
	if err != nil {
		var re *signalfx.ResponseError
		if errors.As(err, &re) && re.Code() == http.StatusNotFound {
			log.Printf("[DEBUG] SignalFx: integration %s not found (already deleted?), skipping cleanup procedure", d.Id())
			return nil
		}
		return fmt.Errorf("error fetching existing integration %s, %s", d.Id(), err.Error())
	}

	// Disable the AWS logs sync and/or CloudWatch metric streams sync if needed
	needToDisableMetricStreams := int.Enabled && int.MetricStreamsSyncState != "" && int.MetricStreamsSyncState != "DISABLED"
	needToDisableLogsSync := int.Enabled && int.LogsSyncState != "" && int.LogsSyncState != "DISABLED"
	if needToDisableMetricStreams || needToDisableLogsSync {
		if needToDisableMetricStreams {
			int.MetricStreamsSyncState = "CANCELLING"
		}
		if needToDisableLogsSync {
			int.LogsSyncState = "CANCELLING"
		}
		_, err := config.Client.UpdateAWSCloudWatchIntegration(context.TODO(), d.Id(), int)
		if err != nil {
			if strings.Contains(err.Error(), "40") {
				err = fmt.Errorf("%s\nPlease verify you are using an admin token when working with integrations", err.Error())
			}
			return err
		}
		if needToDisableMetricStreams {
			if _, err = waitForIntegrationSpecificSyncStateToSettle(d, false, config, int.Id, "use_metric_streams_sync", metricStreamsStateSupplier); err != nil {
				return err
			}
		}
		if needToDisableLogsSync {
			if _, err = waitForIntegrationSpecificSyncStateToSettle(d, false, config, int.Id, "enable_logs_sync", logsSyncStateSupplier); err != nil {
				return err
			}
		}
	}

	return config.Client.DeleteAWSCloudWatchIntegration(context.TODO(), d.Id())
}

func waitForIntegrationStateToSettle(d *schema.ResourceData, config *signalfxConfig, intId string, syncStateField string,
	stateSupplier stateSupplierFunc) (*integration.AwsCloudWatchIntegration, error) {
	return waitForIntegrationSpecificSyncStateToSettle(d, d.Get(syncStateField).(bool), config, intId, syncStateField, stateSupplier)
}

func waitForIntegrationSpecificSyncStateToSettle(d *schema.ResourceData, syncState bool, config *signalfxConfig, intId string, syncStateField string,
	stateSupplier stateSupplierFunc) (*integration.AwsCloudWatchIntegration, error) {
	var pending, target []string
	var expectedState string
	if syncState {
		expectedState = "enabled"
		pending = []string{"DISABLED", "CANCELLING", "CANCELLATION_FAILED"}
		target = []string{"ENABLED"}
	} else {
		expectedState = "disabled"
		pending = []string{"ENABLED", "CANCELLING"}
		target = []string{"", "DISABLED"}
	}

	stateConf := &resource.StateChangeConf{
		Pending: pending,
		Target:  target,
		Refresh: func() (any, string, error) {
			int, err := config.Client.GetAWSCloudWatchIntegration(context.TODO(), intId)
			if err != nil {
				return 0, "", err
			}
			return int, stateSupplier(int), nil
		},
		Timeout:    d.Timeout(schema.TimeoutUpdate) - time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	int, err := stateConf.WaitForState()
	if err != nil {
		return nil, fmt.Errorf("Error waiting for integration %s state for %s to become %s: %s", intId, syncStateField, expectedState, err)
	}
	return int.(*integration.AwsCloudWatchIntegration), nil
}

func integrationAWSDelete(d *schema.ResourceData, meta any) error {
	return DoIntegrationAWSDelete(d, meta)
}

func noop(_ *schema.ResourceData, _ any) error {
	return nil
}

func validateFilterAction(v any, _ string) (we []string, errors []error) {
	value := v.(string)
	if value != string(integration.EXCLUDE) && value != string(integration.INCLUDE) {
		errors = append(errors, fmt.Errorf("%s not allowed; filter action must be one of %s or %s", value, integration.EXCLUDE, integration.INCLUDE))
	}
	return
}
