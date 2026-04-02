# An example configuration with a service account.
provider "signalfx" {
  email           = "service.account@example"
  password        = "${var.service_account_password}"
  organization_id = "${var.service_account_org_id}"
  # If your organization uses a different realm
  # api_url = "https://api.<realm>.observability.splunk.com"
  # If your organization uses a custom URL
  # custom_app_url = "https://myorg.observability.splunk.com"
}
