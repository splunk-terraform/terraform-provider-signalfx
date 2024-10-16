// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package rule

import (
	"testing"

	"github.com/signalfx/signalfx-go/detector"
	"github.com/stretchr/testify/assert"
)

func TestNewSchema(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, NewSchema(), "Must not have an empty schema")
}

func TestHashSchema(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		rule map[string]any
		code int
	}{
		{
			name: "no values defined",
			rule: make(map[string]any),
			code: 1995047473,
		},
		{
			name: "min values defined",
			rule: map[string]any{
				"description":  "my custom rule",
				"severity":     detector.CRITICAL,
				"detect_label": "my-metric",
				"disabled":     false,
			},
			code: 3636232935,
		},
		{
			name: "all values defined",
			rule: map[string]any{
				"description":           "my custom rule",
				"severity":              detector.CRITICAL,
				"detect_label":          "my-metric",
				"disabled":              false,
				"parameterized_body":    "something has gone wrong, go investigate",
				"parameterized_subject": "there is an alert",
				"runbook_url":           "http://example.com",
				"tip":                   "login to investigate the issue",
				"notifications":         []any{"Email,example@com", "Email,foo@bar"},
			},
			code: 137896672,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.code, HashSchema(tc.rule), "Must match the expected rule")
		})
	}
}
