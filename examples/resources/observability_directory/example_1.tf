terraform {
  required_providers {
    signalfx = {
      source = "splunk-terraform/signalfx"
    }
  }
}

resource "signalfx_observability_directory" "dashboards" {
  path   = "teams/platform/dashboards"
  pinned = true
}
