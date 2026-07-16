resource "signalfx_metric_ruleset" "test" {
  metric_name = "demo.metric"
  description = "initial ruleset"

  aggregation_rules {
    name        = "aggregate"
    description = "aggregate by service"
    enabled     = true

    matcher {
      type = "dimension"

      filters {
        property       = "realm"
        property_value = ["us0"]
        not            = false
      }
    }

    aggregator {
      type            = "rollup"
      output_name     = "demo.metric.by.service"
      dimensions      = ["service"]
      drop_dimensions = false
    }
  }

  exception_rules {
    name        = "exception"
    description = "retain premium data"
    enabled     = true

    matcher {
      type = "dimension"

      filters {
        property       = "tier"
        property_value = ["premium"]
        not            = false
      }
    }

    restoration {
      start_time = 100
    }
  }

  routing_rule {
    destination = "Archived"
  }
}
