resource "signalfx_integration_splunk_oncall" "test" {
  name        = "Test Integration"
  enabled     = false
  post_url    = "https://example.com/post"
}
