---
layout: "signalfx"
page_title: "Splunk Observability Cloud: signalfx_metric_ruleset"
sidebar_current: "docs-signalfx-resource-metric-ruleset"
description: |-
Allows Terraform to create and manage Splunk Infrastructure Monitoring metric rulesets
---

# Resource: signalfx_metric_ruleset

Provides an Observability Cloud resource for managing metric rulesets.

~> **NOTE** When managing metric rulesets to drop data use a session token for an administrator to authenticate the Splunk Observability Cloud provider. See [Operations that require a session token for an administrator](https://dev.splunk.com/observability/docs/administration/authtokens#Operations-that-require-a-session-token-for-an-administrator). Otherwise you'll receive a 4xx error.

## Example

```tf
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
```

## Arguments

The following arguments are supported in the resource block:

* `metric_name` - (Required) Name of the input metric
* `description` - (Optional) Information about the metric ruleset
* `aggregation_rules` - (Optional) List of aggregation rules for the metric
  * `enabled` - (Required) When false, this rule will not generate aggregated MTSs
  * `name` - (Optional) name of the aggregation rule
  * `description` - (Optional) Information about an aggregation rule
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
* `exception_rules` - (Optional) List of exception rules for the metric
  * `enabled` - (Required) When false, this rule will not route matched data to real-time
  * `name` - (Required) name of the exception rule
  * `description` - (Optional) Information about an exception rule
  * `matcher` - (Required) Matcher object
    * `type` - (Required) Type of matcher. Must always be "dimension"
    * `filters` - (Required) List of filters to filter the set of input MTSs
      * `property` - (Required) - Name of the dimension
      * `property_value` - (Required) - Value of the dimension
      * `not` - When true, this filter will match all values not matching the property_values
  * `restoration` - (Optional) Properties of a restoration job
    * `start_time` - (Required) Time from which the restoration job will restore archived data, in the form of *nix time in milliseconds
    * `stop_time` - (Required) Time to which the restoration job will restore archived data, in the form of *nix time in milliseconds

* `routing_rule` - (Required) Routing Rule object
  * `destination` - (Required) - end destination of the input metric. Must be `RealTime`, `Archived`, or `Drop`