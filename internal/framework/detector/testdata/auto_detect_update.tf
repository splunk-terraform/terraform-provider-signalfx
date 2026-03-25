resource "signalfx_customized_auto_detector" "example" {
  parent_id   = "parent-detector"
  name        = "Modified Example Detector Updated"
  description = "This is an updated customized auto detector resource."
  severity    = "Major"
  tags        = []
  teams       = []

  notifications = [
    { type = "Email", email = "updated@example.com" },
  ]

  inputs = {
    guardrail = "0.95"
  }

  filters = [
    { key = "service", values = ["api"] },
  ]
}
