resource "signalfx_integration_splunk_oncall" "test" {
  name        = "Test Integration"
  enabled     = true
  post_url    = "https://example.com/splunk_oncall"
}
