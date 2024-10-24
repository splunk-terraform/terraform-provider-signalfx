// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package team

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/signalfx/signalfx-go/team"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/check"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func newSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the team",
		},
		"description": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Description of the team (Optional)",
		},
		"members": {
			Type:        schema.TypeSet,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Members of team",
		},
		"notifications_critical": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of notification destinations to use for the critical alerts category.",
		},
		"notifications_default": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of notification destinations to use for the default alerts category.",
		},
		"notifications_info": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of notification destinations to use for the info alerts category.",
		},
		"notifications_major": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of notification destinations to use for the major alerts category.",
		},
		"notifications_minor": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of notification destinations to use for the minor alerts category.",
		},
		"notifications_warning": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Schema{
				Type:             schema.TypeString,
				ValidateDiagFunc: check.Notification(),
			},
			Description: "List of notification destinations to use for the warning alerts category.",
		},
		"url": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "URL of the team",
		},
	}
}

func decodeTerraform(rd tfext.Values) (*team.Team, error) {
	t := &team.Team{
		Id:          rd.Id(),
		Name:        rd.Get("name").(string),
		Description: rd.Get("description").(string),
	}

	for _, m := range rd.Get("members").(*schema.Set).List() {
		t.Members = append(t.Members, m.(string))
	}
	for name, field := range map[string]*[]*notification.Notification{
		"notifications_default": &t.NotificationLists.Default,
		"notifications_info":    &t.NotificationLists.Info,
		"notifications_minor":   &t.NotificationLists.Minor,
		"notifications_warning": &t.NotificationLists.Warning,
		"notifications_major":   &t.NotificationLists.Major,
	} {
		if val, ok := rd.Get(name).([]any); ok && len(val) > 0 {
			var err error
			(*field), err = common.NewNotificationList(val)
			if err != nil {
				return nil, err
			}
		}
	}
	return t, nil
}

func encodeTerraform(tm *team.Team, rd *schema.ResourceData) error {
	rd.SetId(tm.Id)
	if err := rd.Set("name", tm.Name); err != nil {
		return err
	}
	if err := rd.Set("description", tm.Description); err != nil {
		return err
	}

	if l := len(tm.Members); l > 0 {
		members := make([]any, l)
		for i, m := range tm.Members {
			members[i] = m
		}
		if err := rd.Set("members", schema.NewSet(schema.HashString, members)); err != nil {
			return err
		}
	}

	for name, values := range map[string][]*notification.Notification{
		"notifications_default": tm.NotificationLists.Default,
		"notifications_info":    tm.NotificationLists.Info,
		"notifications_minor":   tm.NotificationLists.Minor,
		"notifications_warning": tm.NotificationLists.Warning,
		"notifications_major":   tm.NotificationLists.Major,
	} {
		if len(values) == 0 {
			continue
		}
		items := make([]string, len(values))
		for i, n := range values {
			s, err := common.NewNotificationStringFromAPI(n)
			if err != nil {
				return err
			}
			items[i] = s
		}

		if err := rd.Set(name, items); err != nil {
			return err
		}
	}
	return nil
}
