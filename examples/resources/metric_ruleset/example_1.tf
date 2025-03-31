resource "signalfx_metric_ruleset" "cpu_utilization_metric_ruleset" {
    metric_name = "cpu.utilization"
    description = "Routing ruleset for cpu.utilization"

    aggregation_rules {
        name = "cpu.utilization by service rule"
        description = "Aggregates cpu.utilization data by service"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "realm"
                property_value = [ "us-east-1" ]
                not = false
            }
        }
        aggregator {
            type = "rollup"
            dimensions = [ "service" ]
            drop_dimensions = false
            output_name = "cpu.utilization.by.service.agg"
        }
    }

    exception_rules {
        name = "Exception rule us-east-2"
        description = "Routes us-east-2 data to real-time"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "realm"
                property_value = [ "us-east-2" ]
                not = false
            }
        }
    }
    routing_rule {
        destination = "Archived"
    }
}
