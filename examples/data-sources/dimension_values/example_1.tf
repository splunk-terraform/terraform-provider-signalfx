resource "signalfx_dashboard_group" "mydashboardgroup0" {
  name        = "My team dashboard group"
  description = "Cool dashboard group"
}

data "signalfx_dimension_values" "hosts" {
  query = "key:host"
}

resource "signalfx_time_chart" "host_charts" {
  for_each = toset(data.signalfx_dimension_values.hosts.values)

  name = "CPU Total Idle ${each.value}"

  plot_type         = "ColumnChart"
  axes_include_zero = true
  color_by          = "Metric"

  program_text = <<-EOF
A = data("cpu.idle", filter('host', '${each.key}').publish(label="CPU")
        EOF
}

resource "signalfx_dashboard" "mydashboard1" {
  name            = "My Dashboard"
  dashboard_group = signalfx_dashboard_group.mydashboardgroup0.id

  time_range = "-30m"

  grid {
    chart_ids = toset([for v in signalfx_time_chart.host_charts : v.id])
    width     = 3
    height    = 1
  }
}
