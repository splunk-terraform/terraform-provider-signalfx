---
layout: "signalfx"
page_title: "SignalFx: signalfx_aws_integration"
sidebar_current: "docs-signalfx-resource-aws-integration"
description: |-
  Allows Terraform to create and manage SignalFx AWS Integrations
---

# Resource: signalfx_aws_integration

SignalFx AWS CloudWatch integrations. For help with this integration see [Monitoring Amazon Web Services](https://docs.signalfx.com/en/latest/integrations/amazon-web-services.html#monitor-amazon-web-services).

~> **NOTE** When managing integrations use a session token for an administrator to authenticate the SignalFx provider. See [Operations that require a session token for an administrator].(https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator).

~> **WARNING** This resource implements a part of a workflow. You must use it with one of either `signalfx_aws_external_integration` or `signalfx_aws_token_integration`.

## Example Usage

```tf
// This resource returns an account id in `external_id`…
resource "signalfx_aws_external_integration" "aws_myteam_external" {
  name = "My AWS integration"
}

// Make yourself an AWS IAM role here, use `signalfx_aws_external_integration.aws_myteam_external.external_id`
resource "aws_iam_role" "aws_sfx_role" {
  // Stuff here that uses the external and account ID
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

## Service Names

~> **NOTE** You can use the data source "signalfx_aws_services" to specify all services.

## Argument Reference

* `enabled` - (Required) Whether the integration is enabled.
* `integration_id` - (Required) The id of one of a `signalfx_aws_external_integration` or `signalfx_aws_token_integration`.
* `external_id` - (Required) The `external_id` property from one of a `signalfx_aws_external_integration` or `signalfx_aws_token_integration`
* `custom_cloudwatch_namespaces` - (Optional) List of custom AWS CloudWatch namespaces to monitor. Custom namespaces contain custom metrics that you define in AWS; SignalFx imports the metrics so you can monitor them.
* `custom_namespace_sync_rule` - (Optional) Each element controls the data collected by SignalFx for the specified namespace. Conflicts with the `custom_cloudwatch_namespaces` property.
  * `default_action` - (Optional) Controls the SignalFx default behavior for processing data from an AWS namespace. If you do specify a filter, use this property to control how SignalFx treats data that doesn't match the filter. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_action` - (Optional) Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_source` - (Optional) Expression that selects the data that SignalFx should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.
  * `namespace` - (Required) An AWS custom namespace having custom AWS metrics that you want to sync with SignalFx. See the AWS documentation on publishing metrics for more information.
* `namespace_sync_rule` - (Optional) Each element in the array is an object that contains an AWS namespace name and a filter that controls the data that SignalFx collects for the namespace. Conflicts with the `services` property. If you don't specify either property, SignalFx syncs all data in all AWS namespaces.
  * `default_action` - (Optional) Controls the SignalFx default behavior for processing data from an AWS namespace. If you do specify a filter, use this property to control how SignalFx treats data that doesn't match the filter. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_action` - (Optional) Controls how SignalFx processes data from a custom AWS namespace. The available actions are one of `"Include"` or `"Exclude"`.
  * `filter_source` - (Optional) Expression that selects the data that SignalFx should sync for the custom namespace associated with this sync rule. The expression uses the syntax defined for the SignalFlow `filter()` function; it can be any valid SignalFlow filter expression.
  * `namespace` - (Required) An AWS custom namespace having custom AWS metrics that you want to sync with SignalFx. See the AWS documentation on publishing metrics for more information.
* `enable_aws_usage` - (Optional) Flag that controls how SignalFx imports usage metrics from AWS to use with AWS Cost Optimizer. If `true`, SignalFx imports the metrics.
* `import_cloud_watch` - (Optional) Flag that controls how SignalFx imports Cloud Watch metrics. If true, SignalFx imports Cloud Watch metrics from AWS.
* `key` - (Optional) If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the key (this is typically equivalent to the `AWS_SECRET_ACCESS_KEY` environment variable).
* `token` - (Optional) If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the token (this is typically equivalent to the `AWS_ACCESS_KEY_ID` environment variable).
* `regions` - (Optional) List of AWS regions that SignalFx should monitor.
* `role_arn` - (Optional) Role ARN that you add to an existing AWS integration object. **Note**: Ensure you use the `arn` property of your role, not the id!
* `services` - (Optional) List of AWS services that you want SignalFx to monitor. Each element is a string designating an AWS service. Conflicts with `namespace_sync_rule`. See the documentation for [Creating Integrations](https://developers.signalfx.com/integrations_reference.html#operation/Create%20Integration) for valida values.
* `poll_rate` - (Optional) AWS poll rate (in seconds). Value between `60` and `600`. Default: `300`.
* `use_get_metric_data_method` - (Optional) Enable the use of Amazon's `GetMetricData` for collecting metrics. Note that this requires the inclusion of the `"cloudwatch:GetMetricData"` permission.
* `use_metric_streams_sync` - (Optional) Enable the use of Amazon Cloudwatch Metric Streams for ingesting metrics. Note that this requires the inclusion of `"cloudwatch:ListMetricStreams"`,`"cloudwatch:GetMetricStream"`, `"cloudwatch:PutMetricStream"`, `"cloudwatch:DeleteMetricStream"`, `"cloudwatch:StartMetricStreams"`, `"cloudwatch:StopMetricStreams"` and `"iam:PassRole"` permissions.
* `enable_logs_sync` - (Optional) Enable the AWS logs synchronization. Note that this requires the inclusion of `"logs:DescribeLogGroups"`,  `"logs:DeleteSubscriptionFilter"`, `"logs:DescribeSubscriptionFilters"`, `"logs:PutSubscriptionFilter"`, and `"s3:GetBucketLogging"`,  `"s3:GetBucketNotification"`, `"s3:PutBucketNotification"` permissions. Additional permissions may be required to capture logs from specific AWS services.
* `enable_check_large_volume` - (Optional) Controls how SignalFx checks for large amounts of data for this AWS integration. If `true`, SignalFx monitors the amount of data coming in from the integration.
* `metric_stats_to_sync` - (Optional) Each element in the array is an object that contains an AWS namespace name, AWS metric name and a list of statistics that SignalFx collects for this metric. If you specify this property, SignalFx retrieves only specified AWS statistics. If you don't specify this property, SignalFx retrieves the AWS standard set of statistics.
  * `namespace` - (Required) An AWS namespace having AWS metric that you want to pick statistics for
  * `metric` - (Required) AWS metric that you want to pick statistics for
  * `stats` - (Required) AWS statistics you want to collect
