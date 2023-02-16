---
layout: "signalfx"
page_title: "SignalFx: signalfx_metric_ruleset"
sidebar_current: "docs-signalfx-resource-metric-ruleset"
description: |-
Allows Terraform to create and manage Splunk Infrastructure Monitoring metric rulesets
---

# Resource: signalfx_metric_ruleset

Provides an Observability Cloud resource for managing metric rulesets

## Example Usage

```tf
resource "signalfx_metric_ruleset" "cpu_utilization_metric_ruleset" {
    metric_name = "cpu.utilization"

    aggregation_rules {
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

    routing_rule = {
        destination = "RealTime"
    }
}
```

## Argument Reference

The following arguments are supported in the resource block:

* `metric_name` - (Required) Name of the input metric
* `aggregation_rules` - (Optional) List of aggregation rules for the metric
  * `enabled` - (Required) When false, this rule will not generate aggregated MTSs
  * `matcher` - (Required) Matcher object
    * `type` - (Required) Type of matcher. Must always be "dimension"
    * `filters` - (Optional) List of filters to filter the set of input MTSs
      * `property` - (Required) - Name of the dimension
      * `property_value` - (Required) - Value of the dimension
      * `not` - When true, this filter will match all values not matching the property_values
  * `aggregator` - (Required) - Aggregator object
    * `type` - (Required) Type of aggregator. Must always be "rollup"
    * `dimensions` - (Required) List of dimensions to either be kept or dropped in the new aggregated MTSs
    * `drop_dimensions` - (Required) when true, the specified dimensions will be dropped from the aggregated MTSs
    * `output_name` - (Required) name of the new aggregated metric
* `routing_rule` - (Required) Routing Rule object
  * `destination` - (Required) - end destination of the input metric