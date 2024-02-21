package signalfx

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/signalfx/signalfx-go/slo"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	sloDescription    = "SLO description"
	sloProgramText    = `G = data('spans.count', filter=filter('sf_error', 'false') and filter('sf_environment', 'lab0') and filter('sf_service', 'apm-indexer-api'))\nT = data('spans.count', filter=filter('sf_environment', 'lab0') and filter('sf_service', 'apm-indexer-api'))`
	sloTarget         = 98
	compliancePeriod  = "30d"
	alertNotification = "Email,foo-alerts@example.com"
	fireLasting       = "15m"
	shortWindow       = "15m"

	updatedSloDescription    = "Updated SLO description"
	updatedSloProgramText    = `G = data('spans.count', filter=filter('sf_error', 'false') and filter('sf_service', 'apm-indexer-api'))\nT = data('spans.count', filter=filter('sf_service', 'apm-indexer-api'))`
	updatedSloTarget         = 99
	updatedAlertNotification = "Email,new-alerts@example.com"
	updateFireLasting        = "1m"
	percentErrorBudgetLeft   = 12
)

var sloName = "test slo " + time.Now().String() // we are checking the uniqueness of the slo name, so we need some randomness here

var newSloConfig = fmt.Sprintf(`
resource "signalfx_slo" "test_slo" {
    name = "%s"
    type = "RequestBased"
	description = "%s"
	input {
		program_text = "%s"
		good_events_label = "G"
		total_events_label = "T"
	}

	target {
		type="RollingWindow"
		slo=%s
		compliance_period = "%s"
		
		alert_rule {
			type = "BREACH"
			
			rule {
				severity = "Critical"
				notifications = ["%s"]
				parameters {
					fire_lasting = "%s"
				}
			}
		}
		
		alert_rule {
			type = "BURN_RATE"

			rule {
				severity = "Warning"
				parameters {
					short_window_1 = "%s"
				}
			}
		}
	}
}
`, sloName, sloDescription, sloProgramText, strconv.Itoa(sloTarget), compliancePeriod, alertNotification, fireLasting, shortWindow)

var updateSloConfig = fmt.Sprintf(`
resource "signalfx_slo" "test_slo" {
    name = "%s"
    type = "RequestBased"
	description = "%s"
	input {
		program_text = "%s"
		good_events_label = "G"
		total_events_label = "T"
	}

	target {
		type="RollingWindow"
		slo=%s
		compliance_period = "%s"
		
		alert_rule {
			type = "BREACH"
			
			rule {
				severity = "Critical"
				notifications = ["%s"]
				parameters {
					fire_lasting = "%s"
				}
			}
		}
		
		alert_rule {
			type = "ERROR_BUDGET_LEFT"

			rule {
				severity = "Warning"
				parameters {
					percent_error_budget_left = %s					
				}
			}
		}
	}
}
`, sloName, updatedSloDescription, updatedSloProgramText, strconv.Itoa(updatedSloTarget), compliancePeriod, updatedAlertNotification, updateFireLasting, strconv.Itoa(percentErrorBudgetLeft))

var invalidSloProgramTextInput = fmt.Sprintf(`
resource "signalfx_slo" "test_slo" {
    name = "%s"
    type = "RequestBased"
	input {
		program_text = "G = da('spans.count', filter=filter('sf_error', 'false') and filter('sf_environment', 'lab0') and filter('sf_service', 'apm-indexer-api'))\n"
		good_events_label = "G"
		total_events_label = "T"
	}

	target {
		type="RollingWindow"
		slo=97
		compliance_period = "30d"
		
		alert_rule {
			type = "BREACH"
			
			rule {
				severity = "Critical"
			}
		}
	}
}
`, sloName)

var invalidSloTargetValue = fmt.Sprintf(`
resource "signalfx_slo" "test_slo" {
    name = "%s"
    type = "RequestBased"
	input {
		program_text = "G = data('spans.count', filter=filter('sf_error', 'false') and filter('sf_environment', 'lab0') and filter('sf_service', 'apm-indexer-api'))\nT = data('spans.count', filter=filter('sf_environment', 'lab0') and filter('sf_service', 'apm-indexer-api'))"
		good_events_label = "G"
		total_events_label = "T"
	}

	target {
		type="RollingWindow"
		slo=101
		compliance_period = "30d"
		
		alert_rule {
			type = "BREACH"
			
			rule {
				severity = "Warning"
			}
		}
	}
}
`, sloName)

func TestAccCreateUpdateSlo(t *testing.T) {
	const sloResourceName = "signalfx_slo.test_slo"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSloDestroy,
		Steps: []resource.TestStep{
			// Check invalid slo programText input
			{
				Config:      invalidSloProgramTextInput,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("Invalid programText"),
			},
			// Check invalid SLO target
			{
				Config:      invalidSloTargetValue,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("expected target.0.slo to be in the range \\(0.000000 - 100.000000\\), got 101.000000"),
			},
			// Validate plan
			{
				Config:             newSloConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Create It
			{
				Config: newSloConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSloResourceExists,
					resource.TestCheckResourceAttr(sloResourceName, "name", sloName),
					resource.TestCheckResourceAttr(sloResourceName, "type", "RequestBased"),
					resource.TestCheckResourceAttr(sloResourceName, "description", sloDescription),

					resource.TestCheckResourceAttr(sloResourceName, "input.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "input.0.program_text", strings.Replace(sloProgramText, "\\n", "\n", -1)),
					resource.TestCheckResourceAttr(sloResourceName, "input.0.good_events_label", "G"),
					resource.TestCheckResourceAttr(sloResourceName, "input.0.total_events_label", "T"),

					resource.TestCheckResourceAttr(sloResourceName, "target.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.type", "RollingWindow"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.slo", strconv.Itoa(sloTarget)),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.compliance_period", compliancePeriod),

					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.#", "2"),

					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.type", "BREACH"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.severity", "Critical"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.notifications.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.notifications.0", alertNotification),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.parameters.0.fire_lasting", fireLasting),

					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.type", "BURN_RATE"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.severity", "Warning"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.notifications.#", "0"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.parameters.0.short_window_1", shortWindow),

					// Force sleep before refresh at the end of test execution
					waitBeforeTestStepPlanRefresh,
				),
			},
			{
				ResourceName:      sloResourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc(sloResourceName),
				ImportStateVerify: true,
			},
			// Update It
			{
				Config: updateSloConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSloResourceExists,
					resource.TestCheckResourceAttr(sloResourceName, "name", sloName),
					resource.TestCheckResourceAttr(sloResourceName, "type", "RequestBased"),
					resource.TestCheckResourceAttr(sloResourceName, "description", updatedSloDescription),

					resource.TestCheckResourceAttr(sloResourceName, "input.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "input.0.program_text", strings.Replace(updatedSloProgramText, "\\n", "\n", -1)),
					resource.TestCheckResourceAttr(sloResourceName, "input.0.good_events_label", "G"),
					resource.TestCheckResourceAttr(sloResourceName, "input.0.total_events_label", "T"),

					resource.TestCheckResourceAttr(sloResourceName, "target.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.type", "RollingWindow"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.slo", strconv.Itoa(updatedSloTarget)),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.compliance_period", compliancePeriod),

					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.#", "2"),

					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.type", "BREACH"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.severity", "Critical"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.notifications.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.notifications.0", updatedAlertNotification),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.0.rule.0.parameters.0.fire_lasting", updateFireLasting),

					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.type", "ERROR_BUDGET_LEFT"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.severity", "Warning"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.notifications.#", "0"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.parameters.#", "1"),
					resource.TestCheckResourceAttr(sloResourceName, "target.0.alert_rule.1.rule.0.parameters.0.percent_error_budget_left", strconv.Itoa(percentErrorBudgetLeft)),
				),
			},
		},
	})
}

func testAccCheckSloResourceExists(s *terraform.State) error {
	sloId, sloObject, err := getSloObject(s)

	if err != nil || sloObject.Id != sloId {
		return fmt.Errorf("Error finding SLO %s: %s", sloId, err)
	}

	return nil
}

func testAccSloDestroy(s *terraform.State) error {
	sloId, sloObject, _ := getSloObject(s)

	if sloObject != nil {
		return fmt.Errorf("Found deleted SLO %s", sloId)
	}

	return nil
}

func getSloObject(s *terraform.State) (string, *slo.SloObject, error) {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		sloId := rs.Primary.ID

		switch rs.Type {
		case "signalfx_slo":
			sloObject, err := client.GetSlo(context.TODO(), sloId)
			return sloId, sloObject, err
		default:
			return sloId, nil, fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return "", nil, nil
}
