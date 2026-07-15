resource "signalfx_webhook_integration" "test" {
  name             = "Primary Webhook"
  enabled          = true
  url              = "https://webhook.test/primary"
  shared_secret    = "primary-secret"
  method           = "POST"
  payload_template = "{\"primary\":true}"

  headers {
    header_key   = "x-primary"
    header_value = "primary-value"
  }
}
