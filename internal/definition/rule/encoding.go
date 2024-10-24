// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package rule

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/detector"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func DecodeTerraform(rd tfext.Values) ([]*detector.Rule, error) {
	set, ok := rd.Get("rule").(*schema.Set)
	if !ok {
		return nil, errors.New("no field defined for rule")
	}
	rules := make([]*detector.Rule, 0, set.Len())
	for _, v := range set.List() {
		data := v.(map[string]any)
		rule := &detector.Rule{
			Description:          data["description"].(string),
			Disabled:             data["disabled"].(bool),
			DetectLabel:          data["detect_label"].(string),
			Severity:             detector.Severity(data["severity"].(string)),
			ParameterizedBody:    data["parameterized_body"].(string),
			ParameterizedSubject: data["parameterized_subject"].(string),
			RunbookUrl:           data["runbook_url"].(string),
			Tip:                  data["tip"].(string),
		}
		if n, ok := data["notifications"].([]any); ok {
			notifiy, err := common.NewNotificationList(n)
			if err != nil {
				return nil, err
			}
			rule.Notifications = notifiy
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func EncodeTerraform(rules []*detector.Rule, rd *schema.ResourceData) error {
	if len(rules) == 0 {
		return nil
	}

	items := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		notifys, err := common.NewNotificationStringList(r.Notifications)
		if err != nil {
			return fmt.Errorf("notification issue: %w", err)
		}
		items = append(items, map[string]any{
			"detect_label":          r.DetectLabel,
			"description":           r.Description,
			"disabled":              r.Disabled,
			"notifications":         notifys,
			"parameterized_body":    r.ParameterizedBody,
			"parameterized_subject": r.ParameterizedSubject,
			"runbook_url":           r.RunbookUrl,
			"severity":              r.Severity,
			"tip":                   r.Tip,
		})
	}

	return rd.Set("rule", items)
}
