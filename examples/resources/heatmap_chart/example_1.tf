resource "signalfx_heatmap_chart" "myheatmapchart0" {
  name = "CPU Total Idle - Heatmap"

  program_text = <<-EOF
        myfilters = filter("cluster_name", "prod") and filter("role", "search")
        data("cpu.total.idle", filter=myfilters).publish()
        EOF

  description = "Very cool Heatmap"

  disable_sampling = true
  sort_by          = "+host"
  group_by         = ["hostname", "host"]
  hide_timestamp   = true
  timezone         = "Europe/Paris"

  color_range {
    min_value = 0
    max_value = 100
    color     = "#ff0000"
  }

  # You can only use one of color_range or color_scale!
  color_scale {
    gte   = 99
    color = "green"
  }
  color_scale {
    lt    = 99 # This ensures terraform recognizes that we cover the range 95-99
    gte   = 95
    color = "yellow"
  }
  color_scale {
    lt    = 95
    color = "red"
  }
}
