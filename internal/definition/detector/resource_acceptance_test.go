// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/detector"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/team"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestAcceptance(t *testing.T) {
	for _, tc := range []struct {
		name  string
		steps []resource.TestStep
	}{
		{
			name: "minimal detector",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/minimal.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_detector.minimal", "name", "my minimal detector"),
						resource.TestCheckResourceAttr("signalfx_detector.minimal", "program_text", "detect(when(const(1) > 1)).publish('HCF')\n"),
					),
					ExpectNonEmptyPlan: false,
				},
			},
		},
		{
			name: "advanced detector",
			steps: []resource.TestStep{
				{
					Config: tftest.LoadConfig("testdata/advanced_00.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "name", "example detector"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "description", "A detector made from terraform"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "timezone", "Europe/Paris"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.#", "2"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.0", "tag-1"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.1", "tag-2"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "teams.#", "1"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "max_delay", "30"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "min_delay", "15"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "program_text", "signal = data('app.delay').max().publish('app delay')\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\ndetect(when(signal > 60, '30m')).publish('Processing old messages 30m')\n"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.#", "2"),

						// Rule #1
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.description", "maximum > 60 for 5m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.detect_label", "Processing old messages 5m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.disabled", "false"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.notifications.#", "1"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.notifications.0", "Email,foo-alerts@example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.parameterized_body", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.parameterized_subject", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.runbook_url", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.severity", "Warning"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.tip", ""),

						// Rule #2
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.description", "maximum > 60 for 30m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.detect_label", "Processing old messages 30m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.disabled", "false"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.notifications.#", "1"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.notifications.0", "Email,foo-alerts@example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.parameterized_body", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.parameterized_subject", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.runbook_url", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.severity", "Critical"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.tip", ""),
					),
					ExpectNonEmptyPlan: false,
					Destroy:            false,
				},
				{
					Config: tftest.LoadConfig("testdata/advanced_01.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "name", "max average delay UPDATED"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "description", "your application is slowER"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "timezone", "Europe/Paris"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.#", "3"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.0", "tag-1"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.1", "tag-2"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.2", "tag-3"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "teams.#", "0"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "max_delay", "60"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "min_delay", "30"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector",
							"time_range", "3600"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "program_text", "signal = data('app.delay2').max().publish('app delay')\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\ndetect(when(signal > 60, '30m')).publish('Processing old messages 30m')\n"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "show_data_markers", "true"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "show_event_lines", "true"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "disable_sampling", "true"),

						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "label_resolutions.%", "2"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "label_resolutions.Processing old messages 30m", "1000"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "label_resolutions.Processing old messages 5m", "1000"),

						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.#", "2"),

						// Rule #1
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.description", "NEW maximum > 60 for 5m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.notifications.0", "Email,foo-alerts@example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.parameterized_body", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.parameterized_subject", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.severity", "Warning"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.runbook_url", "https://www.example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.1.tip", "reboot it"),

						// Rule #2
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.description", "NEW maximum > 60 for 30m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.detect_label", "Processing old messages 30m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.disabled", "false"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.notifications.#", "1"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.notifications.0", "Email,foo-alerts@example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.parameterized_body", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.parameterized_subject", ""),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.runbook_url", "https://www.example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.severity", "Critical"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.tip", ""),
					),
				},
				{
					Config: tftest.LoadConfig("testdata/advanced_02.tf"),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "name", "max average delay UPDATED"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "description", "your application is slowER"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "timezone", "Europe/Paris"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "tags.#", "0"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "teams.#", "0"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "max_delay", "60"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "min_delay", "30"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector",
							"time_range", "3600"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "program_text", "signal = data('app.delay2').max().publish('app delay')\ndetect(when(signal > 60, '5m')).publish('Processing old messages 5m')\n"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "show_data_markers", "true"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "show_event_lines", "true"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "disable_sampling", "true"),

						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.#", "1"),

						// Rule #1
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.description", "NEW maximum > 60 for 5m"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.notifications.0", "Email,foo-alerts@example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.severity", "Warning"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.runbook_url", "https://www.example.com"),
						resource.TestCheckResourceAttr("signalfx_detector.my_detector", "rule.0.tip", "reboot it"),
					),
					ExpectNonEmptyPlan: true,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tftest.NewAcceptanceHandler(
				tftest.WithAcceptanceResources(map[string]*schema.Resource{
					detector.ResourceName: detector.NewResource(),
					team.ResourceName:     team.NewResource(),
				}),
			).
				Test(t, tc.steps)
		})
	}
}
