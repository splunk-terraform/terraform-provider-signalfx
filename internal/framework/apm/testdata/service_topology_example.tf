data "signalfx_apm_service_topology" "production" {
  start_time = "-1w"
  filters = [
    { name = "environment", scope = "GLOBAL", exactly = "production" },
    { name = "capability", scope = "GLOBAL", matches = ["function-1", "function-2"] }
  ]
}

output "nodes" {
  value = data.signalfx_apm_service_topology.production.nodes
}
output "edges" {
  value = data.signalfx_apm_service_topology.production.edges
}
