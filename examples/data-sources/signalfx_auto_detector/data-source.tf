# Fetches all of the built-in auto detectors available to the user.
# This can be used to reference auto detector IDs when composing other resources.
data "signalfx_auto_detector" "example" {}

# This shows all built-in auto detectors available to the user.
output "all-auto-detectors" {
  value = data.signalfx_auto_detector.example.results
}

# A simple example referencing a specific auto detector by its Terraform-compatible name.
output "cpu_utilization_auto_detector_id" {
  value = data.signalfx_auto_detector.example.results.CPU_Utilization
}
