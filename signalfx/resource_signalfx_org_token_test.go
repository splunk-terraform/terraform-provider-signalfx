// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const newOrgTokenConfig = `
resource "signalfx_org_token" "myorgtokenTOK1" {
  name = "FarToken"
  description = "Farts"
	notifications = ["Email,foo-alerts@example.com"]

  host_or_usage_limits {
    host_limit = 100
    host_notification_threshold = 90
    container_limit = 200
    container_notification_threshold = 180
    custom_metrics_limit = 1000
    custom_metrics_notification_threshold = 900
    high_res_metrics_limit = 1000
    high_res_metrics_notification_threshold = 900
  }
}
`

const updatedOrgTokenConfig = `
resource "signalfx_org_token" "myorgtokenTOK1" {
  name = "FarToken"
  description = "Farts NEW"
	notifications = ["Email,foo-alerts@example.com"]

  host_or_usage_limits {
    host_limit = 100
    host_notification_threshold = 90
    container_limit = 200
    container_notification_threshold = 180
    custom_metrics_limit = 1000
    custom_metrics_notification_threshold = 900
    high_res_metrics_limit = 1000
    high_res_metrics_notification_threshold = 900
  }
}
`

const newOrgTokenLimitConfig = `
resource "signalfx_org_token" "mylimitorgtokenTOK1" {
  name = "LimitToken"
  description = "Limits"
  auth_scopes = ["INGEST"]

  dpm_limits {
    dpm_limit = 1000
  }
}
`

const updatedOrgTokenLimitConfig = `
resource "signalfx_org_token" "mylimitorgtokenTOK1" {
  name = "LimitToken"
  description = "Limits NEW"
  auth_scopes = ["INGEST"]

  dpm_limits {
    dpm_limit = 2000
  }
}
`

const orgTokenSecureConfig = `
resource "signalfx_org_token" "secure_token" {
  name = "SecureToken"
  description = "Token with no secret storage"
  store_secret = false
  auth_scopes = ["INGEST", "API"]
  disabled = false
}
`

const orgTokenSecureWithLimitsConfig = `
resource "signalfx_org_token" "secure_token_limits" {
  name = "SecureTokenWithLimits"
  description = "Token with limits and no secret storage"
  store_secret = false
  auth_scopes = ["INGEST"]
  disabled = false
  
  host_or_usage_limits {
    host_limit = 100
    host_notification_threshold = 80
    container_limit = 200
    container_notification_threshold = 150
    custom_metrics_limit = 1000
    custom_metrics_notification_threshold = 800
  }
}
`

const orgTokenSecureWithDPMLimitsConfig = `
resource "signalfx_org_token" "secure_token_dpm" {
  name = "SecureTokenWithDPM"
  description = "Token with DPM limits and no secret storage"
  store_secret = false
  auth_scopes = ["INGEST"]
  
  dpm_limits {
    dpm_limit = 5000
    dpm_notification_threshold = 4000
  }
}
`

const orgTokenSecureUpdatedConfig = `
resource "signalfx_org_token" "secure_token" {
  name = "SecureToken"
  description = "Token with updated description and no secret storage"
  store_secret = false
  auth_scopes = ["INGEST", "API", "RUM"]
  disabled = true
}
`

const orgTokenBackwardCompatConfig = `
resource "signalfx_org_token" "backward_compat_token" {
  name = "BackwardCompatToken"
  description = "Token with default behavior"
  auth_scopes = ["INGEST"]
}
`

const orgTokenExplicitStoreConfig = `
resource "signalfx_org_token" "explicit_store_token" {
  name = "ExplicitStoreToken"
  description = "Token with explicit secret storage"
  store_secret = true
  auth_scopes = ["INGEST", "API"]
}
`

func TestAccOrgTokenSecureStorage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: orgTokenSecureConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "name", "SecureToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "description", "Token with no secret storage"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "store_secret", "false"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "auth_scopes.#", "2"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "auth_scopes.0", "API"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "auth_scopes.1", "INGEST"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "secret", ""),
				),
			},
			{
				ResourceName:      "signalfx_org_token.secure_token",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_org_token.secure_token"),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOrgTokenSecureStorageUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: orgTokenSecureConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "description", "Token with no secret storage"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "disabled", "false"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "secret", ""),
				),
			},
			{
				Config: orgTokenSecureUpdatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "description", "Token with updated description and no secret storage"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "disabled", "true"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "auth_scopes.#", "3"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token", "secret", ""),
				),
			},
		},
	})
}

func TestAccOrgTokenBackwardCompatibility(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: orgTokenBackwardCompatConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.backward_compat_token", "name", "BackwardCompatToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.backward_compat_token", "description", "Token with default behavior"),
					resource.TestCheckResourceAttr("signalfx_org_token.backward_compat_token", "store_secret", "true"),
					resource.TestCheckResourceAttrSet("signalfx_org_token.backward_compat_token", "secret"),
				),
			},
			{
				ResourceName:      "signalfx_org_token.backward_compat_token",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_org_token.backward_compat_token"),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOrgTokenSecureStorageWithDPMLimits(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: orgTokenSecureWithDPMLimitsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_dpm", "name", "SecureTokenWithDPM"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_dpm", "store_secret", "false"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_dpm", "dpm_limits.#", "1"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_dpm", "dpm_limits.0.dpm_limit", "5000"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_dpm", "dpm_limits.0.dpm_notification_threshold", "4000"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_dpm", "secret", ""),
				),
			},
		},
	})
}

func TestAccOrgTokenSecureStorageWithLimits(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: orgTokenSecureWithLimitsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "name", "SecureTokenWithLimits"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "store_secret", "false"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "host_or_usage_limits.#", "1"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "host_or_usage_limits.0.host_limit", "100"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "host_or_usage_limits.0.host_notification_threshold", "80"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "host_or_usage_limits.0.container_limit", "200"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "host_or_usage_limits.0.container_notification_threshold", "150"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "host_or_usage_limits.0.custom_metrics_limit", "1000"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "host_or_usage_limits.0.custom_metrics_notification_threshold", "800"),
					resource.TestCheckResourceAttr("signalfx_org_token.secure_token_limits", "secret", ""),
				),
			},
		},
	})
}

func TestAccOrgTokenExplicitSecretStorage(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: orgTokenExplicitStoreConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.explicit_store_token", "name", "ExplicitStoreToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.explicit_store_token", "description", "Token with explicit secret storage"),
					resource.TestCheckResourceAttr("signalfx_org_token.explicit_store_token", "store_secret", "true"),
					resource.TestCheckResourceAttrSet("signalfx_org_token.explicit_store_token", "secret"),
				),
			},
		},
	})
}

func TestAccOrgTokenStoreSecretToggle(t *testing.T) {
	const initialConfig = `
resource "signalfx_org_token" "toggle_token" {
  name = "ToggleToken"
  description = "Token to test store_secret toggle"
  store_secret = true
  auth_scopes = ["INGEST"]
}
`

	const updatedConfig = `
resource "signalfx_org_token" "toggle_token" {
  name = "ToggleToken"
  description = "Token to test store_secret toggle"
  store_secret = false
  auth_scopes = ["INGEST"]
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.toggle_token", "store_secret", "true"),
					// secret should be stored initially
					resource.TestCheckResourceAttrSet("signalfx_org_token.toggle_token", "secret"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.toggle_token", "store_secret", "false"),
					// secret should no longer be stored after toggle
					resource.TestCheckResourceAttr("signalfx_org_token.toggle_token", "secret", ""),
				),
			},
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.toggle_token", "store_secret", "true"),
					// secret should be stored again after toggling back
					resource.TestCheckResourceAttrSet("signalfx_org_token.toggle_token", "secret"),
				),
			},
		},
	})
}

func TestAccCreateUpdateOrgToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newOrgTokenConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "name", "FarToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "description", "Farts"),
				),
			},
			{
				ResourceName:      "signalfx_org_token.myorgtokenTOK1",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_org_token.myorgtokenTOK1"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedOrgTokenConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "name", "FarToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.myorgtokenTOK1", "description", "Farts NEW"),
				),
			},
		},
	})
}

func TestAccCreateUpdateLimitOrgToken(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccOrgTokenDestroy,
		Steps: []resource.TestStep{
			// Create It
			{
				Config: newOrgTokenLimitConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.mylimitorgtokenTOK1", "name", "LimitToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.mylimitorgtokenTOK1", "description", "Limits"),
					resource.TestCheckResourceAttr("signalfx_org_token.mylimitorgtokenTOK1", "dpm_limits.#", "1"),
				),
			},
			{
				ResourceName:      "signalfx_org_token.mylimitorgtokenTOK1",
				ImportState:       true,
				ImportStateIdFunc: testAccStateIdFunc("signalfx_org_token.mylimitorgtokenTOK1"),
				ImportStateVerify: true,
			},
			// Update Everything
			{
				Config: updatedOrgTokenLimitConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgTokenResourceExists,
					resource.TestCheckResourceAttr("signalfx_org_token.mylimitorgtokenTOK1", "name", "LimitToken"),
					resource.TestCheckResourceAttr("signalfx_org_token.mylimitorgtokenTOK1", "description", "Limits NEW"),
					resource.TestCheckResourceAttr("signalfx_org_token.mylimitorgtokenTOK1", "dpm_limits.#", "1"),
				),
			},
		},
	})
}

func testAccCheckOrgTokenResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_org_token":
			tok, err := client.GetOrgToken(context.TODO(), rs.Primary.ID)
			if err != nil || tok.Name != rs.Primary.ID {
				return fmt.Errorf("Error finding org token %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}
	return nil
}

func testAccOrgTokenDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_org_token":
			tok, _ := client.GetOrgToken(context.TODO(), rs.Primary.ID)
			if tok != nil {
				return fmt.Errorf("Found deleted org token %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}
