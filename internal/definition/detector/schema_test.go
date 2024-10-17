// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
)

func TestDecodeTerraform(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		data   map[string]any
		expect *detector.Detector
		errVal string
	}{
		{
			name: "empty data",
			data: map[string]any{},
			expect: &detector.Detector{
				AuthorizedWriters: &detector.AuthorizedWriters{},
				TimeZone:          "UTC",
				MaxDelay:          common.AsPointer[int32](0),
				MinDelay:          common.AsPointer[int32](0),
				Rules:             []*detector.Rule{},
				VisualizationOptions: &detector.Visualization{
					ShowDataMarkers: true,
					Time: &detector.Time{
						Type:  "relative",
						Range: common.AsPointer[int64](3600000),
					},
				},
				DetectorOrigin: "Standard",
			},
			errVal: "",
		},
		{
			name: "using absolute time references",
			data: map[string]any{
				"start_time": 100,
				"end_time":   1000,
			},
			expect: &detector.Detector{
				AuthorizedWriters: &detector.AuthorizedWriters{},
				TimeZone:          "UTC",
				MaxDelay:          common.AsPointer[int32](0),
				MinDelay:          common.AsPointer[int32](0),
				Rules:             []*detector.Rule{},
				VisualizationOptions: &detector.Visualization{
					ShowDataMarkers: true,
					Time: &detector.Time{
						Type:  "absolute",
						Start: common.AsPointer[int64](100000),
						End:   common.AsPointer[int64](1000000),
					},
				},
				DetectorOrigin: "Standard",
			},
		},
		{
			name: "Defines added fields",
			data: map[string]any{
				"teams":                   []any{"team-02", "team-01"},
				"tags":                    []any{"tag-02", "tag-01"},
				"authorized_writer_teams": []any{"team-01"},
				"authorized_writer_users": []any{"user-01"},
				"viz_options": []any{
					map[string]any{"label": "label-01", "color": "pink"},
				},
			},
			expect: &detector.Detector{
				AuthorizedWriters: &detector.AuthorizedWriters{
					Teams: []string{"team-01"},
					Users: []string{"user-01"},
				},
				TimeZone: "UTC",
				MaxDelay: common.AsPointer[int32](0),
				MinDelay: common.AsPointer[int32](0),
				Rules:    []*detector.Rule{},
				Teams:    []string{"team-02", "team-01"},
				Tags:     []string{"tag-02", "tag-01"},
				VisualizationOptions: &detector.Visualization{
					ShowDataMarkers: true,
					Time: &detector.Time{
						Type:  "relative",
						Range: common.AsPointer[int64](3600000),
					},
					PublishLabelOptions: []*detector.PublishLabelOptions{
						{Label: "label-01", PaletteIndex: common.AsPointer[int32](14)},
					},
				},
				DetectorOrigin: "Standard",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dt, err := decodeTerraform(
				schema.TestResourceDataRaw(t, newSchema(), tc.data),
			)
			assert.Equal(t, tc.expect, dt, "Must match the expected value")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not report an error")
			}
		})
	}
}

func TestEncodeTerraform(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		input  *detector.Detector
		errVal string
	}{
		{
			name:   "empty detector",
			input:  &detector.Detector{},
			errVal: "",
		},
		{
			name: "time range",
			input: &detector.Detector{
				VisualizationOptions: &detector.Visualization{
					Time: &detector.Time{
						Type:  "relative",
						Range: common.AsPointer[int64](1000),
					},
				},
			},
		},
		{
			name: "completely populated detector",
			input: &detector.Detector{
				Id:          "01",
				Name:        "my detector",
				Description: "description",
				TimeZone:    "UTC",
				ProgramText: `detect(when(data('*').count() < 1)).publish('no data sent')`,
				Teams:       []string{"team-01"},
				Tags:        []string{"tag-01"},
				MinDelay:    common.AsPointer[int32](1000),
				MaxDelay:    common.AsPointer[int32](1000),
				AuthorizedWriters: &detector.AuthorizedWriters{
					Users: []string{"user-01"},
					Teams: []string{"team-01"},
				},
				VisualizationOptions: &detector.Visualization{
					DisableSampling: true,
					ShowDataMarkers: true,
					ShowEventLines:  false,
					Time: &detector.Time{
						Type:  "absolute",
						Start: common.AsPointer[int64](100),
						End:   common.AsPointer[int64](200),
					},
					PublishLabelOptions: []*detector.PublishLabelOptions{
						{Label: "label-01", PaletteIndex: common.AsPointer[int32](12)},
					},
				},
				Rules: []*detector.Rule{
					{
						Description: "Default team alert",
						DetectLabel: "label-01",
						Notifications: []*notification.Notification{
							{Type: "Team", Value: &notification.TeamNotification{Type: "Team", Team: "team-01"}},
						},
					},
				},
			},
		},
	} {
		err := encodeTerraform(tc.input, schema.TestResourceDataRaw(
			t,
			newSchema(),
			map[string]any{},
		))

		if tc.errVal != "" {
			assert.EqualError(t, err, tc.errVal, "Must match the expected error message")
		} else {
			assert.NoError(t, err, "Must not error ")
		}
	}
}
