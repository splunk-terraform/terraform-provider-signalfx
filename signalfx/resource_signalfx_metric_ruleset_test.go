package signalfx

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"testing"
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
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property_value.708291584", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property_value.3654727247", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.not", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.type", "rollup"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.output_name", "demo_trans_latency.by.demo_datacenter.agg"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.drop_dimensions", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.dimensions.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.dimensions.2525319496", "demo_customer"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "routing_rule.destination", "RealTime"),
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
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property", "demo_datacenter"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property_value.#", "2"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property_value.708291584", "Paris"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.property_value.3654727247", "Tokyo"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.matcher.3477209961.filters.0.not", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.type", "rollup"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.output_name", "demo_trans_latency.by.demo_datacenter.agg"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.drop_dimensions", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.dimensions.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.0.aggregator.3002091158.dimensions.2525319496", "demo_customer"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.name", ""),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.matcher.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.matcher.247457994.type", "dimension"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.2210828267.type", "rollup"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.2210828267.output_name", "demo_trans_latency.by.demo_host.agg"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.2210828267.drop_dimensions", "false"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.2210828267.dimensions.#", "1"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "aggregation_rules.1.aggregator.2210828267.dimensions.2152883157", "demo_host"),
					resource.TestCheckResourceAttr("signalfx_metric_ruleset.demo_trans_latency_metric_ruleset", "routing_rule.destination", "Drop"),
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
