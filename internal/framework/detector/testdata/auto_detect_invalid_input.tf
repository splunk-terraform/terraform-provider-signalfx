resource "signalfx_customized_auto_detector" "example" {
  parent_id = "parent-detector"

  inputs = {
    unsupported = "value"
  }
}
