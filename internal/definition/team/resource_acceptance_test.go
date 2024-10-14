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
		opts  []tftest.AcceptanceHandlerOption
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := []tftest.AcceptanceHandlerOption{
				tftest.WithAcceptanceResources(map[string]*schema.Resource{
					ResourceName: NewResource(),
				}),
			}

			tftest.NewAcceptanceHandler(append(base, tc.opts...)).
				Test(t, tc.steps)
		})
	}
}
