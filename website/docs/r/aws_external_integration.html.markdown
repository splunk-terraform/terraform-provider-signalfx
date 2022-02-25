---
layout: "signalfx"
page_title: "SignalFx: signalfx_aws_external_integration"
sidebar_current: "docs-signalfx-resource-aws-external-integration"
description: |-
  Allows Terraform to create and manage SignalFx AWS External ID Integrations
---

# Resource: signalfx_aws_external_integration

SignalFx AWS CloudWatch integrations using Role ARNs. For help with this integration see [Connect to AWS CloudWatch](https://docs.signalfx.com/en/latest/integrations/amazon-web-services.html#connect-to-aws).

~> **NOTE** When managing integrations you need to use a session token for an administrator to authenticate the SignalFx provider. See [Operations that require a session token for an administrator].(https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator).

~> **WARNING** This resource implements a part of a workflow. You must use it with `signalfx_aws_integration`. Check with SignalFx support for your realm's AWS account id.

## Example Usage

```tf
resource "signalfx_aws_external_integration" "aws_myteam_extern" {
  name = "AWSFooNEW"
}

data "aws_iam_policy_document" "signalfx_assume_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = [signalfx_aws_external_integration.aws_myteam_extern.signalfx_aws_account]
    }

    condition {
      test     = "StringEquals"
      variable = "sts:ExternalId"
      values   = [signalfx_aws_external_integration.aws_myteam_extern.external_id]
    }
  }
}

resource "aws_iam_role" "aws_sfx_role" {
  name               = "signalfx-reads-from-cloudwatch2"
  description        = "signalfx integration to read out data and send it to signalfxs aws account"
  assume_role_policy = data.aws_iam_policy_document.signalfx_assume_policy.json
}

resource "aws_iam_policy" "aws_read_permissions" {
  name        = "SignalFxReadPermissionsPolicy"
  description = "farts"
  policy      = <<EOF
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": [
				"dynamodb:ListTables",
		    "dynamodb:DescribeTable",
		    "dynamodb:ListTagsOfResource",
		    "ec2:DescribeInstances",
		    "ec2:DescribeInstanceStatus",
		    "ec2:DescribeVolumes",
		    "ec2:DescribeReservedInstances",
		    "ec2:DescribeReservedInstancesModifications",
		    "ec2:DescribeTags",
		    "organizations:DescribeOrganization",
		    "cloudwatch:ListMetrics",
		    "cloudwatch:GetMetricData",
		    "cloudwatch:GetMetricStatistics",
		    "cloudwatch:DescribeAlarms",
		    "sqs:ListQueues",
		    "sqs:GetQueueAttributes",
		    "sqs:ListQueueTags",
		    "elasticmapreduce:ListClusters",
		    "elasticmapreduce:DescribeCluster",
		    "kinesis:ListShards",
		    "kinesis:ListStreams",
		    "kinesis:DescribeStream",
		    "kinesis:ListTagsForStream",
		    "rds:DescribeDBInstances",
		    "rds:ListTagsForResource",
		    "elasticloadbalancing:DescribeLoadBalancers",
		    "elasticloadbalancing:DescribeTags",
		    "elasticache:describeCacheClusters",
		    "redshift:DescribeClusters",
		    "lambda:GetAlias",
		    "lambda:ListFunctions",
		    "lambda:ListTags",
		    "autoscaling:DescribeAutoScalingGroups",
		    "s3:ListAllMyBuckets",
		    "s3:ListBucket",
		    "s3:GetBucketLocation",
		    "s3:GetBucketTagging",
		    "ecs:ListServices",
		    "ecs:ListTasks",
		    "ecs:DescribeTasks",
		    "ecs:DescribeServices",
		    "ecs:ListClusters",
		    "ecs:DescribeClusters",
		    "ecs:ListTaskDefinitions",
		    "ecs:ListTagsForResource",
		    "apigateway:GET",
		    "cloudfront:ListDistributions",
		    "cloudfront:ListTagsForResource",
		    "tag:GetResources",
		    "es:ListDomainNames",
		    "es:DescribeElasticsearchDomain"
			],
			"Effect": "Allow",
			"Resource": "*"
		}
	]
}
EOF
}

resource "aws_iam_role_policy_attachment" "sfx-read-attach" {
  role       = aws_iam_role.aws_sfx_role.name
  policy_arn = aws_iam_policy.aws_read_permissions.arn
}


resource "signalfx_aws_integration" "aws_myteam" {
  enabled = true

  integration_id = signalfx_aws_external_integration.aws_myteam_extern.id
  external_id    = signalfx_aws_external_integration.aws_myteam_extern.external_id
  role_arn       = aws_iam_role.aws_sfx_role.arn
  # token = "abc123"
  # key = "abc123"
  regions            = ["us-east-1"]
  poll_rate          = 300
  import_cloud_watch = true
  enable_aws_usage   = true
}

```

## Argument Reference

* `name` - (Required) The name of this integration

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of this integration, used with `signalfx_aws_integration`
* `external_id` - The external ID to use with your IAM role and with `signalfx_aws_integration`.
* `signalfx_aws_account` - The AWS Account ARN to use with your policies/roles, provided by SignalFx.
