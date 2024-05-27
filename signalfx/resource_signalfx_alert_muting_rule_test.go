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

func TestAccCreateUpdateFutureAlertMutingRule(t *testing.T) {
	firstTime := time.Now().Unix() + 86400
	secondTime := time.Now().Unix() + (86400 * 2)

	newAlertMutingRuleConfigFuture := fmt.Sprintf(`
	resource "signalfx_alert_muting_rule" "rool_mooter_two" {
	    description = "mooted it FUTURE"

	    start_time = %d

	    filter {
	      property = "foo"
	      property_value = "bar"
	    }

        recurrence {
          unit = "d"
          value = 2
        }
	}
	`, firstTime)

	updatedAlertMutingRuleConfigFuture := fmt.Sprintf(`
	resource "signalfx_alert_muting_rule" "rool_mooter_two" {
	    description = "mooted it FUTURE"

	    start_time = %d

	    filter {
	      property = "foo"
	      property_value = "bar"
	    }

        recurrence {
          unit = "d"
          value = 4
        }
	}
	`, secondTime)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAlertMutingRuleDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newAlertMutingRuleConfigFuture,
				Check: resource.ComposeTestCheckFunc(
					testAccCreateUpdateAlertMutingRuleResourceExists,
					resource.TestCheckResourceAttr("signalfx_alert_muting_rule.rool_mooter_two", "description", "mooted it FUTURE"),
					resource.TestCheckResourceAttr("signalfx_alert_muting_rule.rool_mooter_two", "start_time", strconv.Itoa(int(firstTime))),
					resource.TestCheckResourceAttr("signalfx_alert_muting_rule.rool_mooter_two", "recurrence.0.unit", "d"),
					resource.TestCheckResourceAttr("signalfx_alert_muting_rule.rool_mooter_two", "recurrence.0.value", "2"),
				),
			},
			{
				ResourceName:      "signalfx_alert_muting_rule.rool_mooter_two",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_alert_muting_rule.rool_mooter_two"),
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"start_time",
				},
			},
			// Update It
			{
				Config: updatedAlertMutingRuleConfigFuture,
				Check: resource.ComposeTestCheckFunc(
					testAccCreateUpdateAlertMutingRuleResourceExists,
					resource.TestCheckResourceAttr("signalfx_alert_muting_rule.rool_mooter_two", "start_time", strconv.Itoa(int(secondTime))),
					resource.TestCheckResourceAttr("signalfx_alert_muting_rule.rool_mooter_two", "recurrence.0.unit", "d"),
					resource.TestCheckResourceAttr("signalfx_alert_muting_rule.rool_mooter_two", "recurrence.0.value", "4"),
				),
			},
		},
	})
}

func testAccCreateUpdateAlertMutingRuleResourceExists(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_alert_muting_rule":
			amr, err := client.GetAlertMutingRule(context.TODO(), rs.Primary.ID)
			if amr.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding alert muting rule %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccAlertMutingRuleDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_alert_muting_rule":
			amr, _ := client.GetAlertMutingRule(context.TODO(), rs.Primary.ID)
			if amr != nil {
				return fmt.Errorf("Found deleted alert muting rule %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
