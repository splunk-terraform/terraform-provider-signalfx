// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/orgtoken"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	vnext "github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/orgtoken"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
)

var previewVNextOrgToken = feature.GetGlobalRegistry().MustRegister(
	"vnext.org-token",
	feature.WithPreviewDescription("When enabled, org token will be managed with the updated behaviour."),
	feature.WithPreviewAddInVersion("v9.8.0"),
)

func orgTokenResource() *schema.Resource {
	if previewVNextOrgToken.Enabled() {
		return vnext.NewResource()
	}
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the token",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the token (Optional)",
			},
			"auth_scopes": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "Authentication scope, ex: INGEST, API, RUM ... (Optional)",
			},
			"disabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Flag that controls enabling the token. If set to `true`, the token is disabled, and you can't use it for authentication. Defaults to `false`",
			},
			"host_or_usage_limits": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"dpm_limits"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_notification_threshold": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Notification threshold for hosts",
						},
						"host_limit": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Max number of hosts that can use this token",
						},
						"container_notification_threshold": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Notification threshold for containers",
						},
						"container_limit": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Max number of containers that can use this token",
						},
						"custom_metrics_notification_threshold": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Notification threshold for custom metrics",
						},
						"custom_metrics_limit": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Max number of custom metrics that can be sent with this token",
						},
						"high_res_metrics_notification_threshold": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Notification threshold for high-res metrics",
						},
						"high_res_metrics_limit": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "Max number of high-res metrics that can be sent with this token",
						},
					},
				},
			},
			"dpm_limits": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"host_or_usage_limits"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dpm_notification_threshold": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     -1,
							Description: "DPM level at which Splunk Observability Cloud sends the notification for this token. If you don't specify a notification, Splunk Observability Cloud sends the generic notification.",
						},
						"dpm_limit": &schema.Schema{
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The datapoints per minute (dpm) limit for this token. If you exceed this limit, Splunk Observability Cloud sends out an alert.",
						},
					},
				},
			},
			"notifications": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: check.Notification(),
				},
				Description: "List of strings specifying where notifications will be sent when an incident occurs. See https://developers.signalfx.com/v2/docs/detector-model#notifications-models for more info",
			},
			"secret": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"store_secret": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to store the token's secret in the terraform state. Set to false for improved security. Defaults to true for backward compatibility.",
			},
		},

		Create: orgTokenCreate,
		Read:   orgTokenRead,
		Update: orgTokenUpdate,
		Delete: orgTokenDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func getPayloadOrgToken(d *schema.ResourceData) (*orgtoken.CreateUpdateTokenRequest, error) {
	token := &orgtoken.CreateUpdateTokenRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Disabled:    d.Get("disabled").(bool),
	}

	if val, ok := d.GetOk("auth_scopes"); ok {
		var auths []string
		for _, v := range val.([]interface{}) {
			auths = append(auths, v.(string))
		}
		token.AuthScopes = auths
	}

	if hostLimits, ok := d.GetOk("host_or_usage_limits"); ok {
		hostLimits := hostLimits.(*schema.Set).List()[0].(map[string]interface{})
		categoryQuotas := &orgtoken.UsageLimits{}
		categoryThresholds := &orgtoken.UsageLimits{}
		if threshold, ok := hostLimits["host_notification_threshold"]; ok && threshold.(int) != -1 {
			v := int64(threshold.(int))
			categoryThresholds.HostThreshold = &v
		}
		if threshold, ok := hostLimits["container_notification_threshold"]; ok && threshold.(int) != -1 {
			v := int64(threshold.(int))
			categoryThresholds.ContainerThreshold = &v
		}
		if threshold, ok := hostLimits["custom_metrics_notification_threshold"]; ok && threshold.(int) != -1 {
			v := int64(threshold.(int))
			categoryThresholds.CustomMetricThreshold = &v
		}
		if threshold, ok := hostLimits["high_res_metrics_notification_threshold"]; ok && threshold.(int) != -1 {
			v := int64(threshold.(int))
			categoryThresholds.HighResMetricThreshold = &v
		}

		if limit, ok := hostLimits["host_limit"]; ok && limit.(int) != -1 {
			v := int64(limit.(int))
			categoryQuotas.HostThreshold = &v
		}
		if limit, ok := hostLimits["container_limit"]; ok && limit.(int) != -1 {
			v := int64(limit.(int))
			categoryQuotas.ContainerThreshold = &v
		}
		if limit, ok := hostLimits["custom_metrics_limit"]; ok && limit.(int) != -1 {
			v := int64(limit.(int))
			categoryQuotas.CustomMetricThreshold = &v
		}
		if limit, ok := hostLimits["high_res_metrics_limit"]; ok && limit.(int) != -1 {
			v := int64(limit.(int))
			categoryQuotas.HighResMetricThreshold = &v
		}

		limits := &orgtoken.Limit{
			CategoryQuota:                 categoryQuotas,
			CategoryNotificationThreshold: categoryThresholds,
		}

		token.Limits = limits
	}

	if dpmLimits, ok := d.GetOk("dpm_limits"); ok {
		dpmLimits := dpmLimits.(*schema.Set).List()[0].(map[string]interface{})
		dq := int32(dpmLimits["dpm_limit"].(int))
		limits := &orgtoken.Limit{
			DpmQuota: &dq,
		}

		if limit, ok := dpmLimits["dpm_notification_threshold"]; ok {
			v := int32(limit.(int))
			limits.DpmNotificationThreshold = &v
		}

		token.Limits = limits
	}

	if notifications, ok := d.GetOk("notifications"); ok {
		notify, err := common.NewNotificationList(notifications.([]any))
		if err != nil {
			return nil, err
		}
		token.Notifications = notify
	}

	return token, nil
}

func orgTokenCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadOrgToken(d)
	if err != nil {
		return err
	}

	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Create Org Token Payload: %s", string(debugOutput))

	t, err := config.Client.CreateOrgToken(context.TODO(), payload)
	if err != nil {
		return err
	}
	d.SetId(t.Name)
	return orgTokenAPIToTF(d, t)
}

func orgTokenAPIToTF(d *schema.ResourceData, t *orgtoken.Token) error {
	debugOutput, _ := json.Marshal(t)
	log.Printf("[DEBUG] SignalFx: Got Org Token to enState: %s", string(debugOutput))

	if err := d.Set("name", t.Name); err != nil {
		return err
	}
	if err := d.Set("description", t.Description); err != nil {
		return err
	}
	if err := d.Set("disabled", t.Disabled); err != nil {
		return err
	}

	sort.Strings(t.AuthScopes)
	if err := d.Set("auth_scopes", t.AuthScopes); err != nil {
		return err
	}

	if t.Limits != nil {
		limits := t.Limits
		if limits.DpmQuota != nil {
			dpmStuff := map[string]interface{}{
				"dpm_limit": limits.DpmQuota,
			}
			if limits.DpmNotificationThreshold != nil {
				dpmStuff["dpm_notification_threshold"] = limits.DpmNotificationThreshold
			}
			if err := d.Set("dpm_limits", []map[string]interface{}{dpmStuff}); err != nil {
				return err
			}
		} else {
			usageStuff := map[string]interface{}{}
			if limits.CategoryQuota != nil && *limits.CategoryQuota != (orgtoken.UsageLimits{}) {
				cq := limits.CategoryQuota
				if cq.HostThreshold != nil {
					usageStuff["host_limit"] = *cq.HostThreshold
				}
				if cq.ContainerThreshold != nil {
					usageStuff["container_limit"] = *cq.ContainerThreshold
				}
				if cq.CustomMetricThreshold != nil {
					usageStuff["custom_metrics_limit"] = *cq.CustomMetricThreshold
				}
				if cq.HighResMetricThreshold != nil {
					usageStuff["high_res_metrics_limit"] = *cq.HighResMetricThreshold
				}
			}
			if limits.CategoryNotificationThreshold != nil && *limits.CategoryNotificationThreshold != (orgtoken.UsageLimits{}) {
				cnt := limits.CategoryNotificationThreshold
				if cnt.HostThreshold != nil {
					usageStuff["host_notification_threshold"] = *cnt.HostThreshold
				}
				if cnt.ContainerThreshold != nil {
					usageStuff["container_notification_threshold"] = *cnt.ContainerThreshold
				}
				if cnt.CustomMetricThreshold != nil {
					usageStuff["custom_metrics_notification_threshold"] = *cnt.CustomMetricThreshold
				}
				if cnt.HighResMetricThreshold != nil {
					usageStuff["high_res_metrics_notification_threshold"] = *cnt.HighResMetricThreshold
				}
			}
			if len(usageStuff) > 0 {
				if err := d.Set("host_or_usage_limits", []map[string]interface{}{usageStuff}); err != nil {
					return err
				}
			}
		}
	}

	notifications := make([]string, len(t.Notifications))
	for i, not := range t.Notifications {
		tfNot, err := common.NewNotificationStringFromAPI(not)
		if err != nil {
			return err
		}
		notifications[i] = tfNot
	}
	if err := d.Set("notifications", notifications); err != nil {
		return err
	}

	secret := func() string {
		if d.Get("store_secret").(bool) {
			return t.Secret
		}
		return ""
	}()

	if err := d.Set("secret", secret); err != nil {
		return err
	}

	return nil
}

func orgTokenLookupDataSource() *schema.Resource {
	return &schema.Resource{
		Read: orgTokenLookupRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the token to look up",
			},
			"secret": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The token's secret value",
			},
		},
	}
}

func orgTokenLookupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	name := d.Get("name").(string)

	t, err := config.Client.GetOrgToken(context.TODO(), name)
	if err != nil {
		return fmt.Errorf("error looking up org token %s: %s", name, err)
	}

	d.SetId(name)
	if err := d.Set("secret", t.Secret); err != nil {
		return err
	}

	return nil
}

func orgTokenRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	log.Printf("[DEBUG] SignalFx: Looking for org token %s\n", d.Id())
	t, err := config.Client.GetOrgToken(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	return orgTokenAPIToTF(d, t)
}

func orgTokenUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	payload, err := getPayloadOrgToken(d)
	if err != nil {
		return err
	}
	debugOutput, _ := json.Marshal(payload)
	log.Printf("[DEBUG] SignalFx: Update Org Token Payload: %s", string(debugOutput))

	t, err := config.Client.UpdateOrgToken(context.TODO(), d.Id(), payload)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] SignalFx: Update Org Token Response: %v", t)

	d.SetId(t.Name)
	return orgTokenAPIToTF(d, t)
}

func orgTokenDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)

	return config.Client.DeleteOrgToken(context.TODO(), d.Id())
}
