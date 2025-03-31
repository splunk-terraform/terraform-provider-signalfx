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
