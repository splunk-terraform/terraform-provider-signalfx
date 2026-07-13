// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const newIntegrationBigPandaConfig = `
resource "signalfx_big_panda_integration" "big_panda_myresXX" {
    name = "BigPanda #1"
    enabled = false
    app_key = "app-key"
    token = "token"
}
`

const updatedIntegrationBigPandaConfig = `
resource "signalfx_big_panda_integration" "big_panda_myresXX" {
    name = "BigPanda #222"
    enabled = false
    app_key = "app-key"
    token = "token"
    alert_triggered_payload_template = "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}"
    alert_resolved_payload_template = "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}"
}`

func TestAccCreateUpdateIntegrationBigPanda(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCreateCheckDestroyIntegrationResource("signalfx_big_panda_integration"),
		Steps: []resource.TestStep{
			// Create it without custom payload templates. This verifies existing/basic BigPanda
			// integrations continue to work and the new fields are truly optional.
			{
				Config: newIntegrationBigPandaConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCreateCheckIntegrationResource("signalfx_big_panda_integration"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "name", "BigPanda #1"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "app_key", "app-key"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "token", "token"),
				),
			},
			{
				ResourceName:      "signalfx_big_panda_integration.big_panda_myresXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_big_panda_integration.big_panda_myresXX"),
				ImportStateVerify: true,
				// The API doesn't return these sensitive values, so ignore them.
				ImportStateVerifyIgnore: []string{"app_key", "token"},
			},
			// Update it with custom payload templates.
			{
				Config: updatedIntegrationBigPandaConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCreateCheckIntegrationResource("signalfx_big_panda_integration"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "name", "BigPanda #222"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "app_key", "app-key"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "token", "token"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "alert_triggered_payload_template", "{\"status\":\"critical\",\"summary\":\"{{{messageTitle}}}\"}"),
					resource.TestCheckResourceAttr("signalfx_big_panda_integration.big_panda_myresXX", "alert_resolved_payload_template", "{\"status\":\"ok\",\"summary\":\"{{{messageTitle}}}\"}"),
				),
			},
		},
	})
}
