package signalfx

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/signalfx/signalfx-go/orgtoken"
)

func orgTokenResource() *schema.Resource {
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
							Description: "DPM level at which SignalFx sends the notification for this token. If you don't specify a notification, SignalFx sends the generic notification.",
						},
						"dpm_limit": &schema.Schema{
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The datapoints per minute (dpm) limit for this token. If you exceed this limit, SignalFx sends out an alert.",
						},
					},
				},
			},
			"notifications": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateNotification,
				},
				Description: "List of strings specifying where notifications will be sent when an incident occurs. See https://developers.signalfx.com/v2/docs/detector-model#notifications-models for more info",
			},
			"secret": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},

		Create: orgTokenCreate,
		Read:   orgTokenRead,
		Update: orgTokenUpdate,
		Delete: orgTokenDelete,
		Exists: orgTokenExists,
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
		notify, err := getNotifications(notifications.([]interface{}))
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

	if t.Limits != nil {
		limits := t.Limits
		if limits.DpmQuota != nil {
			dpmStuff := map[string]interface{}{
				"dpm_limit": limits.DpmQuota,
			}
			if limits.DpmNotificationThreshold != nil {
				dpmStuff["dpm_notification_threshold"] = limits.DpmNotificationThreshold
			}
			if err := d.Set("dpm_limit", dpmStuff); err != nil {
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

		notifications := make([]string, len(t.Notifications))
		for i, not := range t.Notifications {
			tfNot, err := getNotifyStringFromAPI(not)
			if err != nil {
				return err
			}
			notifications[i] = tfNot
		}
		if err := d.Set("notifications", notifications); err != nil {
			return err
		}

		if err := d.Set("secret", t.Secret); err != nil {
			return err
		}
	}

	return nil
}

func orgTokenRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*signalfxConfig)
	fmt.Printf("[DEBUG] SignalFx: Looking for org token %s\n", d.Id())
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

func orgTokenExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*signalfxConfig)
	_, err := config.Client.GetOrgToken(context.TODO(), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
