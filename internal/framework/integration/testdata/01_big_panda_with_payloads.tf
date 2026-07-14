resource "signalfx_big_panda_integration" "test" {
  name    = "BigPanda - My Team"
  enabled = false

  app_key = "my-app-key"
  token   = "my-token"

  alert_triggered_payload_template = "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}"
  alert_resolved_payload_template  = "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}"
}
