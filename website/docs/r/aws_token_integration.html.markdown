---
layout: "signalfx"
page_title: "SignalFx: signalfx_aws_token_integration"
sidebar_current: "docs-signalfx-resource-aws-token-integration"
description: |-
  Allows Terraform to create and manage SignalFx AWS Security Token Integrations
---

# Resource: signalfx_aws_external_integration

SignalFx AWS CloudWatch integrations using security tokens. For help with this integration see [Connect to AWS CloudWatch](https://docs.signalfx.com/en/latest/integrations/amazon-web-services.html#connect-to-aws).

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the SignalFx provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator).

~> **WARNING** This resource implements a part of a workflow. You must use it with `signalfx_aws_integration`.

## Example Usage

```tf
resource "signalfx_aws_token_integration" "aws_myteam_token" {
  name = "My AWS integration"
}

// Make yourself an AWS IAM role here
resource "aws_iam_role" "aws_sfx_role" {
  // Stuff here that uses the external and account ID
}

resource "signalfx_aws_integration" "aws_myteam" {
  enabled = true

  integration_id     = signalfx_aws_token_integration.aws_myteam_token.id
  token              = "put_your_token_here"
  key                = "put_your_key_here"
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
}
```

## Argument Reference

* `name` - (Required) The name of this integration

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the integration to use with `signalfx_aws_integration`
* `signalfx_aws_account` - The AWS Account ARN to use with your policies/roles, provided by SignalFx.
