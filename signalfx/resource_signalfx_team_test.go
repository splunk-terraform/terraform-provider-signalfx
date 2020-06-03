package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const newTeamConfig = `
resource "signalfx_team" "myteamXX" {
    name = "Super Cool Team"
		description = "Fart noise"

    notifications_critical = [ "Email,test@example.com" ]
    notifications_default = [ "Webhook,,secret,https://www.example.com" ]
    notifications_info = [ "Webhook,,secret,https://www.example.com/2" ]
    notifications_major = [ "Webhook,,secret,https://www.example.com/3" ]
    notifications_minor = [ "Webhook,,secret,https://www.example.com/4" ]
    notifications_warning = [ "Webhook,,secret,https://www.example.com/5" ]
}
`

const updatedTeamConfig = `
resource "signalfx_team" "myteamXX" {
    name = "Super Cool Team NEW"
		description = "Fart noise NEW"

    notifications_critical = [ "Email,test@example.com" ]
    notifications_default = [ "Webhook,,secret,https://www.example.com" ]
    notifications_info = [ "Webhook,,secret,https://www.example.com/2" ]
    notifications_major = [ "Webhook,,secret,https://www.example.com/3" ]
    notifications_minor = [ "Webhook,,secret,https://www.example.com/4" ]
    notifications_warning = [ "Webhook,,secret,https://www.example.com/5" ]
}
`

func TestAccCreateUpdateTeam(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTeamDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newTeamConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamResourceExists,
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "name", "Super Cool Team"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "description", "Fart noise"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_critical.#", "1"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_critical.0", "Email,test@example.com"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_default.#", "1"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_default.0", "Webhook,,secret,https://www.example.com"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_info.#", "1"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_info.0", "Webhook,,secret,https://www.example.com/2"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_major.#", "1"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_major.0", "Webhook,,secret,https://www.example.com/3"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_minor.#", "1"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_minor.0", "Webhook,,secret,https://www.example.com/4"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_warning.#", "1"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "notifications_warning.0", "Webhook,,secret,https://www.example.com/5"),
				),
			},
			// Update Everything
			{
				Config: updatedTeamConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamResourceExists,
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "name", "Super Cool Team NEW"),
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "description", "Fart noise NEW"),
				),
			},
		},
	})
}

func testAccCheckTeamResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_team":
			team, err := client.GetTeam(context.TODO(), rs.Primary.ID)
			if team.Id != rs.Primary.ID || err != nil {
				return fmt.Errorf("Error finding team %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccTeamDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_team":
			team, _ := client.GetTeam(context.TODO(), rs.Primary.ID)
			if team != nil {
				return fmt.Errorf("Found deleted team %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
