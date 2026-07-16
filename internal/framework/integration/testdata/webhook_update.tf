resource "signalfx_webhook_integration" "test" {
  name             = "Updated Webhook"
  enabled          = false
  url              = "https://webhook.test/updated"
  shared_secret    = "updated-secret"
  method           = "PUT"
  payload_template = "{\"updated\":true}"

  headers {
    header_key   = "x-updated"
    header_value = "updated-value"
  }
}
