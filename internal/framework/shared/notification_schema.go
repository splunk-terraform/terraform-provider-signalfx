// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewNotificationResourceAttribute(opts ...func(*schema.NestedAttributeObject)) schema.NestedAttributeObject {
	attr := schema.NestedAttributeObject{
		Validators: []validator.Object{
			newNotificationValidator(),
		},
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required:    true,
				Description: "This is the name of the integration type that the notification should be routed through.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"AmazonEventBridge",
						"BigPanda",
						"Email",
						"Jira",
						"Office365",
						"Opsgenie",
						"PagerDuty",
						"ServiceNow",
						"Slack",
						"TeamEmail",
						"Team",
						"VictorOps",
						"Webhook",
						"XMatters",
					),
				},
			},
			"credential_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the credential to use for this notification.",
			},
			"email": schema.StringAttribute{
				Optional: true,
				Description: "The email address to send the notification to. " +
					"Only used for Email and TeamEmail notification types.",
			},
			"credential_name": schema.StringAttribute{
				Optional: true,
				Description: "The name of the credential to use for this notification." +
					" Only used for Team notification types.",
			},
			"responder_id": schema.StringAttribute{
				Optional: true,
				Description: "The ID of the responder to send the notification to." +
					" Only used for Team notification types.",
			},
			"responder_name": schema.StringAttribute{
				Optional: true,
				Description: "The name of the responder to send the notification to." +
					" Only used for Team notification types.",
			},
			"responder_type": schema.StringAttribute{
				Optional: true,
				Description: "The type of the responder to send the notification to." +
					" Only used for Team notification types.",
			},
			"channel": schema.StringAttribute{
				Optional: true,
				Description: "The channel to send the notification to." +
					" Only used for Slack notification types.",
			},
			"team": schema.StringAttribute{
				Optional: true,
				Description: "The team to send the notification to. " +
					"Only used for Team notification types.",
			},
			"routing_key": schema.StringAttribute{
				Optional: true,
				Description: "The routing key to use for this notification. " +
					"Only used for Splunk OnCall (formerly VictorOps) notification types.",
			},
			"secret": schema.StringAttribute{
				Optional: true,
				Description: "The secret to use for this notification. " +
					"Only used for Webhook notification types.",
			},
			"url": schema.StringAttribute{
				Optional: true,
				Description: "The URL to send the notification to. " +
					"Only used for Webhook notification types.",
			},
		},
	}
	for _, opt := range opts {
		opt(&attr)
	}
	return attr
}

func NewNotificationObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: AttributeTypeMap(NewNotificationResourceAttribute().Attributes),
	}
}
