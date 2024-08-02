package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	demoTransLatencyRuleset = `
resource "signalfx_metric_ruleset" "demo_trans_latency_metric_ruleset" {
    metric_name = "demo.trans.latency"

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
    }

    routing_rule {
        destination = "Drop"
    }
}
`
)

const (
	demoTransCountRuleset = `
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

    routing_rule {
        destination = "Archived"
    }
}
`
)

const (
	demoTransCountRulesetUpdated = `
resource "signalfx_metric_ruleset" "demo_trans_count_metric_ruleset" {
    metric_name = "demo.trans.count"

    exception_rules {
        name = "rule1"
        enabled = true
        matcher {
            type = "dimension"
            filters {
                property = "demo_datacenter"
                property_value = [ "Tokyo" ]
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

func TestAccCreateUpdateMetricRuleset(t *testing.T) {
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
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "routing_rule.0.destination", "RealTime"),
				),
			},
			// Update it
			{
				Config: demoTransLatencyRulesetUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "metric_name", "demo.trans.latency"),
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
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.name", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.type", "rollup"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.output_name", "demo_trans_latency.by.demo_host.agg"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.drop_dimensions", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.dimensions.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.0.dimensions.0", "demo_host"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "routing_rule.0.destination", "Archived"),
				),
			},
		},
	})
}

func TestAccCreateUpdateMetricRulesetArchived(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccMetricRulesetDestroy,
		Steps: []resource.TestStep{
			// Validate plan
			{
				Config:             demoTransCountRuleset,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Create it
			{
				Config: demoTransCountRuleset,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "metric_name", "demo.trans.count"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "version", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.0", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.1", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "routing_rule.0.destination", "Archived"),
				),
			},
			// Validate plan
			{
				Config:             demoTransCountRulesetUpdated,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Update it
			{
				Config: demoTransCountRulesetUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccMetricRulesetExists,
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "metric_name", "demo.trans.count"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "version", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.name", "rule1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.property_value.0", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.0.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.name", "rule2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property", "demo_host"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.0", "server1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.property_value.1", "server3"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "exception_rules.1.matcher.0.filters.0.not", "false"),

					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_count_metric_ruleset", "routing_rule.0.destination", "Drop"),
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
