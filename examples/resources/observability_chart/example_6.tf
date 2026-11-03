resource "signalfx_observability_chart" "runbook" {
  title = "Service runbook"

  text = {
    title    = "Troubleshooting"
    markdown = "## Request rate\nCompare the request charts before investigating individual hosts."
  }
}
