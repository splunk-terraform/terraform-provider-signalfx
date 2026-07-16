resource "signalfx_service_now_integration" "test" {
  name          = "Invalid ServiceNow"
  enabled       = true
  username      = "user"
  password      = "password"
  instance_name = "invalid.service-now.com"
  issue_type    = "Change"
}
