resource "signalfx_observability_chart" "requests_by_host" {
  title = "Requests by host"

  metrics_list = {
    program        = "A = data('requests.count').sum(by=['host']).publish('A')"
    display_fields = [{ property = "host", enabled = true }]
    sort           = { by = "value", direction = "desc" }
  }
}
