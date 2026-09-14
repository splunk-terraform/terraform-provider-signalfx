resource "signalfx_observability_directory" "test" {
  path = "teams/platform/dashboards"
  templates = [
    "/v2/template/dashboard-b",
    "/v2/template/dashboard-c",
  ]
  pinned = false
}
