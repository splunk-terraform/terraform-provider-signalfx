package signalfx

import (
	"bytes"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"os"
	"regexp"
	"testing"
	"text/template"
)

const myInt = `
resource "signalfx_aws_token_integration" "aws_myteam_token" {
  name = "My AWS integration (tf dev)"
}

resource "signalfx_aws_integration" "aws_myteam" {
  enabled = {{.Enabled}}

  integration_id     = signalfx_aws_token_integration.aws_myteam_token.id
  token              = "{{.AccessKeyID}}"
  key			     = "{{.SecretAccessKey}}"
  regions            = ["ap-south-1"]
  services           = ["AWS/Lambda"]  
  poll_rate          = 300
  import_cloud_watch = false
  enable_aws_usage   = false

  {{if .IgnoreCancelFailures}}ignore_cancellation_failure = {{.IgnoreCancelFailures}}{{end}}
 
  {{if .MetricStreams}}metric_streams_sync_state   = "{{.MetricStreams}}"{{end}}
}
`

type AwsIntegration struct {
	Enabled bool

	AccessKeyID     string
	SecretAccessKey string

	IgnoreCancelFailures *bool

	MetricStreams *string
}

func (integration AwsIntegration) withCreds() AwsIntegration {
	integration.AccessKeyID = os.Getenv("SFX_TEST_AWS_ACCESS_KEY_ID")
	integration.SecretAccessKey = os.Getenv("SFX_TEST_AWS_SECRET_ACCESS_KEY")
	return integration
}

func (integration AwsIntegration) withIgnoreCancelFailures(ignore bool) AwsIntegration {
	integration.IgnoreCancelFailures = &ignore
	return integration
}

func (integration AwsIntegration) withMetricStreams(state string) AwsIntegration {
	integration.MetricStreams = &state
	return integration
}

func TestAccRejectCreatingOfDisabledIntegrationWithEnabledMetricStreams(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAWSDestroy,

		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile("metric_streams_sync_state must be disabled"),
				Config:      fromTmpl(myInt, AwsIntegration{Enabled: false}.withCreds().withMetricStreams("enabled")),
			},
		},
	})
}

func TestAccShouldUpdateIgnoreCancelFailureInState(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAWSDestroy,

		Steps: []resource.TestStep{
			{
				Config: fromTmpl(myInt, AwsIntegration{Enabled: false}.withCreds()),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "ignore_cancellation_failure", "false"),
				),
			},
			{
				Config: fromTmpl(myInt, AwsIntegration{Enabled: false}.withCreds().withIgnoreCancelFailures(true)),
				Check:  resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "ignore_cancellation_failure", "true"),
			},
			{
				Config: fromTmpl(myInt, AwsIntegration{Enabled: false}.withCreds().withIgnoreCancelFailures(false)),
				Check:  resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "ignore_cancellation_failure", "false"),
			},
		},
	})
}

func TestAccRequireDisabledMetricStreamsWhenDisablingIntegration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAWSDestroy,

		Steps: []resource.TestStep{
			{
				Config: fromTmpl(myInt, AwsIntegration{Enabled: true}.withCreds().withMetricStreams("enabled")),
				Check:  resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "enabled", "true"),
			},
			{
				ExpectError: regexp.MustCompile("metric_streams_sync_state must be disabled"),
				Config:      fromTmpl(myInt, AwsIntegration{Enabled: false}.withCreds().withMetricStreams("enabled")),
			},
		},
	})
}

func TestAccMetricStreamsSwitch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccIntegrationAWSDestroy,

		Steps: []resource.TestStep{
			// Create enabled
			{
				Config: fromTmpl(myInt, AwsIntegration{Enabled: true}.withCreds().withMetricStreams("enabled")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "metric_streams_sync_state", "enabled"),
				),
			},
			// Disable
			{
				Config: fromTmpl(myInt, AwsIntegration{Enabled: true}.withCreds().withMetricStreams("disabled")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "metric_streams_sync_state", "disabled"),
				),
			},
			// Enable
			{
				Config: fromTmpl(myInt, AwsIntegration{Enabled: true}.withCreds().withMetricStreams("enabled")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIntegrationAWSResourceExists,
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "enabled", "true"),
					resource.TestCheckResourceAttr("signalfx_aws_integration.aws_myteam", "metric_streams_sync_state", "enabled"),
				),
			},
		},
	})
}

func fromTmpl(tmpl string, params AwsIntegration) string {
	t := template.Must(template.New("letter").Parse(tmpl))

	var buf bytes.Buffer

	err := t.Execute(&buf, params)

	if err != nil {
		panic(err)
	}

	return buf.String()
}
