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
