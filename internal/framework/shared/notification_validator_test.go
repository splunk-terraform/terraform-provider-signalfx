// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestNotificationValidatorDescription(t *testing.T) {
	t.Parallel()

	validator := newNotificationValidator()

	expect := "Ensures that the notification configuration is valid."
	assert.Equal(t, expect, validator.Description(t.Context()), "Must match the expected description")
}

func TestNotificationValidatorMarkdownDescription(t *testing.T) {
	t.Parallel()

	validator := newNotificationValidator()

	expect := "Ensures that the notification configuration is valid.\n\n" +
		"The required fields for a notification depend on the value of the `type` field. " +
		"The following table outlines the required fields for each notification type:\n\n" +
		"| Notification Type | Required Fields |\n" +
		"|-------------------|-----------------|\n" +
		"| AmazonEventBridge | `credential_id` |\n" +
		"| BigPanda | `credential_id` |\n" +
		"| Email | `email` |\n" +
		"| Jira | `credential_id` |\n" +
		"| Office365 | `credential_id` |\n" +
		"| Opsgenie | `credential_id` |\n" +
		"| PagerDuty | `credential_id` |\n" +
		"| ServiceNow | `credential_id` |\n" +
		"| Slack | `credential_id`, `channel` |\n" +
		"| Team | `team` |\n" +
		"| TeamEmail | `team` |\n" +
		"| VictorOps | `credential_id`, `routing_key` |\n" +
		"| Webhook |  |\n" +
		"| XMatters | `credential_id` |\n"

	assert.Equal(t, expect, validator.MarkdownDescription(t.Context()), "Must match the expected markdown description")
}

func TestNotificationValidatorValidateObject(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		values map[string]attr.Value
		expect diag.Diagnostics
	}{
		{
			name:   "no values set",
			values: map[string]attr.Value{},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("type"),
					"Invalid Notification Type",
					"The notification type must be set to a valid value before other fields can be validated.",
				),
			},
		},
		{
			name: "Amazon Event Bridge",
			values: map[string]attr.Value{
				"type": types.StringValue("AmazonEventBridge"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=AmazonEventBridge"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=AmazonEventBridge\" is specified",
				),
			},
		},
		{
			name: "Big Panda",
			values: map[string]attr.Value{
				"type": types.StringValue("BigPanda"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=BigPanda"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=BigPanda\" is specified",
				),
			},
		},
		{
			name: "email type with required field set",
			values: map[string]attr.Value{
				"type": types.StringValue("Email"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=Email"),
					"Invalid Attribute Combination",
					"Attribute \"notification.email\" must be specified when \"notification.type=Email\" is specified",
				),
			},
		},
		{
			name: "Jira type",
			values: map[string]attr.Value{
				"type": types.StringValue("Jira"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=Jira"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=Jira\" is specified",
				),
			},
		},
		{
			name: "Opsgenie type",
			values: map[string]attr.Value{
				"type": types.StringValue("Opsgenie"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=Opsgenie"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=Opsgenie\" is specified",
				),
			},
		},
		{
			name: "Office365 type",
			values: map[string]attr.Value{
				"type": types.StringValue("Office365"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=Office365"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=Office365\" is specified",
				),
			},
		},
		{
			name: "PagerDuty",
			values: map[string]attr.Value{
				"type": types.StringValue("PagerDuty"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=PagerDuty"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=PagerDuty\" is specified",
				),
			},
		},
		{
			name: "ServiceNow",
			values: map[string]attr.Value{
				"type": types.StringValue("ServiceNow"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=ServiceNow"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=ServiceNow\" is specified",
				),
			},
		},
		{
			name: "Slack",
			values: map[string]attr.Value{
				"type": types.StringValue("Slack"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=Slack"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=Slack\" is specified",
				),
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=Slack"),
					"Invalid Attribute Combination",
					"Attribute \"notification.channel\" must be specified when \"notification.type=Slack\" is specified",
				),
			},
		},
		{
			name: "Team",
			values: map[string]attr.Value{
				"type": types.StringValue("Team"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=Team"),
					"Invalid Attribute Combination",
					"Attribute \"notification.team\" must be specified when \"notification.type=Team\" is specified",
				),
			},
		},
		{
			name: "Team Email",
			values: map[string]attr.Value{
				"type": types.StringValue("TeamEmail"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=TeamEmail"),
					"Invalid Attribute Combination",
					"Attribute \"notification.team\" must be specified when \"notification.type=TeamEmail\" is specified",
				),
			},
		},
		{
			name: "VictorOps",
			values: map[string]attr.Value{
				"type": types.StringValue("VictorOps"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=VictorOps"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=VictorOps\" is specified",
				),
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=VictorOps"),
					"Invalid Attribute Combination",
					"Attribute \"notification.routing_key\" must be specified when \"notification.type=VictorOps\" is specified",
				),
			},
		},
		{
			name: "Webhook",
			values: map[string]attr.Value{
				"type": types.StringValue("Webhook"),
			},
		},
		{
			name: "XMatters",
			values: map[string]attr.Value{
				"type": types.StringValue("XMatters"),
			},
			expect: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("notification").AtName("type=XMatters"),
					"Invalid Attribute Combination",
					"Attribute \"notification.credential_id\" must be specified when \"notification.type=XMatters\" is specified",
				),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			notificationAttr := NewNotificationResourceAttribute()
			typed := make(map[string]attr.Type)
			for field, attr := range notificationAttr.Attributes {
				typed[field] = attr.GetType()
				if _, ok := tc.values[field]; !ok {
					tc.values[field] = attr.GetType().ValueType(t.Context())
				}
			}

			configValue := types.ObjectValueMust(typed, tc.values)
			rawConfigValue, err := configValue.ToTerraformValue(t.Context())
			assert.NoError(t, err)

			var (
				req = validator.ObjectRequest{
					Path:           path.Root("notification"),
					PathExpression: path.MatchRoot("notification"),
					Config: tfsdk.Config{
						Raw: tftypes.NewValue(
							tftypes.Object{
								AttributeTypes: map[string]tftypes.Type{
									"notification": rawConfigValue.Type(),
								},
							},
							map[string]tftypes.Value{
								"notification": rawConfigValue,
							},
						),
						Schema: schema.Schema{
							Attributes: map[string]schema.Attribute{
								"notification": schema.SingleNestedAttribute{
									Required:   true,
									Attributes: notificationAttr.Attributes,
								},
							},
						},
					},
					ConfigValue: configValue,
				}
				resp = &validator.ObjectResponse{}
			)
			newNotificationValidator().ValidateObject(t.Context(), req, resp)

			assert.Equal(t, tc.expect, resp.Diagnostics, "Must match expected diagnostics")
		})
	}
}
