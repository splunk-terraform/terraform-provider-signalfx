package signalfx

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	sfx "github.com/signalfx/signalfx-go"
)

const newTeamConfig = `
resource "signalfx_team" "myteamXX" {
    name = "Super Cool Team"
		description = "Fart noise"
}
`

const updatedTeamConfig = `
resource "signalfx_team" "myteamXX" {
    name = "Super Cool Team NEW"
		description = "Fart noise NEW"
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
				),
			},
			// Update Everything
			{
				Config: updatedTeamConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamResourceExists,
					resource.TestCheckResourceAttr("signalfx_team.myteamXX", "name", "Super Cool Team NEW"),
				),
			},
		},
	})
}

func testAccCheckTeamResourceExists(s *terraform.State) error {
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_team":
			team, err := client.GetTeam(rs.Primary.ID)
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
	client, _ := sfx.NewClient(os.Getenv("SFX_AUTH_TOKEN"))
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_team":
			team, _ := client.GetTeam(rs.Primary.ID)
			if team != nil {
				return fmt.Errorf("Found deleted team %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
