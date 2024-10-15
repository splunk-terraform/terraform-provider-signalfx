// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package team

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestAcceptance(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []resource.TestStep
	}{
		{
			name: "simple lifecylce",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/resource_team.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_team.example_test", "name", "my team"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "description", "An example of team"),
					),
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
				{
					Config: tftest.LoadConfig("testdata/resource_team.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_team.example_test", "name", "my team"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "description", "An example of team"),
					),
				},
				{
					Config:  tftest.LoadConfig("testdata/resource_team.tf"),
					Destroy: true,
				},
			},
		},
		{
			name: "original lifecycle",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/resource_team.tf"),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_team.example_test", "name", "my team"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "description", "An example of team"),
					),
				},
				// Update Everything
				{
					Config: tftest.LoadConfig("testdata/resource_team_updated.tf"),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_team.example_test", "name", "my team"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "description", "An example of team"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_critical.#", "1"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_critical.0", "Email,test@example.com"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_default.#", "1"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_default.0", "Webhook,,secret,https://www.example.com"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_info.#", "1"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_info.0", "Webhook,,secret,https://www.example.com/2"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_major.#", "1"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_major.0", "Webhook,,secret,https://www.example.com/3"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_minor.#", "1"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_minor.0", "Webhook,,secret,https://www.example.com/4"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_warning.#", "1"),
						resource.TestCheckResourceAttr("signalfx_team.example_test", "notifications_warning.0", "Webhook,,secret,https://www.example.com/5"),
					),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			tftest.NewAcceptanceHandler(
				tftest.WithAcceptanceResources(map[string]*schema.Resource{
					ResourceName: NewResource(),
				}),
			).
				Test(t, tc.steps)
		})
	}
}
