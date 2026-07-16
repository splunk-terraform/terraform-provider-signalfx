resource "signalfx_aws_integration" "test" {
  integration_id = "aws-id"
  enabled        = true
  external_id    = "external-id"
  token          = "security-token"
  regions        = []
}
