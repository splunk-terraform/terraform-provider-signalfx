resource "signalfx_webhook_integration" "webhook_myteam" {
  name             = "Webhook - My Team"
  enabled          = true
  url              = "https://www.example.com"
  shared_secret    = "abc1234"
  method           = "POST"
  payload_template = <<-EOF
    {
      "incidentId": "{{{incidentId}}}"
    }
  EOF

  headers {
    header_key   = "some_header"
    header_value = "value_for_that_header"
  }
}
