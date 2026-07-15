resource "signalfx_service_now_integration" "test" {
  name          = "Updated ServiceNow"
  enabled       = false
  username      = "updated-user"
  password      = "updated-password"
  instance_name = "updated.service-now.com"
  issue_type    = "Problem"

  alert_triggered_payload_template = "{\"short_description\":\"updated\"}"
  alert_resolved_payload_template  = "{\"close_notes\":\"updated\"}"
}
