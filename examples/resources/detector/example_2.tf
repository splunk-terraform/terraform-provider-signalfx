resource "signalfx_detector" "enhanced_multi_condition" {
  name        = "Enhanced multi-condition detector"
  description = "Historical anomaly and threshold conditions with custom logic."
  max_delay   = 30
  tags        = ["detectors", "historical-anomaly"]

  program_text = <<-EOF
    from signalfx.detectors.against_periods import conditions

    latency = data('service.latency').mean(by=['service']).publish('service latency')
    error_rate = data('service.error_rate').mean(by=['service']).publish('service error rate')
    saturation = data('service.saturation').mean(by=['service']).publish('service saturation')

    latency_anomaly_fire, latency_anomaly_clear = conditions.mean_std(
      latency,
      window_to_compare=duration('15m'),
      space_between_windows=duration('1w'),
      fire_num_stddev=3,
      clear_num_stddev=2.5,
      orientation='above',
    )

    sustained_errors = when(error_rate > 5, '5m')
    high_saturation = when(saturation > 80, '10m')
    critical_saturation = when(saturation > 95, '5m')

    detect(
      (latency_anomaly_fire and sustained_errors and high_saturation) or critical_saturation,
      latency_anomaly_clear and when(error_rate < 2, '10m') and when(saturation < 70, '10m'),
    ).publish('Historical anomaly and service health')
  EOF

  rule {
    description   = "Historical latency anomaly with elevated error rate and saturation, or critical saturation"
    severity      = "Critical"
    detect_label  = "Historical anomaly and service health"
    notifications = ["Email,foo-alerts@example.com"]
  }
}

provider "signalfx" {}
