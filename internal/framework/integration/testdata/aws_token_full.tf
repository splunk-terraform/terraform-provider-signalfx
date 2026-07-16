resource "signalfx_aws_integration" "test" {
  integration_id = "aws-token-id"
  enabled        = true
  token          = "security-token"
  key            = "access-key"
  regions        = ["ap-southeast-2"]
}
