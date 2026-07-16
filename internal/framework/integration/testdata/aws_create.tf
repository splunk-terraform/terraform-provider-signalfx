resource "signalfx_aws_integration" "test" {
  integration_id = "aws-id"
  enabled        = true
  external_id    = "external-id"
  role_arn       = "arn:aws:iam::111111111111:role/signalfx"
  regions        = ["us-east-1", "us-west-2"]
  services       = ["ec2", "s3"]

  custom_namespace_sync_rule {
    namespace      = "Example/Custom"
    default_action = "Exclude"
    filter_action  = "Include"
    filter_source  = "filter('aws_tag_environment', 'production')"
  }

  metric_stats_to_sync {
    namespace = "AWS/EC2"
    metric    = "CPUUtilization"
    stats     = ["Average", "p99"]
  }
}
