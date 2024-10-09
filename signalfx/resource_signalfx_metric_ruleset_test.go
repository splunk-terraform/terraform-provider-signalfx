// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	demoTransLatencyRuleset = `
resource "signalfx_metric_ruleset" "demo_trans_latency_metric_ruleset" {
    metric_name = "demo.trans.latency"
	description = "demo_trans_latency_metric_ruleset with aggregation"

	aggregation_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_datacenter"
                property_value = [ "Paris", "Tokyo" ]
                not = false
            }
        }
        aggregator {
            type = "rollup"
            dimensions = [ "demo_customer" ]
            drop_dimensions = false
            output_name = "demo_trans_latency.by.demo_datacenter.agg"
        }
		description = "aggregation rule 1"
    }

    routing_rule {
        destination = "RealTime"
    }
}
`
)

const (
	demoTransLatencyRulesetUpdated = `
resource "signalfx_metric_ruleset" "demo_trans_latency_metric_ruleset" {
    metric_name = "demo.trans.latency"
	description = "demo_trans_latency_metric_ruleset with aggregation updated"

    aggregation_rules {
        name = "newRule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_datacenter"
                property_value = [ "Paris", "Tokyo" ]
                not = false
            }
        }
        aggregator {
            type = "rollup"
            dimensions = [ "demo_customer" ]
            drop_dimensions = false
            output_name = "demo_trans_latency.by.demo_datacenter.agg"
        }
		description = "aggregation rule 1 updated"
    }

    aggregation_rules {
        enabled = false
        matcher {
            type = "dimension"
        }
        aggregator {
            type = "rollup"
            dimensions = [ "demo_host" ]
            drop_dimensions = false
            output_name = "demo_trans_latency.by.demo_host.agg"
        }
		description = "aggregation rule 2"
    }

    routing_rule {
        destination = "Drop"
    }
}
`
)

const (
	archivedDemoTransCountRuleset = `
resource "signalfx_metric_ruleset" "demo_trans_count_metric_ruleset" {
    metric_name = "demo.trans.count"

    exception_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_datacenter"
                property_value = [ "Paris" ]
                not = false
            }
        }
    }

    exception_rules {
        name = "rule2"
		description ="exception rule 2"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_host"
                property_value = [ "server1", "server3" ]
                not = false
            }
        }
    }

    routing_rule {
        destination = "Archived"
    }
}
`
)

// Update: rule 1 - add Tokyo; rule 2 - replace filters with server3; add rule 3
const (
	archivedDemoTransCountRulesetUpdated = `
resource "signalfx_metric_ruleset" "demo_trans_count_metric_ruleset" {
    metric_name = "demo.trans.count"

    exception_rules {
        name = "rule1"
		description = "exception rule 1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_datacenter"
                property_value = [ "Paris", "Tokyo" ]
                not = false
            }
        }
    }

    exception_rules {
        name = "rule2"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_host"
                property_value = [ "server4" ]
                not = false
            }
        }
    }

    exception_rules {
        name = "rule3"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_customer"
                property_value = [ "customer1@email.com", "customer2@email.com", "customer3@email.com" ]
                not = false
            }
        }
    }

    routing_rule {
        destination = "Archived"
    }
}
`
)

// Update: rule 2 - remove it; rule 3 - add server2 to filters
const (
	archivedDemoTransCountRulesetUpdatedRemoveRule2 = `
resource "signalfx_metric_ruleset" "demo_trans_count_metric_ruleset" {
    metric_name = "demo.trans.count"

    exception_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_datacenter"
                property_value = [ "Paris", "Tokyo" ]
                not = false
            }
        }
    }

    exception_rules {
        name = "rule3"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_customer"
                property_value = [ "customer1@email.com", "customer2@email.com", "customer3@email.com" ]
                not = false
            }
            filters {
                property = "demo_host"
                property_value = [ "server2" ]
                not = false
            }

        }
    }

	routing_rule {
        destination = "Archived"
    }
}
`
)

func TestAccMetricRulesetAggregation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccMetricRulesetDestroy,
		Steps: []resource.TestStep{
			// Validate plan
			{
				Config:             demoTransLatencyRuleset,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Create it
			{
				Config: demoTransLatencyRuleset,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "metric_name", "demo.trans.latency"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "description", "demo_trans_latency_metric_ruleset with aggregation"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "version", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property_value.0", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property_value.1", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.not", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.type", "rollup"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.output_name", "demo_trans_latency.by.demo_datacenter.agg"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.drop_dimensions", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.dimensions.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.dimensions.0", "demo_customer"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.description", "aggregation rule 1"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "routing_rule.0.destination", "RealTime"),
				),
			},
			// Update it
			{
				Config: demoTransLatencyRulesetUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "metric_name", "demo.trans.latency"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "description", "demo_trans_latency_metric_ruleset with aggregation updated"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "version", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.name", "newRule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property_value.0", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.property_value.1", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.0.filters.0.not", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.type", "rollup"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.output_name", "demo_trans_latency.by.demo_datacenter.agg"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.drop_dimensions", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.dimensions.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.0.dimensions.0", "demo_customer"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.description", "aggregation rule 1 updated"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.name", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.type", "rollup"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.output_name", "demo_trans_latency.by.demo_host.agg"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.drop_dimensions", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.dimensions.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.dimensions.0", "demo_host"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.description", "aggregation rule 2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "routing_rule.0.destination", "Drop"),
				),
			},
		},
	})
}

func TestAccMetricRulesetArchived(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccMetricRulesetDestroy,
		Steps: []resource.TestStep{
			// Validate plan
			{
				Config:             archivedDemoTransCountRuleset,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Create a new Archived Ruleset: metric demo.trans.count
			{
				Config: archivedDemoTransCountRuleset,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "metric_name", "demo.trans.count"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "version", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.0", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.name", "rule2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.description", "exception rule 2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property", "demo_host"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.0", "server1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.1", "server3"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "routing_rule.0.destination", "Archived"),
				),
			},
			// Validate plan
			{
				Config:             archivedDemoTransCountRulesetUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Update: add Paris to rule 1, replace with server3 in rule 2, add rule 3
			{
				Config: archivedDemoTransCountRulesetUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "metric_name", "demo.trans.count"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "version", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.#", "3"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.description", "exception rule 1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.0", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.1", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.name", "rule2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property", "demo_host"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.0", "server4"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.name", "rule3"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.filters.0.property", "demo_customer"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.filters.0.property_value.#", "3"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.filters.0.property_value.0", "customer1@email.com"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.filters.0.property_value.1", "customer2@email.com"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.filters.0.property_value.2", "customer3@email.com"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.2.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "routing_rule.0.destination", "Archived"),
				),
			},
			// Validate plan
			{
				Config:             archivedDemoTransCountRulesetUpdatedRemoveRule2,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Remove rule 2
			{
				Config: archivedDemoTransCountRulesetUpdatedRemoveRule2,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "metric_name", "demo.trans.count"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "version", "3"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.#", "2"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.0", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.1", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.name", "rule3"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.description", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property", "demo_customer"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.#", "3"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.0", "customer1@email.com"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.1", "customer2@email.com"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.2", "customer3@email.com"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.not", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.1.property", "demo_host"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.1.property_value.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.1.property_value.0", "server2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.1.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "routing_rule.0.destination", "Archived"),
				),
			},
		},
	})
}

func TestAccMetricRulesetRestoration(t *testing.T) {
	// 15 minutes ago in milliseconds
	startTime := (time.Now().Unix() - 900) * 1000
	stopTime := (time.Now().Unix() - 200) * 1000

	archivedCartSizeRestore := fmt.Sprintf(`
resource "signalfx_metric_ruleset" "cart_size" {
    metric_name = "cart.size"

    exception_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "customer-spend"
                property_value = [ "low" ]
                not = false
            }
        }
		restoration {
			start_time = %d
			stop_time = %d
		}
    }

	routing_rule {
        destination = "Archived"
    }
}	`, startTime, stopTime)

	archivedCartSizeRestoreUpdate := fmt.Sprintf(`
resource "signalfx_metric_ruleset" "cart_size" {
    metric_name = "cart.size"

    exception_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "customer-spend"
                property_value = [ "low", "medium" ]
                not = false
            }
        }
		restoration {
			start_time = %d
			stop_time = %d
		}
    }

	routing_rule {
        destination = "Archived"
    }
}	`, startTime, stopTime)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccMetricRulesetDestroy,
		Steps: []resource.TestStep{
			// Validate plan
			{
				Config:             archivedCartSizeRestore,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Create a new Archived Ruleset metric cart.size with restoration
			{
				Config: archivedCartSizeRestore,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "metric_name", "cart.size"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "version", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property", "customer-spend"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.0", "low"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.restoration.0.start_time", strconv.FormatInt(startTime, 10)),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.restoration.0.stop_time", strconv.FormatInt(stopTime, 10)),
				),
			},
			// Validate plan
			{
				Config:             archivedCartSizeRestoreUpdate,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Update ruleset by adding filter property medium.
			{
				Config: archivedCartSizeRestoreUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "metric_name", "cart.size"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "version", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property", "customer-spend"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.0", "low"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.1", "medium"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.restoration.0.start_time", strconv.FormatInt(startTime, 10)),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.restoration.0.stop_time", strconv.FormatInt(stopTime, 10)),
				),
			},
		},
	})
}
func TestAccMetricRulesetRestorationNoStopTime(t *testing.T) {
	// 15 minutes ago in milliseconds
	startTime := (time.Now().Unix() - 900) * 1000

	archivedCartSizeRestore := fmt.Sprintf(`
resource "signalfx_metric_ruleset" "cart_size" {
    metric_name = "cart.size"

    exception_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "customer-spend"
                property_value = [ "low" ]
                not = false
            }
        }
		restoration {
			start_time = %d
		}
    }

	routing_rule {
        destination = "Archived"
    }
}	`, startTime)

	archivedCartSizeRestoreUpdate := fmt.Sprintf(`
resource "signalfx_metric_ruleset" "cart_size" {
    metric_name = "cart.size"

    exception_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "customer-spend"
                property_value = [ "low", "medium" ]
                not = false
            }
        }
		restoration {
			start_time = %d
		}
    }

	routing_rule {
        destination = "Archived"
    }
}	`, startTime)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccMetricRulesetDestroy,
		Steps: []resource.TestStep{
			// Validate plan
			{
				Config:             archivedCartSizeRestore,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Create a new Archived Ruleset metric cart.size with restoration
			{
				Config: archivedCartSizeRestore,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "metric_name", "cart.size"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "version", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property", "customer-spend"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.0", "low"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.restoration.0.start_time", strconv.FormatInt(startTime, 10)),
				),
			},
			// Validate plan
			{
				Config:             archivedCartSizeRestoreUpdate,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Update ruleset by adding filter property medium.
			{
				Config: archivedCartSizeRestoreUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "metric_name", "cart.size"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "version", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property", "customer-spend"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.0", "low"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.property_value.1", "medium"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.cart_size", "exception_rules.0.restoration.0.start_time", strconv.FormatInt(startTime, 10)),
				),
			},
		},
	})
}
func testAccMetricRulesetExists(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_metric_ruleset":
			metricRuleset, err := client.GetMetricRuleset(context.TODO(), rs.Primary.ID)
			if err != nil || *metricRuleset.Id != rs.Primary.ID {
				return fmt.Errorf("Error finding metric ruleset %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccMetricRulesetDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_metric_ruleset":
			metricRuleset, _ := client.GetMetricRuleset(context.TODO(), rs.Primary.ID)
			if metricRuleset != nil {
				return fmt.Errorf("Found deleted metric ruleset %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}
