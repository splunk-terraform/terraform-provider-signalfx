---
layout: "signalfx"
page_title: "SignalFx: signalfx_aws_integration"
sidebar_current: "docs-signalfx-resource-aws-integration"
description: |-
  Allows Terraform to create and manage SignalFx AWS Integrations
---

# Resource: signalfx_aws_integration

SignalFx AWS CloudWatch integrations. For help with this integration see [Monitoring Amazon Web Services](https://docs.signalfx.com/en/latest/integrations/amazon-web-services.html#monitor-amazon-web-services).

**Note:** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider.

~> **WARNING** This resource implements a part of a workflow. You must use it with one of either `signalfx_aws_external_integration` or `signalfx_aws_token_integration`.

## Example Usage

```terraform
// This resource returns an account id in `external_id`…
resource "signalfx_aws_external_integration" "aws_myteam_external" {
    name = "AWSFoo"
}

// Make yourself an AWS IAM role here, use `signalfx_aws_external_integration.aws_myteam_external.external_id`
resource "aws_iam_role" "aws_sfx_role" {
  // Stuff here that uses the external and account ID
}

resource "signalfx_aws_integration" "aws_myteam" {
    enabled = true

    integration_id = signalfx_aws_external_integration.aws_myteam_external.id
    external_id = signalfx_aws_external_integration.aws_myteam_external.external_id
		role_arn = aws_iam_role.aws_sfx_role.arn
		regions = ["us-east-1"]
		poll_rate = 300
		import_cloud_watch = true
		enable_aws_usage = true

		custom_namespace_sync_rule {
			default_action = "Exclude"
			filter_action = "Include"
			filter_source = "filter('code', '200')"
			namespace = "fart"
		}

		namespace_sync_rule {
			default_action = "Exclude"
			filter_action = "Include"
			filter_source = "filter('code', '200')"
			namespace = "AWS/EC2"
		}
}
```

## Service Names

Fields that expect an AWS service/namespace will work with one of: "AWS/ApiGateway" "AWS/AppStream" "AWS/AutoScaling" "AWS/Billing" "AWS/CloudFront" "AWS/CloudSearch" "AWS/Events" "AWS/Logs" "AWS/Connect" "AWS/DMS" "AWS/DX" "AWS/DynamoDB" "AWS/EC2" "AWS/EC2Spot" "AWS/ECS" "AWS/ElasticBeanstalk" "AWS/EBS" "AWS/EFS" "AWS/ELB" "AWS/ApplicationELB" "AWS/NetworkELB" "AWS/ElasticTranscoder" "AWS/ElastiCache" "AWS/ES" "AWS/ElasticMapReduce" "AWS/GameLift" "AWS/Inspector" "AWS/IoT" "AWS/KMS" "AWS/KinesisAnalytics" "AWS/Firehose" "AWS/Kinesis" "AWS/KinesisVideo" "AWS/Lambda" "AWS/Lex" "AWS/ML" "AWS/OpsWorks" "AWS/Polly" "AWS/Redshift" "AWS/RDS" "AWS/Route53" "AWS/SageMaker" "AWS/DDoSProtection" "AWS/SES" "AWS/SNS" "AWS/SQS" "AWS/S3" "AWS/SWF" "AWS/States" "AWS/StorageGateway" "AWS/Translate" "AWS/NATGateway" "AWS/VPN (VPN)" "WAF" "AWS/WorkSpaces".

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
* `key` - (Optional) If you specify `auth_method = \"SecurityToken\"` in your request to create an AWS integration object, use this property to specify the key.
* `regions` - (Optional) List of AWS regions that SignalFx should monitor.
* `role_arn` - (Optional) Role ARN that you add to an existing AWS integration object. **Note**: Ensure you use the `arn` property of your role, not the id!
* `services` - (Optional) List of AWS services that you want SignalFx to monitor. Each element is a string designating an AWS service. Conflicts with `namespace_sync_rule`.
* `poll_rate` - (Optional) AWS poll rate (in seconds). One of `60` or `300`.
* `use_get_metric_data_method` - (Optional) Enable the use of Amazon's `GetMetricData` for collecting metrics. Note that this requires the inclusion of the `"cloudwatch:GetMetricData"` permission.
