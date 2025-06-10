// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package autoarchivesettings

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	autoarch "github.com/signalfx/signalfx-go/automated-archival"
	"github.com/stretchr/testify/assert"
)

func TestSchemaDecode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		values map[string]any
		expect *autoarch.AutomatedArchivalSettings
		errVal string
	}{
		{
			name:   "no values provided",
			values: map[string]any{},
			expect: &autoarch.AutomatedArchivalSettings{},
			errVal: "",
		},
		{
			name: "all values provided",
			values: map[string]any{
				"enabled":         true,
				"lookback_period": "P30D",
				"grace_period":    "P15D",
				"ruleset_limit":   10,
				"version":         1.0,
			},
			expect: &autoarch.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P15D",
				RulesetLimit:   autoarch.PtrInt32(10),
				Version:        1.0,
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := schema.TestResourceDataRaw(t, newSchema(), tc.values)
			settings, err := decodeTerraform(data)
			assert.Equal(t, tc.expect, settings, "Must match expected automated archival settings")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not error when decoding settings")
			}
		})
	}
}

func TestSchemaEncode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		settings *autoarch.AutomatedArchivalSettings
		errVal   string
	}{
		{
			name:     "empty settings",
			settings: &autoarch.AutomatedArchivalSettings{},
			errVal:   "",
		},
		{
			name: "all values provided",
			settings: &autoarch.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P15D",
				RulesetLimit:   autoarch.PtrInt32(10),
				Version:        1.0,
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := schema.TestResourceDataRaw(t, newSchema(), map[string]any{})
			err := encodeTerraform(tc.settings, data)

			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error value")
			} else {
				assert.NoError(t, err, "Must not error when encoding settings")
			}
		})
	}
}
