package signalfx

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newIntegrationJiraConfig = `
resource "signalfx_jira_integration" "jira_myteamXX" {
    name = "JiraFoo"
    enabled = false

    auth_method = "UsernameAndPassword"
    username = "yoosername"
    password = "paasword"

    assignee_name = "testytesterson"
    assignee_display_name = "Testy Testerson"

    base_url = "https://www.example.com"
    issue_type = "Story"
    project_key = "TEST"
}
`

const updatedIntegrationJiraConfig = `
resource "signalfx_jira_integration" "jira_myteamXX" {
    name = "JiraFoo NEW"
    enabled = false

    auth_method = "EmailAndToken"
    user_email = "yoosername@example.com"
    api_token = "abc123"

    assignee_name = "testytesterson"
    assignee_display_name = "Testy Testerson"

    base_url = "https://www.example.com"
    issue_type = "Story"
    project_key = "TEST"
}
`

func TestAccCreateUpdateIntegrationJira(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationJiraDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationJiraConfig,
				Check:  testAccCheckIntegrationJiraResourceExists,
			},
			{
				ResourceName:      "signalfx_jira_integration.jira_myteamXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_jira_integration.jira_myteamXX"),
				ImportStateVerify: true,
				// The API doesn't return this value, so blow it up
				ImportStateVerifyIgnore: []string{"password", "api_token"},
			},
			// Update It
			{
				Config: updatedIntegrationJiraConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationJiraResourceExists,
					resource.TestCheckResourceAttr("signalfx_jira_integration.jira_myteamXX", "name", "JiraFoo NEW"),
				),
			},
		},
	})
}

func testAccCheckIntegrationJiraResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_jira_integration":
			integration, err := client.GetJiraIntegration(rs.Primary.ID)
			if integration == nil {
				return fmt.Errorf("Error finding integration %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testAccIntegrationJiraDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_jira_integration":
			integration, _ := client.GetJiraIntegration(rs.Primary.ID)
			if integration != nil {
				return fmt.Errorf("Found deleted integration %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
