package signalfx

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"

	"github.com/stretchr/testify/assert"
)

const newIntegrationAWSConfig = `
  resource "signalfx_aws_external_integration" "aws_ext_myteamXX" {
	name = "AWS TF Test (ext/new)"
  }

  resource "signalfx_aws_integration" "aws_myteamXX" {
	enabled = false

	integration_id     = signalfx_aws_external_integration.aws_ext_myteamXX.id
	external_id        = signalfx_aws_external_integration.aws_ext_myteamXX.external_id
	role_arn           = "arn:aws:iam::XXX:role/SignalFx-Read-Role"
	regions            = ["us-east-1"]
	poll_rate          = 300
	import_cloud_watch = true
	enable_aws_usage   = true

	custom_namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "fart"
	}

	custom_namespace_sync_rule {
	  namespace = "custom"
	}

	namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "AWS/EC2"
	}
  }

  resource "signalfx_aws_token_integration" "aws_tok_myteamXX" {
	name = "AWS TF Test (token/new)"
  }

  resource "signalfx_aws_integration" "aws_myteam_tokXX" {
	enabled = false

	integration_id             = signalfx_aws_token_integration.aws_tok_myteamXX.id
	token                      = "token123"
	key                        = "key123"
	regions                    = ["us-east-1"]
	poll_rate                  = 300
	import_cloud_watch         = true
	enable_aws_usage           = true
	use_get_metric_data_method = true

	custom_namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "fart"
	}

	custom_namespace_sync_rule {
	  namespace = "custom"
	}

	namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "AWS/EC2"
	}
  }
`

const updatedIntegrationAWSConfig = `
  resource "signalfx_aws_external_integration" "aws_ext_myteamXX" {
	name = "AWS TF Test (ext/updated)"
  }

  resource "signalfx_aws_integration" "aws_myteamXX" {
	enabled = false

	integration_id     = signalfx_aws_external_integration.aws_ext_myteamXX.id
	external_id        = signalfx_aws_external_integration.aws_ext_myteamXX.external_id
	role_arn           = "arn:aws:iam::XXX:role/SignalFx-Read-Role"
	regions            = ["us-east-1"]
	poll_rate          = 300
	import_cloud_watch = true
	enable_aws_usage   = true

	custom_namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "fart"
	}

	custom_namespace_sync_rule {
	  namespace = "custom"
	}

	namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "AWS/EC2"
	}
  }

  resource "signalfx_aws_token_integration" "aws_tok_myteamXX" {
	name = "AWS TF Test (token/updated)"
  }

  resource "signalfx_aws_integration" "aws_myteam_tokXX" {
	enabled = false

	integration_id             = signalfx_aws_token_integration.aws_tok_myteamXX.id
	token                      = "token123"
	key                        = "key123"
	regions                    = ["us-east-1"]
	poll_rate                  = 300
	import_cloud_watch         = true
	enable_aws_usage           = true
	use_get_metric_data_method = true

	custom_namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "fart"
	}

	custom_namespace_sync_rule {
	  namespace = "custom"
	}

	namespace_sync_rule {
	  default_action = "Exclude"
	  filter_action  = "Include"
	  filter_source  = "filter('code', '200')"
	  namespace      = "AWS/EC2"
	}
  }
`

const updatedIntegrationAWSConfigMetricStreams = `
  resource "signalfx_aws_token_integration" "aws_tok_myteamXX" {
	name = "AWS TF Test (token/updated/ms:%s)"
  }

  resource "signalfx_aws_integration" "aws_myteam_tokXX" {
	enabled = true # This is required to be able to cancel AWS Metric Streams synchronization.

	integration_id          = signalfx_aws_token_integration.aws_tok_myteamXX.id
	token                   = "%s"
	key                     = "%s"
	regions                 = ["us-east-1"]
	services                = ["AWS/Lambda"]
	poll_rate               = 300
	import_cloud_watch      = true
	use_metric_streams_sync = %s
  }
`

const updatedIntegrationAWSConfigLogsSync = `
  resource "signalfx_aws_token_integration" "aws_tok_myteamXX" {
	name = "AWS TF Test (token/updated/logs:%s)"
  }

  resource "signalfx_aws_integration" "aws_myteam_tokXX" {
	enabled = true # This is required to be able to cancel AWS Metric Streams synchronization.

	integration_id          = signalfx_aws_token_integration.aws_tok_myteamXX.id
	token                   = "%s"
	key                     = "%s"
	regions                 = ["eu-north-1"]
	services                = ["AWS/Lambda"]
	poll_rate               = 300
	import_cloud_watch      = true
	enable_logs_sync        = %s
  }
`

func TestAccCreateUpdateIntegrationAWS(t *testing.T) {
	awsAccessKeyID := os.Getenv("SFX_TEST_AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("SFX_TEST_AWS_SECRET_ACCESS_KEY")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAWSDestroy,
		Steps: []resource.TestStep{
			// Create
			{
				Config: newIntegrationAWSConfig,
				Check:  testAccCheckIntegrationAWSResourceExists,
			},
			// Update
			{
				Config: updatedIntegrationAWSConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteamXX", "name", "AWS TF Test (ext/updated)"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "name", "AWS TF Test (token/updated)"),
				),
			},
			// Update again to enable Cloudwatch Metric Streams synchronization
			{
				SkipFunc: skipTestWhenAWSCredentialsAreMissing(t, awsAccessKeyID, awsSecretAccessKey),
				Config:   fmt.Sprintf(updatedIntegrationAWSConfigMetricStreams, "enabled", awsAccessKeyID, awsSecretAccessKey, "true"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "name", "AWS TF Test (token/updated/ms:enabled)"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "use_metric_streams_sync", "true"),
				),
			},
			// Update again to disable Cloudwatch Metric Streams synchronization
			{
				SkipFunc: skipTestWhenAWSCredentialsAreMissing(t, awsAccessKeyID, awsSecretAccessKey),
				Config:   fmt.Sprintf(updatedIntegrationAWSConfigMetricStreams, "disabled", awsAccessKeyID, awsSecretAccessKey, "false"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "name", "AWS TF Test (token/updated/ms:disabled)"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "use_metric_streams_sync", "false"),
				),
			},
			// Update again to enable AWS logs synchronization
			{
				SkipFunc: skipTestWhenAWSCredentialsAreMissing(t, awsAccessKeyID, awsSecretAccessKey),
				Config:   fmt.Sprintf(updatedIntegrationAWSConfigLogsSync, "enabled", awsAccessKeyID, awsSecretAccessKey, "true"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "name", "AWS TF Test (token/updated/logs:enabled)"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "enable_logs_sync", "true"),
				),
			},
			// Update again to disable AWS logs synchronization
			{
				SkipFunc: skipTestWhenAWSCredentialsAreMissing(t, awsAccessKeyID, awsSecretAccessKey),
				Config:   fmt.Sprintf(updatedIntegrationAWSConfigLogsSync, "disabled", awsAccessKeyID, awsSecretAccessKey, "false"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "name", "AWS TF Test (token/updated/logs:disabled)"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "enable_logs_sync", "false"),
				),
			},
		},
	})
}

func TestAccCreateDeleteIntegrationAWSMetricStream(t *testing.T) {
	awsAccessKeyID := os.Getenv("SFX_TEST_AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("SFX_TEST_AWS_SECRET_ACCESS_KEY")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAWSDestroy,
		Steps: []resource.TestStep{
			// Create integration with Cloudwatch Metric Streams synchronization enabled without any additional step to disable it before deletion. That should automatically be done in the delete phase.
			{
				SkipFunc: skipTestWhenAWSCredentialsAreMissing(t, awsAccessKeyID, awsSecretAccessKey),
				Config:   fmt.Sprintf(updatedIntegrationAWSConfigMetricStreams, "enabled", awsAccessKeyID, awsSecretAccessKey, "true"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "name", "AWS TF Test (token/updated/ms:enabled)"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam_tokXX", "use_metric_streams_sync", "true"),
				),
			},
		},
	})
}

func testAccCheckIntegrationAWSResourceExists(s *terraform.State) error {
	client := newTestClient()

	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_aws_integration", "signalfx_aws_external_integration", "signalfx_aws_token_integration":
			integration, err := client.GetAWSCloudWatchIntegration(context.TODO(), rs.Primary.ID)
			if integration == nil {
				return fmt.Errorf("Error finding integration %s: %s", rs.Primary.ID, err)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func testAccIntegrationAWSDestroy(s *terraform.State) error {
	client := newTestClient()
	for _, rs := range s.RootModule().Resources {
		switch rs.Type {
		case "signalfx_aws_integration", "signalfx_aws_external_integration", "signalfx_aws_token_integration":
			integration, _ := client.GetAWSCloudWatchIntegration(context.TODO(), rs.Primary.ID)
			if integration != nil {
				return fmt.Errorf("Found deleted integration %s", rs.Primary.ID)
			}
		default:
			return fmt.Errorf("Unexpected resource of type: %s", rs.Type)
		}
	}

	return nil
}

func TestValidateAwsService(t *testing.T) {
	_, errors := validateAwsService("AWS/Logs", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateAwsService("Fart", "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}

func TestValidateFilterAction(t *testing.T) {
	_, errors := validateFilterAction("Exclude", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateFilterAction("Include", "")
	assert.Equal(t, 0, len(errors), "No errors for valid value")

	_, errors = validateFilterAction("Fart", "")
	assert.Equal(t, 1, len(errors), "Errors for invalid value")
}

func skipTestWhenAWSCredentialsAreMissing(t *testing.T, awsAccessKeyID, awsSecretAccessKey string) func() (bool, error) {
	return func() (bool, error) {
		if awsAccessKeyID != "" && awsSecretAccessKey != "" {
			return false, nil
		}
		t.Log("Skipping step: Env vars SFX_TEST_AWS_ACCESS_KEY_ID and SFX_TEST_AWS_SECRET_ACCESS_KEY must be set to " +
			"test AWS CloudWatch Metric Streams synchronization.")
		return true, nil
	}
}
