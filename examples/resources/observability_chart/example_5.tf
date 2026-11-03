resource "signalfx_observability_chart" "request_clusters" {
  title = "Request clusters"

  metrics_cluster_map = {
    program     = "A = data('requests.count').sum(by=['host']).publish('A')"
    group_by    = ["host"]
    color_by    = "Range"
    color_range = { palette = "temperature" }
  }
}
