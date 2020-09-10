package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	newTeamConfig = `
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

	updatedTeamConfig = `
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

	teamWithDetector = `
resource "signalfx_detector" "team_detector" {
    name = "team detector"
    description = "team detector"
    max_delay = 30

    program_text = <<-EOF
        signal = data('app.delay').max().publish('app delay')
        detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
	EOF
    rule {
        description = "maximum > 60 for 30m"
        severity = "Critical"
        detect_label = "Processing old messages 30m"
        notifications = ["Email,foo-alerts@example.com"]
    }
}

resource "signalfx_team" "team_with_detector" {
	name = "Team with detector"
	description = "Team with detector"
	detectors = ["${signalfx_detector.team_detector.id}"]
}
`

	teamWithDashboardGroup = `
resource "signalfx_dashboard_group" "team_dashboard_group" {
    name = "My team dashboard group"
    description = "Cool dashboard group"
}

resource "signalfx_team" "team_with_dashboard_group" {
	name = "Team with dashboard_group"
	description = "Team with dashboard_group"
	dashboard_groups = ["${signalfx_dashboard_group.team_dashboard_group.id}"]
}
`
)

func TestAccTeamWithDetector(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: teamWithDetector,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamResourceExists,
					resource.TestCheckResourceAttr("signalfx_team.team_with_detector", "name", "Team with detector"),
				),
			},
		},
	})
}

func TestAccTeamWithDashboardGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: teamWithDashboardGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTeamResourceExists,
					resource.TestCheckResourceAttr("signalfx_team.team_with_dashboard_group", "name", "Team with dashboard_group"),
				),
			},
		},
	})
}

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
		case "signalfx_detector":
		case "signalfx_dashboard_group":
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
		case "signalfx_detector":
		case "signalfx_dashboard_group":
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
