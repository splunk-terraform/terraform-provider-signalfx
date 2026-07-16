resource "signalfx_metric_ruleset" "test" {
  metric_name = "demo.metric"
  description = "updated ruleset"

  aggregation_rules {
    name        = "aggregate-updated"
    description = "aggregate by service and host"
    enabled     = true

    matcher {
      type = "dimension"

      filters {
        property       = "realm"
        property_value = ["us0", "us1"]
        not            = false
      }
    }

    aggregator {
      type            = "rollup"
      output_name     = "demo.metric.by.service-host"
      dimensions      = ["service", "host"]
      drop_dimensions = false
    }
  }

  exception_rules {
    name        = "exception"
    description = "retain non-free data"
    enabled     = true

    matcher {
      type = "dimension"

      filters {
        property       = "tier"
        property_value = ["free"]
        not            = true
      }
    }

    restoration {
      start_time = 100
      stop_time  = 200
    }
  }

  routing_rule {
    destination = "Drop"
  }
}
