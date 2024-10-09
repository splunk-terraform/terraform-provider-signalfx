// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const newIntegrationServiceNowConfig = `
resource "signalfx_service_now_integration" "snow_myresXX" {
    name = "SNOW #1"
    enabled = false
    username = "thisis_me"
    password = "youd0ntsee1t"
    instance_name = "myinst.service-now.com"
    issue_type = "Incident"
}
`

const updatedIntegrationServiceNowConfig = `
resource "signalfx_service_now_integration" "snow_myresXX" {
    name = "SNOW #222"
    enabled = false
    username = "thisis_me"
    password = "youd0ntsee1t"
    instance_name = "myinst.service-now.com"
    issue_type = "Problem"
    alert_triggered_payload_template = "{\"short_description\": \"{{{messageTitle}}} (customized)\"}"
    alert_resolved_payload_template = "{\"close_notes\": \"{{{messageTitle}}} (customized close msg)\"}"
}`

func TestAccCreateUpdateIntegrationServiceNow(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCreateCheckDestroyIntegrationResource("signalfx_service_now_integration"),
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newIntegrationServiceNowConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCreateCheckIntegrationResource("signalfx_service_now_integration"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "name", "SNOW #1"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "username", "thisis_me"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "password", "youd0ntsee1t"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "instance_name", "myinst.service-now.com"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "issue_type", "Incident"),
				),
			},
			{
				ResourceName:      "signalfx_service_now_integration.snow_myresXX",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_service_now_integration.snow_myresXX"),
				ImportStateVerify: true,
				// The API doesn't return this value, so blow it up
				ImportStateVerifyIgnore: []string{"username", "password"},
			},
			// Update It
			{
				Config: updatedIntegrationServiceNowConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCreateCheckIntegrationResource("signalfx_service_now_integration"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "name", "SNOW #222"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "enabled", "false"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "username", "thisis_me"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "password", "youd0ntsee1t"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "instance_name", "myinst.service-now.com"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "issue_type", "Problem"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "alert_triggered_payload_template", "{\"short_description\": \"{{{messageTitle}}} (customized)\"}"),
					resource.TestCheckResourceAttr("signalfx_service_now_integration.snow_myresXX", "alert_resolved_payload_template", "{\"close_notes\": \"{{{messageTitle}}} (customized close msg)\"}"),
				),
			},
		},
	})
}
