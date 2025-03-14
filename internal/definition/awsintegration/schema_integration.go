// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package awsintegration

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/signalfx/signalfx-go/integration"
	"go.uber.org/multierr"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/convert"
)

func newIntegrationSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
						Type:             schema.TypeString,
						Optional:         true,
						ValidateDiagFunc: check.FilterAction(),
						Description:      "Controls the Splunk Observability default behavior for processing data from an AWS namespace. Splunk Observability ignores this property unless you specify the `filter_action` and `filter_source` properties. If you do specify them, use this property to control how Splunk Observability treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
					},
					"filter_action": {
						Type:             schema.TypeString,
						Optional:         true,
						ValidateDiagFunc: check.FilterAction(),
						Description:      "Controls how Splunk Observability processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
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
						Type:             schema.TypeString,
						Optional:         true,
						ValidateDiagFunc: check.FilterAction(),
						Description:      "Controls the Splunk Observability default behavior for processing data from an AWS namespace. Splunk Observability ignores this property unless you specify the `filter_action` and `filter_source` properties. If you do specify them, use this property to control how Splunk Observability treats data that doesn't match the filter. The available actions are one of \"Include\" or \"Exclude\".",
					},
					"filter_action": {
						Type:             schema.TypeString,
						Optional:         true,
						ValidateDiagFunc: check.FilterAction(),
						Description:      "Controls how Splunk Observability processes data from a custom AWS namespace. The available actions are one of \"Include\" or \"Exclude\".",
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
	}
}

func decodeTerraform(rd *schema.ResourceData) (*integration.AwsCloudWatchIntegration, error) {
	cwi := &integration.AwsCloudWatchIntegration{
		Type:                           "AWSCloudWatch",
		Id:                             rd.Get("integration_id").(string),
		Name:                           rd.Get("name").(string),
		Enabled:                        rd.Get("enabled").(bool),
		EnableAwsUsage:                 rd.Get("enable_aws_usage").(bool),
		ImportCloudWatch:               rd.Get("import_cloud_watch").(bool),
		EnableCheckLargeVolume:         rd.Get("enable_check_large_volume").(bool),
		SyncCustomNamespacesOnly:       rd.Get("sync_custom_namespaces_only").(bool),
		CollectOnlyRecommendedStats:    rd.Get("collect_only_recommended_stats").(bool),
		MetricStreamsManagedExternally: rd.Get("metric_streams_managed_externally").(bool),
		Regions:                        convert.SchemaListAll(rd.Get("regions"), convert.ToString),
		Services:                       convert.SchemaListAll(rd.Get("services"), convert.ToStringLike[integration.AwsService]),
		NamespaceSyncRules:             convert.SchemaListAll(rd.Get("namespace_sync_rule"), convert.ToAWSNamespaceRule),
		CustomNamespaceSyncRules:       convert.SchemaListAll(rd.Get("custom_namespace_sync_rule"), convert.ToAWSCustomNamespaceRule),
		CustomCloudWatchNamespaces:     strings.Join(convert.SchemaListAll(rd.Get("custom_cloudwatch_namespaces"), convert.ToString), ","),
	}

	if v, ok := rd.GetOk("use_metric_streams_sync"); ok && v.(bool) {
		cwi.MetricStreamsSyncState = "ENABLED"
	} else if rd.HasChange("use_metric_streams_sync") {
		// use_metric_streams_sync is false, and it has changed, meaning it was ENABLED before
		cwi.MetricStreamsSyncState = "CANCELING"
	}

	if v, ok := rd.GetOk("enable_log_sync"); ok && v.(bool) {
		cwi.LogsSyncState = "ENABLED"
	} else if rd.HasChange("enable_log_sync") {
		// enable_logs_sync is false, and it has changed, meaning it was ENABLED before
		cwi.LogsSyncState = "CANCELING"
	}

	if v, ok := rd.GetOk("external_id"); ok && v != "" {
		cwi.AuthMethod = integration.EXTERNAL_ID
		cwi.ExternalId = v.(string)
		cwi.RoleArn = rd.Get("role_arn").(string)
	} else if v, ok := rd.GetOk("token"); ok && v != "" {
		cwi.AuthMethod = integration.SECURITY_TOKEN
		cwi.Token = v.(string)
		cwi.Key = rd.Get("key").(string)
	} else {
		return nil, fmt.Errorf("requires either `external_id` or `token` and `key`")
	}

	if cwi.Regions == nil {
		return nil, fmt.Errorf("regions should be defined explicitly, see https://docs.splunk.com/Observability/gdi/get-data-in/connect/aws/aws-prereqs.html#supported-aws-regions")
	}

	if val, ok := rd.GetOk("poll_rate"); ok {
		cwi.PollRate = int64(val.(int)) * 1000
	}

	if val, ok := rd.GetOk("named_token"); ok {
		cwi.NamedToken = val.(string)
	}

	if val, ok := rd.GetOk("metric_stats_to_sync"); ok && val.(*schema.Set).Len() > 0 {
		sync := make(map[string]map[string][]string)
		for _, v := range val.(*schema.Set).List() {
			var (
				v         = v.(map[string]any)
				namespace = v["namespace"].(string)
				metric    = v["metric"].(string)
				stats     = convert.SchemaListAll(v["stats"], convert.ToString)
			)
			if _, ok := sync[namespace]; !ok {
				sync[namespace] = make(map[string][]string)
			}
			sync[namespace][metric] = stats
		}
		cwi.MetricStatsToSync = sync
	}

	return cwi, nil
}

func encodeTerraform(aws *integration.AwsCloudWatchIntegration, d *schema.ResourceData) (errs error) {
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
	errs = multierr.Append(errs, err)

	if aws.Token != "" {
		errs = multierr.Append(errs, d.Set("token", aws.Token))
	}

	if len(aws.Regions) > 0 {
		errs = multierr.Append(errs, d.Set("regions", schema.NewSet(
			schema.HashString,
			convert.SliceAll(aws.Regions, convert.ToAny),
		)))
	}

	if len(aws.CustomNamespaceSyncRules) > 0 {
		errs = multierr.Append(errs, d.Set(
			"custom_namespace_sync_rule",
			convert.SliceAll(aws.CustomNamespaceSyncRules, func(ns *integration.AwsCustomNameSpaceSyncRule) map[string]any {
				rule := map[string]any{
					"default_action": string(ns.DefaultAction),
					"namespace":      ns.Namespace,
				}
				if ns.Filter != nil {
					rule["filter_action"] = ns.Filter.Action
					rule["filter_source"] = ns.Filter.Source
				}
				return rule
			})))
	} else if aws.CustomCloudWatchNamespaces != "" {
		errs = multierr.Append(errs, d.Set("custom_cloudwatch_namespaces", schema.NewSet(schema.HashString, convert.SliceAll(
			strings.Split(aws.CustomCloudWatchNamespaces, ","),
			convert.ToAny,
		))))
	}

	if _, ok := d.GetOk("services"); ok {
		errs = multierr.Append(errs, d.Set("services", convert.SliceAll(
			aws.Services,
			func(in integration.AwsService) any {
				return string(in)
			},
		)))
	} else if len(aws.NamespaceSyncRules) > 0 {
		errs = multierr.Append(errs, d.Set("namespace_sync_rule", convert.SliceAll(aws.NamespaceSyncRules, func(in *integration.AwsNameSpaceSyncRule) map[string]any {
			rule := map[string]any{
				"default_action": string(in.DefaultAction),
				"namespace":      in.Namespace,
			}
			if in.Filter != nil {
				rule["filter_action"] = in.Filter.Action
				rule["filter_source"] = in.Filter.Source
			}
			return rule
		})))
	}

	if len(aws.MetricStatsToSync) > 0 {
		sync := []map[string]any{}
		for namespace, v := range aws.MetricStatsToSync {
			for metric, stats := range v {
				sync = append(sync, map[string]any{
					"namespace": namespace,
					"metric":    metric,
					"stats":     stats,
				})
			}
		}
		errs = multierr.Append(errs, d.Set("metric_stats_to_sync", sync))
	}

	return errs
}
