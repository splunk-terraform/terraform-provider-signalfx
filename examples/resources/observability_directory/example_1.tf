terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_dashboard" "service" {
  title = "Service overview"
}

resource "signalfx_observability_directory" "dashboards" {
  path   = "teams/platform/dashboards"
  pinned = true
  templates = [
    "/v2/template/${signalfx_observability_dashboard.service.id}",
  ]
}
