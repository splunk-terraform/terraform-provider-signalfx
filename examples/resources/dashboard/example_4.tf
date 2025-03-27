resource "signalfx_dashboard" "grid_example" {
  name            = "Grid"
  dashboard_group = signalfx_dashboard_group.example.id
  time_range      = "-15m"

  grid {
    chart_ids = [
      concat(
        signalfx_time_chart.rps.*.id,
        signalfx_time_chart.p50ths.*.id,
        signalfx_time_chart.p99ths.*.id,
        signalfx_time_chart.idle_workers.*.id,
        signalfx_time_chart.cpu_idle.*.id,
      )
    ]
    width  = 3
    height = 1
  }
}
