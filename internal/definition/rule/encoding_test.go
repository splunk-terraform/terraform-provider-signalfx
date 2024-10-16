// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package rule

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"
)

func TestDecodeTerraform(t *testing.T) {
	t.Parallel()

	resource := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"rule": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: NewSchema(),
				},
				Set: Hash,
			},
		},
	}

	for _, tc := range []struct {
		name   string
		data   func() *schema.ResourceData
		expect []*detector.Rule
		errVal string
	}{
		{
			name: "invalid resource data",
			data: func() *schema.ResourceData {
				return &schema.ResourceData{}
			},
			expect: nil,
			errVal: "no field defined for rule",
		},
		{
			name: "default resource value",
			data: func() *schema.ResourceData {
				return resource.TestResourceData()
			},
			expect: []*detector.Rule{},
			errVal: "",
		},
		{
			name: "values defined",
			data: func() *schema.ResourceData {
				data := resource.TestResourceData()
				_ = EncodeTerraform(
					[]*detector.Rule{
						{
							Severity:    detector.INFO,
							Disabled:    false,
							Description: "my first rule",
							Notifications: []*notification.Notification{
								{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
							},
						},
					},
					data,
				)
				return data
			},
			expect: []*detector.Rule{
				{
					Severity:    detector.INFO,
					Disabled:    false,
					Description: "my first rule",
					Notifications: []*notification.Notification{
						{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
					},
				},
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := DecodeTerraform(tc.data())
			assert.Equal(t, tc.expect, actual, "Must matcht the expected values")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error message")
			} else {
				assert.NoError(t, err, "Must not error")
			}
		})
	}
}

func TestEncodeTerraform(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		rules  []*detector.Rule
		errVal string
	}{
		{
			name:   "no values provided",
			rules:  nil,
			errVal: "",
		},
		{
			name: "bad notification value",
			rules: []*detector.Rule{
				{Notifications: []*notification.Notification{
					{Type: "Unset"},
				}},
			},
			errVal: "notification issue: unknown type <nil> provided",
		},
		{
			name: "valid rules provided",
			rules: []*detector.Rule{
				{
					Description:          "custom rule",
					DetectLabel:          "errs",
					Disabled:             true,
					ParameterizedSubject: "errors going brrr",
					ParameterizedBody:    "The number of errors have reached brrr levels",
					Severity:             detector.MAJOR,
					RunbookUrl:           "http://localhost/brrr",
					Tip:                  "Avoid errors going brr",
					Notifications: []*notification.Notification{
						{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
					},
				},
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resource := &schema.Resource{
				Schema: map[string]*schema.Schema{
					"rule": {
						Type:     schema.TypeSet,
						Required: true,
						Elem: &schema.Resource{
							Schema: NewSchema(),
						},
						Set: Hash,
					},
				},
			}

			err := EncodeTerraform(tc.rules, resource.TestResourceData())
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error value")
			} else {
				assert.NoError(t, err, "Must not error")
			}
		})

	}
}
