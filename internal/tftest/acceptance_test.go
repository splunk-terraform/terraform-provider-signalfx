// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestAcceptanceHandlerOptions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		opts   []AcceptanceHandlerOption
		errVal string
	}{
		{
			name:   "no options",
			opts:   []AcceptanceHandlerOption{},
			errVal: "missing resource and datasource defintions",
		},
		{
			name: "defines data sources",
			opts: []AcceptanceHandlerOption{
				WithAcceptanceDataSources(map[string]*schema.Resource{
					"blank": {},
				}),
			},
		},
		{
			name: "defines resources",
			opts: []AcceptanceHandlerOption{
				WithAcceptanceResources(map[string]*schema.Resource{
					"blank": {},
				}),
			},
		},
		{
			name: "all options",
			opts: []AcceptanceHandlerOption{
				WithAcceptanceResources(map[string]*schema.Resource{
					"blank": {},
				}),
				WithAcceptanceDataSources(map[string]*schema.Resource{
					"blank": {},
				}),
				WithAcceptanceBeforeAll(func() {
					// Do nothing
				}),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NewAcceptanceHandler(tc.opts...).Validate()
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not error")
			}
		})
	}
}

func TestAcceptanceHandlerTest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		env     map[string]string
		skipped bool
	}{
		{
			name:    "No environment variables set",
			env:     map[string]string{},
			skipped: true,
		},
		{
			name: "environment vars set",
			env: map[string]string{
				"SFX_AUTH_TOKEN": "aaaa",
				"SFX_API_URL":    "https://localhost",
			},
			skipped: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			CleanEnvVars(t)

			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			handler := NewAcceptanceHandler(
				WithAcceptanceResources(map[string]*schema.Resource{
					"nop": {},
				}),
			)

			t.Cleanup(func() {
				assert.Equal(t, tc.skipped, t.Skipped(), "Must have been skipped")
			})

			handler.Test(t, []resource.TestStep{
				{
					Config: "{}", // Empty configuration
				},
			})
		})
	}
}
