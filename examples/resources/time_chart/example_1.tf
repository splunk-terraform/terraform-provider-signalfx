resource "signalfx_time_chart" "mychart0" {
  name = "CPU Total Idle"

  program_text = <<-EOF
        data("cpu.total.idle").publish(label="CPU Idle")
        EOF

  time_range = 3600

  plot_type         = "LineChart"
  show_data_markers = true

  legend_options_fields {
    property = "collector"
    enabled  = false
  }
  legend_options_fields {
    property = "hostname"
    enabled  = false
  }

  viz_options {
    label = "CPU Idle"
    axis  = "left"
    color = "orange"
  }

  axis_left {
    label         = "CPU Total Idle"
    low_watermark = 1000
  }
}
