data "signalfx_apm_service_topology" "production" {
  start_time = "-1w"
  filters = [
    { name = "environment", scope = "GLOBAL", exactly = "production", matches = ["staging"] }
  ]
}
