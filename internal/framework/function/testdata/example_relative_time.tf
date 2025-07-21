provider "signalfx" {}

output "milliseconds" {
  value = provider::signalfx::parse_relative_time("-1h")
}
