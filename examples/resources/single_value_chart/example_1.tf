resource "signalfx_single_value_chart" "mysvchart0" {
  name = "CPU Total Idle - Single Value"

  program_text = <<-EOF
        myfilters = filter("cluster_name", "prod") and filter("role", "search")
        data("cpu.total.idle", filter=myfilters).publish()
        EOF

  description = "Very cool Single Value Chart"

  color_by = "Dimension"

  max_delay           = 2
  refresh_interval    = 1
  max_precision       = 2
  is_timestamp_hidden = true
}
