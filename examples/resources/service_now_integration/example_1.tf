resource "signalfx_service_now_integration" "service_now_myteam" {
  name    = "ServiceNow - My Team"
  enabled = false

  username = "thisis_me"
  password = "youd0ntsee1t"
  
  instance_name = "myinst.service-now.com"
  issue_type    = "Incident"

  alert_triggered_payload_template = "{\"short_description\": \"{{{messageTitle}}} (customized)\"}"
  alert_resolved_payload_template  = "{\"close_notes\": \"{{{messageTitle}}} (customized close msg)\"}"
}
