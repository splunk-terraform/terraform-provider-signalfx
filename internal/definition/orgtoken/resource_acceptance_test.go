// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package orgtoken_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/orgtoken"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestResourceAcceptance(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []resource.TestStep
	}{
		{
			name: "minimal org token",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/minimal.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_org_token.minimal", "name", "my-token"),
						resource.TestCheckResourceAttr("signalfx_org_token.minimal", "description", "This is my token"),
						resource.TestCheckResourceAttr("signalfx_org_token.minimal", "auth_scopes.#", "1"),
						resource.TestCheckResourceAttr("signalfx_org_token.minimal", "auth_scopes.0", "API"),
						resource.TestCheckResourceAttr("signalfx_org_token.minimal", "disabled", "false"),
					),
					ExpectNonEmptyPlan: false,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tftest.NewAcceptanceHandler(
				tftest.WithAcceptanceResources(map[string]*schema.Resource{
					orgtoken.ResourceName: orgtoken.NewResource(),
				}),
			).Test(t, tc.steps)
		})
	}
}
