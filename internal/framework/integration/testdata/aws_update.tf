resource "signalfx_aws_integration" "test" {
  integration_id                    = "aws-id"
  enabled                           = true
  external_id                       = "external-id"
  role_arn                          = "arn:aws:iam::111111111111:role/signalfx"
  regions                           = ["eu-west-1"]
  poll_rate                         = 600
  inactive_metrics_poll_rate        = 3600
  enable_aws_usage                  = true
  import_cloud_watch                = true
  use_metric_streams_sync           = true
  named_token                       = "aws-ingest"
  enable_check_large_volume         = true
  sync_custom_namespaces_only       = true
  collect_only_recommended_stats    = true
  metric_streams_managed_externally = true

  custom_cloudwatch_namespaces = ["Example/Legacy"]

  namespace_sync_rule {
    namespace = "lambda"
  }
}
