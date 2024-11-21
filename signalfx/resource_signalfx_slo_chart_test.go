// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testSloName = "test-slo-" + time.Now().String()
var testSecondSloName = "test-second-slo-" + time.Now().String()

var sloChartCreateConfig = fmt.Sprintf(`
resource "signalfx_slo" "test_slo" {
    name = "%s"
    type = "RequestBased"
	description = ""
	input {
		program_text = "G = data('spans.count', filter=filter('sf_error', 'false') and filter('sf_service', 'apm-indexer-api'))\nT = data('spans.count', filter=filter('sf_service', 'apm-indexer-api'))"
		good_events_label = "G"
		total_events_label = "T"
	}

	target {
		type="RollingWindow"
		slo=99
		compliance_period = "7d"
		
		alert_rule {
			type = "BREACH"
			
			rule {
				severity = "Critical"
				notifications = []
				parameterized_body = "test"
				parameterized_subject = "test"
			}
		}
	}
}

resource "signalfx_slo_chart" "slo_chart" {
  slo_id = "${signalfx_slo.test_slo.id}"
}
`, testSloName)

var sloChartUpdateConfig = fmt.Sprintf(`
resource "signalfx_slo" "test_slo" {
    name = "%s"
    type = "RequestBased"
	description = ""
	input {
		program_text = "G = data('spans.count', filter=filter('sf_error', 'false') and filter('sf_service', 'apm-indexer-api'))\nT = data('spans.count', filter=filter('sf_service', 'apm-indexer-api'))"
		good_events_label = "G"
		total_events_label = "T"
	}

	target {
		type="RollingWindow"
		slo=99
		compliance_period = "7d"
		
		alert_rule {
			type = "BREACH"
			
			rule {
				severity = "Critical"
				notifications = []
				parameterized_body = "test"
				parameterized_subject = "test"
			}
		}
	}
}

resource "signalfx_slo" "second_test_slo" {
    name = "%s"
    type = "RequestBased"
	description = ""
	input {
		program_text = "G = data('spans.count', filter=filter('sf_error', 'false') and filter('sf_service', 'apm-indexer-api'))\nT = data('spans.count', filter=filter('sf_service', 'apm-indexer-api'))"
		good_events_label = "G"
		total_events_label = "T"
	}

	target {
		type="RollingWindow"
		slo=99
		compliance_period = "7d"
		
		alert_rule {
			type = "BREACH"
			
			rule {
				severity = "Critical"
				notifications = []
				parameterized_body = "test"
				parameterized_subject = "test"
			}
		}
	}
}

resource "signalfx_slo_chart" "slo_chart" {
  slo_id = "${signalfx_slo.second_test_slo.id}"
}
`, testSloName, testSecondSloName)

func TestAccCreateUpdateSloChart(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSloChartDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: sloChartCreateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSloChartResourceExists,
					resource.TestMatchResourceAttr("signalfx_slo_chart.slo_chart", "slo_id", regexp.MustCompile(".+")),
				),
			},
			// Update Everything
			{
				Config: sloChartUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSloChartResourceExists,
					resource.TestMatchResourceAttr("signalfx_slo_chart.slo_chart", "slo_id", regexp.MustCompile(".+")),
				),
			},
		},
	})
}

func testAccCheckSloChartResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_slo_chart":
			chart, err := client.GetChart(context.TODO(), rs.Primary.ID)
			if chart.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding chart %s: %w", rs.Primary.ID, err)
			}
		case "signalfx_slo":
			// ignore SLO resource for this test (SLO is required to create SLO chart but is not an object under test)
			return nil
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccSloChartDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_slo_chart":
			chart, _ := client.GetChart(context.TODO(), rs.Primary.ID)
			if chart != nil {
				return fmt.Errorf("Found deleted chart %s", rs.Primary.ID)
			}
		case "signalfx_slo":
			// ignore SLO resource for this test (SLO is required to create SLO chart but is not an object under test)
			return nil
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
