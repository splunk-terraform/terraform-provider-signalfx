---
layout: "signalfx"
page_title: "Splunk Observability Cloud: signalfx_aws_external_integration"
sidebar_current: "docs-signalfx-resource-aws-external-integration"
description: |-
  Allows Terraform to create and manage Splunk Observability AWS External ID Integrations
---

# Resource: signalfx_aws_external_integration

Splunk Observability AWS CloudWatch integrations using Role ARNs. For help with this integration see [Connect to AWS CloudWatch](https://docs.splunk.com/observability/en/gdi/get-data-in/connect/aws/aws-apiconfig.html).

~> **NOTE** When managing integrations, use a session token of an administrator to authenticate the Splunk Observability provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator).

~> **WARNING** This resource implements part of a workflow. Use it with `signalfx_aws_integration`. Check with Splunk support for your realm's AWS account id.

~> **NOTE** When AWS IAM role is also created by Terraform to enforce a proper cleanup sequence add role related resources as dependencies for the `signalfx_aws_integration` resource. See the `depends_on` section in the example script below.

## Example

```tf
resource "signalfx_aws_external_integration" "aws_myteam_extern" {
  name = "My AWS integration"
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

resource "aws_iam_role" "aws_splunk_role" {
  name               = "signalfx-reads-from-cloudwatch2"
  description        = "Splunk Observability Cloud integration to read out data and send it to signalfxs aws account"
  assume_role_policy = data.aws_iam_policy_document.signalfx_assume_policy.json
}

resource "aws_iam_policy" "aws_splunk_policy" {
  name        = "SplunkObservabilityPolicy"
  description = "AWS permissions required by the Splunk Observability Cloud"
  policy      = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "airflow:GetEnvironment",
        "airflow:ListEnvironments",
        "apigateway:GET",
        "autoscaling:DescribeAutoScalingGroups",
        "cloudformation:ListResources",
        "cloudformation:GetResource",
        "cloudfront:GetDistributionConfig",
        "cloudfront:ListDistributions",
        "cloudfront:ListTagsForResource",
        "cloudwatch:GetMetricData",
        "cloudwatch:ListMetrics",
        "directconnect:DescribeConnections",
        "dynamodb:DescribeTable",
        "dynamodb:ListTables",
        "dynamodb:ListTagsOfResource",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceStatus",
        "ec2:DescribeNatGateways",
        "ec2:DescribeRegions",
        "ec2:DescribeReservedInstances",
        "ec2:DescribeReservedInstancesModifications",
        "ec2:DescribeTags",
        "ec2:DescribeVolumes",
        "ecs:DescribeClusters",
        "ecs:DescribeServices",
        "ecs:DescribeTasks",
        "ecs:ListClusters",
        "ecs:ListServices",
        "ecs:ListTagsForResource",
        "ecs:ListTaskDefinitions",
        "ecs:ListTasks",
        "eks:DescribeCluster",
        "eks:ListClusters",
        "elasticache:DescribeCacheClusters",
        "elasticloadbalancing:DescribeLoadBalancerAttributes",
        "elasticloadbalancing:DescribeLoadBalancers",
        "elasticloadbalancing:DescribeTags",
        "elasticloadbalancing:DescribeTargetGroups",
        "elasticmapreduce:DescribeCluster",
        "elasticmapreduce:ListClusters",
        "es:DescribeElasticsearchDomain",
        "es:ListDomainNames",
        "iam:listAccountAliases",
        "kafka:DescribeCluster",
        "kafka:DescribeClusterV2",
        "kafka:ListClusters",
        "kafka:ListClustersV2",
        "kinesis:DescribeStream",
        "kinesis:ListShards",
        "kinesis:ListStreams",
        "kinesis:ListTagsForStream",
        "kinesisanalytics:DescribeApplication",
        "kinesisanalytics:ListApplications",
        "kinesisanalytics:ListTagsForResource",
        "lambda:GetAlias",
        "lambda:ListFunctions",
        "lambda:ListTags",
        "logs:DeleteSubscriptionFilter",
        "logs:DescribeLogGroups",
        "logs:DescribeSubscriptionFilters",
        "logs:PutSubscriptionFilter",
        "network-firewall:DescribeFirewall",
        "network-firewall:ListFirewalls",
        "organizations:DescribeOrganization",
        "rds:DescribeDBInstances",
        "rds:DescribeDBClusters",
        "rds:ListTagsForResource",
        "redshift:DescribeClusters",
        "redshift:DescribeLoggingStatus",
        "s3:GetBucketLocation",
        "s3:GetBucketLogging",
        "s3:GetBucketNotification",
        "s3:GetBucketTagging",
        "s3:ListAllMyBuckets",
        "s3:ListBucket",
        "s3:PutBucketNotification",
        "sqs:GetQueueAttributes",
        "sqs:ListQueues",
        "sqs:ListQueueTags",
        "states:ListActivities",
        "states:ListStateMachines",
        "tag:GetResources",
        "workspaces:DescribeWorkspaces"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "cassandra:Select"
      ],
      "Resource": [
        "arn:aws:cassandra:*:*:/keyspace/system/table/local",
        "arn:aws:cassandra:*:*:/keyspace/system/table/peers",
        "arn:aws:cassandra:*:*:/keyspace/system_schema/*",
        "arn:aws:cassandra:*:*:/keyspace/system_schema_mcs/table/tags",
        "arn:aws:cassandra:*:*:/keyspace/system_schema_mcs/table/tables",
        "arn:aws:cassandra:*:*:/keyspace/system_schema_mcs/table/columns"
      ]
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "splunk_role_policy_attach" {
  role       = aws_iam_role.aws_splunk_role.name
  policy_arn = aws_iam_policy.aws_splunk_policy.arn
}

resource "signalfx_aws_integration" "aws_myteam" {
  enabled = true

  integration_id = signalfx_aws_external_integration.aws_myteam_extern.id
  external_id    = signalfx_aws_external_integration.aws_myteam_extern.external_id
  role_arn       = aws_iam_role.aws_splunk_role.arn
  # token = "abc123"
  # key = "abc123"
  regions            = ["us-east-1"]
  poll_rate          = 300
  import_cloud_watch = true
  enable_aws_usage   = true
  depends_on = [
    aws_iam_role.aws_splunk_role,
    aws_iam_role_policy_attachment.splunk_role_policy_attach
  ]
}
```

## Arguments

* `name` - (Required) The name of this integration

## Attributes

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of this integration, used with `signalfx_aws_integration`
* `external_id` - The external ID to use with your IAM role and with `signalfx_aws_integration`.
* `signalfx_aws_account` - The AWS Account ARN to use with your policies/roles, provided by Splunk Observability Cloud.
