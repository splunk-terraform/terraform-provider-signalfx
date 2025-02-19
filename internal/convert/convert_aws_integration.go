// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import "github.com/signalfx/signalfx-go/integration"

func ToAWSNamespaceRule(in any) *integration.AwsNameSpaceSyncRule {
	data := in.(map[string]any)

	rule := &integration.AwsNameSpaceSyncRule{
		Namespace: integration.AwsService(data["namespace"].(string)),
	}

	if action, ok := data["default_action"].(string); ok {
		rule.DefaultAction = integration.AwsSyncRuleFilterAction(action)
	}

	if action, ok := data["filter_action"].(string); ok {
		rule.Filter = &integration.AwsSyncRuleFilter{
			Action: integration.AwsSyncRuleFilterAction(action),
			Source: data["filter_source"].(string),
		}
	}

	return rule
}

func ToAWSCustomNamespaceRule(in any) *integration.AwsCustomNameSpaceSyncRule {
	data := in.(map[string]any)
	sync := &integration.AwsCustomNameSpaceSyncRule{
		Namespace: data["namespace"].(string),
	}

	if action, ok := data["default_action"].(string); ok && action != "" {
		sync.DefaultAction = integration.AwsSyncRuleFilterAction(action)
	}

	if action, ok := data["filter_action"].(string); ok && action != "" {
		sync.Filter = &integration.AwsSyncRuleFilter{
			Action: integration.AwsSyncRuleFilterAction(action),
			Source: data["filter_source"].(string),
		}
	}

	return sync
}
