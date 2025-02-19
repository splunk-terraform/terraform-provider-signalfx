// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"testing"

	"github.com/signalfx/signalfx-go/integration"
	"github.com/stretchr/testify/assert"
)

func TestToAWSNamespaceRule(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		input  map[string]any
		expect *integration.AwsNameSpaceSyncRule
	}{
		{
			name: "min. required details",
			input: map[string]any{
				"namespace": "my-awesome-namespace",
			},
			expect: &integration.AwsNameSpaceSyncRule{
				Namespace: integration.AwsService("my-awesome-namespace"),
			},
		},
		{
			name: "actions set",
			input: map[string]any{
				"namespace":      "AWS/linux",
				"default_action": "Include",
				"filter_action":  "Exclude",
				"filter_source":  "source",
			},
			expect: &integration.AwsNameSpaceSyncRule{
				Namespace:     integration.AwsService("AWS/linux"),
				DefaultAction: integration.INCLUDE,
				Filter: &integration.AwsSyncRuleFilter{
					Action: integration.EXCLUDE,
					Source: "source",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := ToAWSNamespaceRule(tc.input)
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
		})
	}
}

func TestToAWSCustomNamespaceRule(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		in     map[string]any
		expect *integration.AwsCustomNameSpaceSyncRule
	}{
		{
			name: "min. required values",
			in: map[string]any{
				"namespace": "namespace",
			},
			expect: &integration.AwsCustomNameSpaceSyncRule{
				Namespace: "namespace",
			},
		},
		{
			name: "all fields",
			in: map[string]any{
				"namespace":      "ns",
				"default_action": "Include",
				"filter_action":  "Exclude",
				"filter_source":  "source",
			},
			expect: &integration.AwsCustomNameSpaceSyncRule{
				Namespace:     "ns",
				DefaultAction: integration.INCLUDE,
				Filter: &integration.AwsSyncRuleFilter{
					Action: integration.EXCLUDE,
					Source: "source",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := ToAWSCustomNamespaceRule(tc.in)
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
		})
	}
}
