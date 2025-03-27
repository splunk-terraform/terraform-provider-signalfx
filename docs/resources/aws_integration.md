---
page_title: "Splunk Observability Cloud: signalfx_aws_integration"
description: |-
  Allows Terraform to create and manage AWS Integrations for Splunk Observability Cloud
---

# Resource: signalfx_aws_integration

AWS CloudWatch integrations for Splunk Observability Cloud. For help with this integration see [Monitoring Amazon Web Services](https://docs.splunk.com/observability/en/gdi/get-data-in/connect/aws/get-awstoc.html).

This resource implements a part of a workflow. Use it with one of either `signalfx_aws_external_integration` or `signalfx_aws_token_integration`.

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Splunk Observability provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator).

## Example

```terraform
#  This resource returns an account id in `external_id`…
resource "signalfx_aws_external_integration" "aws_myteam_external" {
  name = "My AWS integration"
}

# Make yourself an AWS IAM role here, use `signalfx_aws_external_integration.aws_myteam_external.external_id`
resource "aws_iam_role" "aws_sfx_example" {
  #  Stuff here that uses the external and account ID
}

resource "signalfx_aws_integration" "aws_myteam" {
  enabled = true

  integration_id     = signalfx_aws_external_integration.aws_myteam_external.id
  external_id        = signalfx_aws_external_integration.aws_myteam_external.external_id
  role_arn           = aws_iam_role.aws_sfx_role.arn
  regions            = ["us-east-1"]
  poll_rate          = 300
  import_cloud_watch = true
  enable_aws_usage   = true

  custom_namespace_sync_rule {
    default_action = "Exclude"
    filter_action  = "Include"
    filter_source  = "filter('code', '200')"
    namespace      = "my-custom-namespace"
  }

  namespace_sync_rule {
    default_action = "Exclude"
    filter_action  = "Include"
    filter_source  = "filter('code', '200')"
    namespace      = "AWS/EC2"
  }

  metric_stats_to_sync {
    namespace  = "AWS/EC2"
    metric     = "NetworkPacketsIn"
    stats      = ["upper"]
  }
}
```

## Arguments

* `enable_aws_usage` - (Optional) Flag that controls how Splunk Observability Cloud imports usage metrics from AWS to use with AWS Cost Optimizer. If `true`, Splunk Observability Cloud imports the metrics.
* `enable_check_large_volume` - (Optional) Controls how Splunk Observability Cloud checks for large amounts of data for this AWS integration. If `true`, Splunk Observability Cloud monitors the amount of data coming in from the integration.
* `enable_logs_sync` - (Optional) Enable the AWS logs synchronization. Note that this requires the inclusion of `"logs:DescribeLogGroups"`, `"logs:DeleteSubscriptionFilter"`, `"logs:DescribeSubscriptionFilters"`, `"logs:PutSubscriptionFilter"`, and `"s3:GetBucketLogging"`, `"s3:GetBucketNotification"`, `"s3:PutBucketNotification"` permissions. Additional permissions may be required to capture logs from specific AWS services.
* `enabled` - (Required) Whether the integration is enabled.
* `external_id` - (Required) The `external_id` property from one of a `signalfx_aws_external_integration` or `signalfx_aws_token_integration`
* `custom_cloudwatch_namespaces` - (Optional) List of custom AWS CloudWatch namespaces to monitor. Custom namespaces contain custom metrics that you define in AWS; Splunk Observability Cloud imports the metrics so you can monitor them.
* `custom_namespace_sync_rule` - (Optional) Each element controls the data collected by Splunk Observability Cloud for the specified namespace. Conflicts with the `custom_cloudwatch_namespaces` property.
  * `default_action` - (Optional) Controls the Splunk Observability Cloud default behavior for processing data from an AWS namespace. Splunk Observability Cloud ignores this property unless you specify the `filter_action` and `filter_source` properties. If you do specify them, use this property to control how Splunk Observability Cloud treats data that doesn't match the filter. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_action` - (Optional) Controls how Splunk Observability Cloud processes data from a custom AWS namespace. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_source` - (Optional) Expression that selects the data that Splunk Observability Cloud should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.
  * `namespace` - (Required) An AWS custom namespace having custom AWS metrics that you want to sync with Splunk Observability Cloud. See the AWS documentation on publishing metrics for more information.
* `import_cloud_watch` - (Optional) Flag that controls how Splunk Observability Cloud imports Cloud Watch metrics. If true, Splunk Observability Cloud imports Cloud Watch metrics from AWS.
* `integration_id` - (Required) The id of one of a `signalfx_aws_external_integration` or `signalfx_aws_token_integration`.
* `key` - (Optional) If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the key (this is typically equivalent to the `AWS_SECRET_ACCESS_KEY` environment variable).
* `metric_stats_to_sync` - (Optional) Each element in the array is an object that contains an AWS namespace name, AWS metric name and a list of statistics that Splunk Observability Cloud collects for this metric. If you specify this property, Splunk Observability Cloud retrieves only specified AWS statistics when AWS metric streams are not used. When AWS metric streams are used this property specifies additional extended statistics to collect (please note that AWS metric streams API supports percentile stats only; other stats are ignored). If you don't specify this property, Splunk Observability Cloud retrieves the AWS standard set of statistics.
  * `metric` - (Required) AWS metric that you want to pick statistics for
  * `namespace` - (Required) An AWS namespace having AWS metric that you want to pick statistics for
  * `stats` - (Required) AWS statistics you want to collect
* `name` - (Required) Name of the integration.
* `named_token` - (Optional) Name of the org token to be used for data ingestion. If not specified then default access token is used.
* `namespace_sync_rule` - (Optional) Each element in the array is an object that contains an AWS namespace name and a filter that controls the data that Splunk Observability Cloud collects for the namespace. Conflicts with the `services` property. If you don't specify either property, Splunk Observability Cloud syncs all data in all AWS namespaces.
  * `default_action` - (Optional) Controls the Splunk Observability Cloud default behavior for processing data from an AWS namespace. Splunk Observability Cloud ignores this property unless you specify the `filter_action` and `filter_source` properties. If you do specify them, use this property to control how Splunk Observability Cloud treats data that doesn't match the filter. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_action` - (Optional) Controls how Splunk Observability Cloud processes data from a custom AWS namespace. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_source` - (Optional) Expression that selects the data that Splunk Observability Cloud should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.
  * `namespace` - (Required) An AWS custom namespace having custom AWS metrics that you want to sync with Splunk Observability Cloud. See `services` field description below for additional information.
* `poll_rate` - (Optional) AWS poll rate (in seconds). Value between `60` and `600`. Default: `300`.
* `regions` - (Required) List of AWS regions that Splunk Observability Cloud should monitor. It cannot be empty.
* `role_arn` - (Optional) Role ARN that you add to an existing AWS integration object. **Note**: Ensure you use the `arn` property of your role, not the id!
* `services` - (Optional) List of AWS services that you want Splunk Observability Cloud to monitor. Each element is a string designating an AWS service. Can be an empty list to import data for all supported services. Conflicts with `namespace_sync_rule`. See [Amazon Web Services](https://docs.splunk.com/Observability/gdi/get-data-in/integrations.html#amazon-web-services) for a list of valid values.
* `sync_custom_namespaces_only` - (Optional) Indicates that Splunk Observability Cloud should sync metrics and metadata from custom AWS namespaces only (see the `custom_namespace_sync_rule` above). Defaults to `false`.
* `token` - (Optional) If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the token (this is typically equivalent to the `AWS_ACCESS_KEY_ID` environment variable).
* `use_metric_streams_sync` - (Optional) Enable the use of Amazon Cloudwatch Metric Streams for ingesting metrics.<br> Note that this requires the inclusion of `"cloudwatch:ListMetricStreams"`,`"cloudwatch:GetMetricStream"`, `"cloudwatch:PutMetricStream"`, `"cloudwatch:DeleteMetricStream"`, `"cloudwatch:StartMetricStreams"`, `"cloudwatch:StopMetricStreams"` and `"iam:PassRole"` permissions.<br> Note you need to deploy additional resources on your AWS account to enable CloudWatch metrics streaming. Select one of the [CloudFormation templates](https://docs.splunk.com/Observability/gdi/get-data-in/connect/aws/aws-cloudformation.html) to deploy all the required resources.
* `collect_only_recommended_stats` - (Optional) The integration will only ingest the recommended statistics published by AWS
* `metric_streams_managed_externally` - (Optional) If set to true, Splunk Observability Cloud accepts data from Metric Streams managed from the AWS console. The AWS account sending the Metric Streams and the AWS account in the Splunk Observability Cloud integration have to match. Requires `use_metric_streams_sync` set to true to work.
