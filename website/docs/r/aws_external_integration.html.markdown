---
layout: "signalfx"
page_title: "SignalFx: signalfx_aws_external_integration"
sidebar_current: "docs-signalfx-resource-aws-external-integration"
description: |-
  Allows Terraform to create and manage SignalFx AWS External ID Integrations
---

# Resource: signalfx_aws_external_integration

SignalFx AWS CloudWatch integrations using Role ARNs. For help with this integration see [Monitoring Amazon Web Services](https://docs.signalfx.com/en/latest/integrations/amazon-web-services.html#monitor-amazon-web-services).

**Note:** When managing integrations you'll need to use an admin token to authenticate the SignalFx provider.

~> **WARNING** This resource implements a part of a workflow. You must use it with one of either `signalfx_aws_integration`.

## Example Usage

```terraform
// This resource returns an account id in `external_id`…
resource "signalfx_aws_external_integration" "aws_myteam_external" {
    name = "AWSFoo"
}

// Make yourself an AWS IAM role here, use `signalfx_aws_external_integration.aws_myteam_external.external_id`
resource "aws_iam_role" "aws_sfx_role" {
  // Stuff here that uses the
}

resource "signalfx_aws_integration" "aws_myteam" {
    enabled = true

    integration_id = "${signalfx_aws_external_integration.aws_myteam_external.id}"
    external_id = "${signalfx_aws_external_integration.aws_myteam_external.external_id}"
		role_arn = "${aws_iam_role.aws_sfx_role.id}"
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

## Argument Reference

* `name` - (Required) The name of this integration

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The time at which the certificate was issued
* `external_id` - The AWS account ID to use with your IAM role and with `signalfx_aws_integration`.
