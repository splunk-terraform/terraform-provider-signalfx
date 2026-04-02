# Configure the Splunk Observability Cloud provider
provider "signalfx" {
  auth_token = "${var.signalfx_auth_token}"
  # If your organization uses a different realm
  # api_url = "https://api.<realm>.observability.splunk.com"
}

# Create a new detector
resource "signalfx_detector" "default" {
  # ...
}

# Create a new dashboard
resource "signalfx_dashboard" "default" {
  # ...
}
