// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package orgtoken

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/orgtoken"
	"go.uber.org/multierr"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the token",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Decription of the token",
		},
		"auth_scopes": {
			Type:        schema.TypeList,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Computed:    true,
			Description: "Authentication scope, ex: INGEST, API, RUM ... (Optional)",
		},
		"disabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Flag that controls enabling the token. If set to `true`, the token is disabled, and you can't use it for authentication. Defaults to `false`",
		},
		"secret": {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "The value of the token used for API actions.",
		},
		"notifications": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of strings specifying where notifications will be sent when an incident occurs. See https://developers.signalfx.com/v2/docs/detector-model#notifications-models for more info",
		},
		"host_or_usage_limits": {
			Type:          schema.TypeSet,
			Optional:      true,
			MaxItems:      1,
			ConflictsWith: []string{"dpm_limits"},
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"host_notification_threshold": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Notification threshold for hosts",
					},
					"host_limit": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Max number of hosts that can use this token",
					},
					"container_notification_threshold": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Notification threshold for containers",
					},
					"container_limit": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Max number of containers that can use this token",
					},
					"custom_metrics_notification_threshold": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Notification threshold for custom metrics",
					},
					"custom_metrics_limit": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Max number of custom metrics that can be sent with this token",
					},
					"high_res_metrics_notification_threshold": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Notification threshold for high-res metrics",
					},
					"high_res_metrics_limit": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "Max number of high-res metrics that can be sent with this token",
					},
				},
			},
		},
		"dpm_limits": {
			Type:          schema.TypeSet,
			Optional:      true,
			MaxItems:      1,
			ConflictsWith: []string{"host_or_usage_limits"},
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"dpm_notification_threshold": {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     -1,
						Description: "DPM level at which Splunk Observability Cloud sends the notification for this token. If you don't specify a notification, Splunk Observability Cloud sends the generic notification.",
					},
					"dpm_limit": {
						Type:        schema.TypeInt,
						Required:    true,
						Description: "The datapoints per minute (dpm) limit for this token. If you exceed this limit, Splunk Observability Cloud sends out an alert.",
					},
				},
			},
		},
		"expires_at": {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The calculated time in Unix milliseconds of when the token will be deactivated.",
		},
	}
}

func encodeTerraform(data *schema.ResourceData) (*orgtoken.Token, error) {
	token := &orgtoken.Token{
		Name:        data.Get("name").(string),
		Description: data.Get("description").(string),
		Disabled:    data.Get("disabled").(bool),
		Limits:      &orgtoken.Limit{},
		Secret:      data.Get("secret").(string),
	}

	if values, ok := data.GetOk("auth_scopes"); ok {
		for _, scope := range values.([]any) {
			token.AuthScopes = append(token.AuthScopes, scope.(string))
		}
	}

	if values, ok := data.GetOk("notifications"); ok {
		notifys, err := common.NewNotificationList(values.([]any))
		if err != nil {
			return nil, err
		}
		token.Notifications = notifys
	}

	if v, ok := data.GetOk("host_or_usage_limits"); ok {
		limits := v.(*schema.Set).List()[0].(map[string]any)
		token.Limits.CategoryQuota = &orgtoken.UsageLimits{
			HostThreshold:          common.AsPointerOnCondition(int64(limits["host_limit"].(int)), unsetThreshold),
			ContainerThreshold:     common.AsPointerOnCondition(int64(limits["container_limit"].(int)), unsetThreshold),
			CustomMetricThreshold:  common.AsPointerOnCondition(int64(limits["custom_metrics_limit"].(int)), unsetThreshold),
			HighResMetricThreshold: common.AsPointerOnCondition(int64(limits["high_res_metrics_limit"].(int)), unsetThreshold),
		}
		token.Limits.CategoryNotificationThreshold = &orgtoken.UsageLimits{
			HostThreshold:          common.AsPointerOnCondition(int64(limits["host_notification_threshold"].(int)), unsetThreshold),
			ContainerThreshold:     common.AsPointerOnCondition(int64(limits["container_notification_threshold"].(int)), unsetThreshold),
			CustomMetricThreshold:  common.AsPointerOnCondition(int64(limits["custom_metrics_notification_threshold"].(int)), unsetThreshold),
			HighResMetricThreshold: common.AsPointerOnCondition(int64(limits["high_res_metrics_notification_threshold"].(int)), unsetThreshold),
		}
	}

	if values, ok := data.GetOk("dpm_limits"); ok {
		limits := values.(*schema.Set).List()[0].(map[string]any)
		//nolint:gosec // Value is restricted to exceeding int32 max
		token.Limits.DpmQuota = common.AsPointer(int32(limits["dpm_limit"].(int)))
		if threshold, ok := limits["dpm_notification_threshold"]; ok {
			//nolint:gosec // Value is restricted to exceeding int32 max
			token.Limits.DpmNotificationThreshold = common.AsPointer(int32(threshold.(int)))
		}
	}

	return token, nil
}

func decodeTerraform(token *orgtoken.Token, data *schema.ResourceData) error {
	notifys, err := common.NewNotificationStringList(token.Notifications)
	if err != nil {
		return fmt.Errorf("notifications: %w", err)
	}

	data.SetId(token.Name)

	errs := multierr.Combine(
		data.Set("name", token.Name),
		data.Set("description", token.Description),
		data.Set("disabled", token.Disabled),
		data.Set("auth_scopes", token.AuthScopes),
		data.Set("notifications", notifys),
		data.Set("secret", token.Secret),
		data.Set("expires_at", token.Expiry),
	)

	if limits := token.Limits; limits != nil {
		switch {
		case limits.CategoryQuota != nil:
			values := map[string]any{}
			for field, val := range map[string]*int64{
				"host_limit":             limits.CategoryQuota.HostThreshold,
				"container_limit":        limits.CategoryQuota.ContainerThreshold,
				"custom_metrics_limit":   limits.CategoryQuota.CustomMetricThreshold,
				"high_res_metrics_limit": limits.CategoryQuota.HighResMetricThreshold,
			} {
				if val != nil && *val != -1 {
					values[field] = val
				}
			}
			if limits.CategoryNotificationThreshold != nil {
				for field, val := range map[string]*int64{
					"host_notification_threshold":             limits.CategoryNotificationThreshold.HostThreshold,
					"container_notification_threshold":        limits.CategoryNotificationThreshold.ContainerThreshold,
					"custom_metrics_notification_threshold":   limits.CategoryNotificationThreshold.CustomMetricThreshold,
					"high_res_metrics_notification_threshold": limits.CategoryNotificationThreshold.HighResMetricThreshold,
				} {
					if val != nil && *val != -1 {
						values[field] = val
					}
				}
			}
			if len(values) > 0 {
				errs = multierr.Append(errs, data.Set("host_or_usage_limits", []map[string]any{values}))
			}
		case limits.DpmQuota != nil:
			values := map[string]any{
				"dpm_limit": *limits.DpmQuota,
			}
			if limits.DpmNotificationThreshold != nil {
				values["dpm_notification_threshold"] = *limits.DpmNotificationThreshold
			}
			errs = multierr.Append(errs, data.Set("dpm_limits", []map[string]any{values}))
		}
	}

	return errs
}

func unsetThreshold(v int64) bool { return v != -1 }
