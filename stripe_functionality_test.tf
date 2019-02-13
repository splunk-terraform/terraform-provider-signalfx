resource "signalform_text_chart" "sample_text_chart" {
  name = " "
  markdown = <<EOF
<table width="100%"  rules="none"><tr><td valign="middle" align="center" bgcolor="#6e9cc1">
<font size="7" color="white">SAMPLE</font><br>
<font size="5" color="white">hi this is dog</font>
</td></tr></table>
<br>
[sample](http://example.com) dashboard
EOF
}
resource "signalform_time_chart" "sample_time_chart" {
  name        = "Sample"
  description = "Sample"
  plot_type   = "LineChart"
  max_delay   = 20
  color_by    = "Metric"
  program_text = <<EOF
A = data('sample.sample_time_ms.99percentile', rollup=None, filter=(filter('host_env', '*') and filter('host_type', '*') and filter('service', '*'))).mean(by=['service']).publish(label='sample.sample_time_ms.99percentile')
B = data('sample.sample_time_ms.95percentile', rollup=None, filter=(filter('host_env', '*') and filter('host_type', '*') and filter('service', '*'))).mean(by=['service']).publish(label='sample.sample_time_ms.95percentile')
C = data('sample.sample_time_ms.90percentile', rollup=None, filter=(filter('host_env', '*') and filter('host_type', '*') and filter('service', '*'))).mean(by=['service']).publish(label='sample.sample_time_ms.90percentile')
D = data('sample.sample_time_ms.50percentile', rollup=None, filter=(filter('host_env', '*') and filter('host_type', '*') and filter('service', '*'))).mean(by=['service']).publish(label='sample.sample_time_ms.50percentile')
EOF
  disable_sampling = true
  viz_options {
    label      = "sample.sample_time_ms.99percentile"
    axis       = "left"
    value_unit = "Millisecond"
  }
  viz_options {
    label      = "sample.sample_time_ms.95percentile"
    axis       = "left"
    value_unit = "Millisecond"
  }
  viz_options {
    label      = "sample.sample_time_ms.90percentile"
    axis       = "left"
    value_unit = "Millisecond"
  }
  viz_options {
    label      = "sample.sample_time_ms.50percentile"
    axis       = "left"
    value_unit = "Millisecond"
  }
}
resource "signalform_list_chart" "sample_list_chart" {
  name      = "Sample"
  max_delay = 20
  color_by  = "Dimension"
  program_text = <<EOF
A = data('sample.sample_time_ms.count', rollup='sum', filter=(filter('host_env', '*') and filter('host_type', '*') and filter('service', '*'))).sum(by=['service']).publish(label='sample.sample_time_ms.count')
EOF
  disable_sampling = true
  sort_by = "-value"
}
resource "signalform_single_value_chart" "sample_value_chart" {
  name      = "Sample"
  max_delay = 20
  color_by  = "Dimension"
  program_text = <<EOF
A = data('sample.sample_time_ms.count', rollup='sum', filter=(filter('host_env', '*') and filter('host_type', '*') and filter('service', '*'))).sum().sum(over='1h').publish(label='sample.sample_time_ms.count')
EOF
  disable_sampling = true
  viz_options {
    label        = "sample.sample_time_ms.count"
    value_suffix = "sample"
  }
}
resource "signalform_dashboard_group" "sample_group" {
  name        = "Sample"
  description = "Sample"
  teams       = ["AAAAAAAAAAA"]
}
resource "signalform_dashboard" "sample_dashboard_grid" {
  name            = "Sample"
  dashboard_group = "${signalform_dashboard_group.sample_group.id}"
  time_range = "-1h"
  variable {
    property         = "host_env"
    alias            = "env"
    values           = ["prod"]
    values_suggested = ["prod", "qa"]
    replace_only     = true
    apply_if_exist   = true
  }
  variable {
    property       = "host_type"
    alias          = "host_type"
    values         = ["*"]
    replace_only   = true
    apply_if_exist = true
  }
  variable {
    property       = "service"
    alias          = "service"
    values         = ["*"]
    replace_only   = true
    apply_if_exist = true
  }
  grid {
    chart_ids = ["${signalform_text_chart.sample_text_chart.id}"]
    start_row = 0
    width     = 12
    height    = 2
  }
  grid {
    chart_ids = [
      "${signalform_time_chart.sample_time_chart.id}",
      "${signalform_list_chart.sample_list_chart.id}",
      "${signalform_single_value_chart.sample_value_chart.id}",
    ]
    start_row = 2
    width     = 4
  }
}
resource "signalform_detector" "sample_detector" {
  name        = "SAMPLE: sample"
  description = "SAMPLE: sample"
  max_delay   = 20
  teams       = ["AAAAAAAAAAA"]
  program_text = <<-EOF
sample = data('sample.sample_time_ms.99percentile', rollup=None, filter=(filter('host_env', 'prod') and filter('host_type', 'foobox') and filter('service', 'foo-srv'))).mean(by=['service'])
detect(on=when(sample > 10000, lasting=None), off=None).publish("sample")
  EOF
  rule {
    detect_label          = "sample"
    severity              = "Critical"
    parameterized_subject = "SAMPLE: sample"
    parameterized_body = <<-EOF
[sample](http://example.com) for {{dimensions.service}}
    EOF
    notifications = ["Slack,AAAAAAAAAAA,foo-team", "PagerDuty,${signalform_integration.pagerduty_foo.id}"]
  }
}
resource "signalform_integration" "pagerduty_foo" {
  name    = "pagerduty foo"
  enabled = true
  type    = "PagerDuty"
  api_key = "long_hexadecimal_here"
}
