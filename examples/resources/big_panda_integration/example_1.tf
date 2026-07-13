resource "signalfx_big_panda_integration" "big_panda_myteam" {
  name    = "BigPanda - My Team"
  enabled = true

  app_key = "my-app-key"
  token   = "my-token"

  # Optional. If omitted, Observability Cloud sends the default BigPanda payload.
  alert_triggered_payload_template = "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}"
  alert_resolved_payload_template  = "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}"
}
