data "signalfx_auto_detector" "all" {}

# Only needed to show within terraform what auto detectors are available to extend
# and can be used as reference for the parent_id field in the customized auto detector resource.
output "auto_detector_results" {
  value = data.signalfx_auto_detector.all.results
}

# This is an example of a customized auto detector resource that extends the EC2 auto detector.
# The parent_id field references the id of the EC2 auto detector from the data source above.
resource "signalfx_customized_auto_detector" "extended_ec2" {
  parent_id   = data.signalfx_auto_detector.all.results.EC2.id
  
  name        = "Modified EC2 Example Detector"
  description = "This is an example of a modified auto detector resource."
  severity    = "Critical"
  tags        = []
  teams       = []

  notifications = [
    { type = "Email", email = "example@example.com" },
    { type = "Slack", channel = "#alerts", credential_id = "slack-credential" },
  ]

  # Inputs are key value pairs that are discoverable by the data 
  # and will change based on the parent's provided configurable inputs.
  inputs = {
    guardrail = "0.81"
  }

  # Each individual filter is combined as an AND condition within the auto detector configuration.
  filters = [
    { key = "service", values = ["web"] },
    { key = "environment", values = ["production"] },
  ]
}
