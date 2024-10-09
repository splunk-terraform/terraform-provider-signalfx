// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package team

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/signalfx/signalfx-go/team"
	"github.com/stretchr/testify/assert"
)

func TestNewSchema(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, newSchema())
}

func TestDecodeTerraform(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		data   map[string]any
		expect *team.Team
		errVal string
	}{
		{
			name:   "empty data",
			data:   map[string]any{},
			expect: &team.Team{},
			errVal: "",
		},
		{
			name: "invalid data set",
			data: map[string]any{
				"notifications_default": []any{
					0,
				},
			},
			expect: nil,
			errVal: "invalid notification string \"0\", not enough commas",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rd := schema.TestResourceDataRaw(
				t,
				newSchema(),
				tc.data,
			)
			tm, err := decodeTerraform(rd)
			assert.Equal(t, tc.expect, tm, "Must match the expected value")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				assert.NoError(t, err, "Must not error performing decode")
			}
		})
	}
}

func TestEncodeTerraform(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		schema map[string]*schema.Schema
		input  *team.Team
		errVal string
	}{
		{
			name:   "empty schema",
			schema: map[string]*schema.Schema{},
			input:  &team.Team{},
			errVal: "Invalid address to set: []string{\"name\"}",
		},
		{
			name: "name only set",
			schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
			},
			input:  &team.Team{},
			errVal: "Invalid address to set: []string{\"description\"}",
		},
		{
			name: "missing members definition",
			schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"description": {
					Type:     schema.TypeString,
					Required: true,
				},
			},
			input: &team.Team{
				Members: []string{"a", "b"},
			},
			errVal: "Invalid address to set: []string{\"members\"}",
		},
		{
			name: "nil notification provided",
			schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"description": {
					Type:     schema.TypeString,
					Required: true,
				},
			},
			input: &team.Team{
				NotificationLists: team.NotificationLists{
					Default: []*notification.Notification{
						nil,
					},
				},
			},
			errVal: "nil value provided",
		},
		{
			name: "missing notification definition",
			schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"description": {
					Type:     schema.TypeString,
					Required: true,
				},
			},
			input: &team.Team{
				NotificationLists: team.NotificationLists{
					Default: []*notification.Notification{
						{
							Type: "Email",
							Value: &notification.EmailNotification{
								Type:  "Email",
								Email: "example@com",
							},
						},
					},
				},
			},
			errVal: "Invalid address to set: []string{\"notifications_default\"}",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rd := schema.TestResourceDataRaw(
				t,
				tc.schema,
				map[string]any{},
			)

			err := encodeTerraform(tc.input, rd)
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				assert.NoError(t, err, "Must not error encoding data")
			}
		})
	}
}
