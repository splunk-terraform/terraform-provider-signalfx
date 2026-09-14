resource "signalfx_observability_directory" "test" {
  path = "teams/platform/dashboards"
  templates = [
    "/v2/template/dashboard-a",
    "/v2/template/dashboard-b",
  ]
  pinned = true
}
