package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestNotification(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		val    any
		expect diag.Diagnostics
	}{
		{
			name: "no value provided",
			val:  nil,
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected <nil> to be of type string"},
			},
		},
		{
			name: "incomplete notification string",
			val:  "notification",
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: `invalid notification string "notification", not enough commas`},
			},
		},
		{
			name:   "amazon event bridge",
			val:    "AmazonEventBridge,....",
			expect: nil,
		},
		{
			name:   "big panda",
			val:    "BigPanda,",
			expect: nil,
		},
		{
			name:   "Jira",
			val:    "Jira,",
			expect: nil,
		},
		{
			name:   "Office 365",
			val:    "Office365,",
			expect: nil,
		},
		{
			name:   "PagerDuty",
			val:    "PagerDuty,",
			expect: nil,
		},
		{
			name:   "Microsoft Teams",
			val:    "Team,",
			expect: nil,
		},
		{
			name:   "Microsoft Teams Email",
			val:    "TeamEmail,",
			expect: nil,
		},
		{
			name:   "XMatters",
			val:    "XMatters,",
			expect: nil,
		},
		{
			name:   "email",
			val:    "Email,example@example",
			expect: nil,
		},
		{
			name:   "email pseudo email",
			val:    "Email,example+alert@example",
			expect: nil,
		},
		{
			name: "email invalid",
			val:  "Email,derp",
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "mail: missing '@' or angle-addr"},
			},
		},
		{
			name:   "Opsgenie",
			val:    "Opsgenie,,,,",
			expect: nil,
		},
		{
			name: "Opsgenie invalid",
			val:  "Opsgenie,",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "invalid OpsGenie notification string, please consult the documentation (not enough parts)",
				},
			},
		},
		{
			name:   "Slack Notification",
			val:    "Slack,cool-slack,my-channel",
			expect: nil,
		},
		{
			name: "Slack invalid",
			val:  "Slack,",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "invalid Slack notification string, please consult the documentation (not enough parts)",
				},
			},
		},
		{
			name: "slack syntax error",
			val:  "Slack,,#channel",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  `exclude the # from channel names in "#channel"`,
				},
			},
		},
		{
			name:   "VictorOps",
			val:    "VictorOps,,",
			expect: nil,
		},
		{
			name: "VictorOps invalid",
			val:  "VictorOps,",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "invalid VictorOps notification string, please consult the documentation (not enough parts)",
				},
			},
		},
		{
			name:   "Webhook using creditial id",
			val:    "Webhook,Aaaaaa,,",
			expect: nil,
		},
		{
			name:   "Webhook using secret and URL",
			val:    "Webhook,,verysercretsecret,http://example.com",
			expect: nil,
		},
		{
			name: "Webhook not enough parts",
			val:  "Webhook,,",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "invalid Webhook notification string, please consult the documentation (not enough parts)",
				},
			},
		},
		{
			name: "Webhook no values set",
			val:  "Webhook,,,",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)",
				},
			},
		},
		{
			name: "Webhook set all values",
			val:  "Webhook,aaa,mysecret,http://example.com",
			expect: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "invalid Webhook notification string, please consult the documentation (use one of URL and secret or credential id)",
				},
			},
		},
		{
			name: "Webhook url invalid",
			val:  "Webhook,,verysercretsecret,foo",
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "invalid Webhook URL \"verysercretsecret\""},
			},
		},
		{
			name: "Unknown notification",
			val:  "alertatron,",
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "invalid notification type \"alertatron\""},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := Notification()(tc.val, cty.Path{})
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
		})
	}
}
