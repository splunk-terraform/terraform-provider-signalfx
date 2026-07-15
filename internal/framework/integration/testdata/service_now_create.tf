resource "signalfx_service_now_integration" "test" {
  name          = "Primary ServiceNow"
  enabled       = true
  username      = "primary-user"
  password      = "primary-password"
  instance_name = "primary.service-now.com"
  issue_type    = "Incident"

  alert_triggered_payload_template = "{\"short_description\":\"primary\"}"
  alert_resolved_payload_template  = "{\"close_notes\":\"primary\"}"
}
