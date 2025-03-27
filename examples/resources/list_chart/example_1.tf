resource "signalfx_list_chart" "mylistchart0" {
  name = "CPU Total Idle - List"

  program_text = <<-EOF
    myfilters = filter("cluster_name", "prod") and filter("role", "search")
    data("cpu.total.idle", filter=myfilters).publish()
    EOF

  description = "Very cool List Chart"

  color_by         = "Metric"
  max_delay        = 2
  timezone         = "Europe/Paris"
  disable_sampling = true
  refresh_interval = 1
  hide_missing_values = true

  legend_options_fields {
    property = "collector"
    enabled  = false
  }

  legend_options_fields {
    property = "cluster_name"
    enabled  = true
  }
  legend_options_fields {
    property = "role"
    enabled  = true
  }
  legend_options_fields {
    property = "collector"
    enabled  = false
  }
  legend_options_fields {
    property = "host"
    enabled  = false
  }
  max_precision = 2
  sort_by       = "-value"
}
