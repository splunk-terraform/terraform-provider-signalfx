package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	teamA = `
resource "signalfx_team" "a" {
    name = "Test team A"
    description = "Terraform test"
}
`
	detectorA = `
resource "signalfx_detector" "a" {
    name = "Test team detector a"
    description = "Terraform test"
    max_delay = 30

    program_text = <<-EOF
        signal = data('app.delay').max().publish('app delay')
        detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
	EOF
    rule {
        description = "maximum > 60 for 30m"
        severity = "Critical"
        detect_label = "Processing old messages 30m"
        notifications = ["Email,noreply@signalfx.com"]
    }
}
`
	detectorB = `
resource "signalfx_detector" "b" {
    name = "Test team detector b"
    description = "Terraform test"
    max_delay = 30

    program_text = <<-EOF
        signal = data('app.delay').max().publish('app delay')
        detect(when(signal > 60, '30m')).publish('Processing old messages 30m')
	EOF
    rule {
        description = "maximum > 60 for 30m"
        severity = "Critical"
        detect_label = "Processing old messages 30m"
        notifications = ["Email,noreply@signalfx.com"]
    }
}
`
	dashboardGroupA = `
resource "signalfx_dashboard_group" "a" {
    name = "Test team dashboard group a"
    description = "Terraform test"
}
`
	dashboardGroupB = `
resource "signalfx_dashboard_group" "b" {
    name = "Test team dashboard group b"
    description = "Terraform test"
}
`
	linkDetectorAtoTeamA = `
resource "signalfx_team_links" "link1" {
	team = signalfx_team.a.id
	detectors = [signalfx_detector.a.id]
}
`
	linkDetectorABtoTeamA = `
resource "signalfx_team_links" "link2" {
	team = signalfx_team.a.id
	detectors = [signalfx_detector.a.id, signalfx_detector.b.id]
}
`
	linkDetectorBtoTeamA = `
resource "signalfx_team_links" "link2" {
	team = signalfx_team.a.id
	detectors = [signalfx_detector.b.id]
}
`
	linkDashboardGroupAtoTeamA = `
resource "signalfx_team_links" "link1" {
	team = signalfx_team.a.id
	dashboard_groups = [signalfx_dashboard_group.a.id]
}
`
	linkDashboardGroupBtoTeamA = `
resource "signalfx_team_links" "link2" {
	team = signalfx_team.a.id
	dashboard_groups = [signalfx_dashboard_group.b.id]
}
`
	linkDashboardGroupABtoTeamA = `
resource "signalfx_team_links" "link2" {
	team = signalfx_team.a.id
	dashboard_groups = [signalfx_dashboard_group.a.id, signalfx_dashboard_group.b.id]
}
`
)

func TestLinkingModifyDetectorLink(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create link and make sure team reflects it being linked.
				Config: teamA + detectorA + linkDetectorAtoTeamA,
				Check:  testAccCheckDetectorTeamLink("a", "a", true),
			},
			{
				// Add link and make sure team reflects it being linked.
				Config: teamA + detectorA + detectorB + linkDetectorABtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorTeamLink("a", "a", true),
					testAccCheckDetectorTeamLink("a", "b", true),
				),
			},
			{
				// Remove a single link.
				Config: teamA + detectorA + detectorB + linkDetectorBtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorTeamLink("a", "a", false),
					testAccCheckDetectorTeamLink("a", "b", true),
				),
			},
		},
	})
}

func TestLinkingModifyDashboardGroupsLink(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create link and make sure team reflects it being linked.
				Config: teamA + dashboardGroupA + linkDashboardGroupAtoTeamA,
				Check:  testAccCheckDashboardGroupsTeamLink("a", "a", true),
			},
			{
				// Add link and make sure team reflects it being linked.
				Config: teamA + dashboardGroupA + dashboardGroupB + linkDashboardGroupABtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupsTeamLink("a", "a", true),
					testAccCheckDashboardGroupsTeamLink("a", "b", true),
				),
			},
			{
				// Remove a single link.
				Config: teamA + dashboardGroupA + dashboardGroupB + linkDashboardGroupBtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupsTeamLink("a", "a", false),
					testAccCheckDashboardGroupsTeamLink("a", "b", true),
				),
			},
		},
	})
}

func TestLinkingDetector(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create link and make sure team reflects it being linked.
				Config: teamA + detectorA + linkDetectorAtoTeamA,
				Check:  testAccCheckDetectorTeamLink("a", "a", true),
			},
			{
				// Remove link and make sure team reflects it being unlinked.
				Config: teamA + detectorA,
				Check:  testAccCheckDetectorTeamLink("a", "a", false),
			},
		},
	})
}

func TestLinkingDashboardGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create link and make sure team reflects it being linked.
				Config: teamA + dashboardGroupA + linkDashboardGroupAtoTeamA,
				Check:  testAccCheckDashboardGroupsTeamLink("a", "a", true),
			},
			{
				// Remove link and make sure team reflects it being unlinked.
				Config: teamA + dashboardGroupA,
				Check:  testAccCheckDashboardGroupsTeamLink("a", "a", false),
			},
		},
	})
}

func TestLinkingToTeamWithUnrelatedDetectors(t *testing.T) {
	// This ensures that if two different link resources link to the same team that
	// modifying the links don't impact one another.
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create link and make sure team reflects it being linked.
				Config: teamA + detectorA + detectorB + linkDetectorAtoTeamA + linkDetectorBtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorTeamLink("a", "a", true),
					testAccCheckDetectorTeamLink("a", "b", true),
				),
			},
			{
				// Make sure the A link stays.
				Config: teamA + detectorA + detectorB + linkDetectorAtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDetectorTeamLink("a", "a", true),
					testAccCheckDetectorTeamLink("a", "b", false),
				),
			},
		},
	})
}

func TestLinkingToTeamWithUnrelatedDashboardGroups(t *testing.T) {
	// This ensures that if two different link resources link to the same team that
	// modifying the links don't impact one another.
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				// Create link and make sure team reflects it being linked.
				Config: teamA + dashboardGroupA + dashboardGroupB + linkDashboardGroupAtoTeamA + linkDashboardGroupBtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupsTeamLink("a", "a", true),
					testAccCheckDashboardGroupsTeamLink("a", "b", true),
				),
			},
			{
				// Make sure the A link stays.
				Config: teamA + dashboardGroupA + dashboardGroupB + linkDashboardGroupAtoTeamA,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDashboardGroupsTeamLink("a", "a", true),
					testAccCheckDashboardGroupsTeamLink("a", "b", false),
				),
			},
		},
	})
}

func testAccCheckDetectorTeamLink(teamName string, detectorName string, wantLink bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := newTestClient()

		rs := state.RootModule().Resources
		teamResource, ok := rs["signalfx_team."+teamName]
		if !ok {
			return fmt.Errorf("no team named %v", teamName)
		}
		detectorResource, ok := rs["signalfx_detector."+detectorName]
		if !ok {
			return fmt.Errorf("no detector named %v", detectorName)
		}

		teamId := teamResource.Primary.ID
		detectorId := detectorResource.Primary.ID

		team, err := client.GetTeam(context.TODO(), teamId)
		if err != nil {
			return err
		}

		if wantLink {
			for _, d := range team.Detectors {
				if d == detectorId {
					return nil
				}
			}
			return fmt.Errorf("did not find detector id %v in team %v", detectorId, teamId)
		}

		for _, d := range team.Detectors {
			if d == detectorId {
				return fmt.Errorf("found detector id %v in team %v", detectorId, teamId)
			}
		}
		return nil
	}
}

func testAccCheckDashboardGroupsTeamLink(teamName string, dashboardGroupName string, wantLink bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		client := newTestClient()

		rs := state.RootModule().Resources
		teamResource, ok := rs["signalfx_team."+teamName]
		if !ok {
			return fmt.Errorf("no team named %v", teamName)
		}
		dashboardGroupResource, ok := rs["signalfx_dashboard_group."+dashboardGroupName]
		if !ok {
			return fmt.Errorf("no dashboard_group named %v", dashboardGroupName)
		}

		teamId := teamResource.Primary.ID
		dashboardGroupId := dashboardGroupResource.Primary.ID

		team, err := client.GetTeam(context.TODO(), teamId)
		if err != nil {
			return err
		}

		if wantLink {
			for _, d := range team.DashboardGroups {
				if d == dashboardGroupId {
					return nil
				}
			}
			return fmt.Errorf("did not find dashboard_group id %v in team %v", dashboardGroupId, teamId)
		}

		for _, d := range team.DashboardGroups {
			if d == dashboardGroupId {
				return fmt.Errorf("found dashboard_group id %v in team %v", dashboardGroupId, teamId)
			}
		}
		return nil
	}
}
