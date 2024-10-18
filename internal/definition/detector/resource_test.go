// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/signalfx/signalfx-go/detector"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/common"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestNewResource(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewResource(), "Must have a valid resource defined")
}

func TestResourceCreate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[detector.Detector]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &detector.Detector{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
	} {
		tc.TestCreate(t)
	}
}

func TestResourceRead(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[detector.Detector]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &detector.Detector{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/detector/id-01": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "failed to read body", http.StatusBadRequest)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &detector.Detector{
				Id: "id-01",
			},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "Bad status 400: failed to read body\n"},
			},
		},
		{
			Name:     "Successful Read",
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/detector/id-01": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&detector.Detector{
						Id:          "id-01",
						Name:        "test detector",
						Description: "An example detector response",
						TimeZone:    "Australia/Sydney",
						MaxDelay:    common.AsPointer[int32](100),
						MinDelay:    common.AsPointer[int32](100),
						ProgramText: `detect(when(data('*').count() < 1)).publish('no data')`,
						Rules: []*detector.Rule{
							{
								DetectLabel: "no data",
								Notifications: []*notification.Notification{
									{Type: "Team", Value: &notification.TeamNotification{Type: "Team", Team: "awesome-team"}},
								},
							},
						},
						Tags:           []string{"tag-01"},
						Teams:          []string{"team-01"},
						DetectorOrigin: "Standard",
					})
				},
			}),
			Input: &detector.Detector{
				Id: "id-01",
			},
			Expect: &detector.Detector{
				Id:                "id-01",
				Name:              "test detector",
				Description:       "An example detector response",
				AuthorizedWriters: &detector.AuthorizedWriters{},
				TimeZone:          "Australia/Sydney",
				MaxDelay:          common.AsPointer[int32](100000),
				MinDelay:          common.AsPointer[int32](100000),
				ProgramText:       `detect(when(data('*').count() < 1)).publish('no data')`,
				Rules: []*detector.Rule{
					{
						DetectLabel: "no data",
						Notifications: []*notification.Notification{
							{Type: "Team", Value: &notification.TeamNotification{Type: "Team", Team: "awesome-team"}},
						},
					},
				},
				Tags:                 []string{"tag-01"},
				Teams:                []string{"team-01"},
				DetectorOrigin:       "Standard",
				VisualizationOptions: &detector.Visualization{},
			},
		},
		{
			Name:     "Reported over mts Read",
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/detector/id-01": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&detector.Detector{
						Id:           "id-01",
						Name:         "test detector",
						Description:  "An example detector response",
						TimeZone:     "Australia/Sydney",
						MaxDelay:     common.AsPointer[int32](100),
						MinDelay:     common.AsPointer[int32](100),
						ProgramText:  `detect(when(data('*').count() < 1)).publish('no data')`,
						OverMTSLimit: true,
						Rules: []*detector.Rule{
							{
								DetectLabel: "no data",
								Notifications: []*notification.Notification{
									{Type: "Team", Value: &notification.TeamNotification{Type: "Team", Team: "awesome-team"}},
								},
							},
						},
						Tags:           []string{"tag-01"},
						Teams:          []string{"team-01"},
						DetectorOrigin: "Standard",
					})
				},
			}),
			Input: &detector.Detector{
				Id: "id-01",
			},
			Expect: &detector.Detector{
				Id:                "id-01",
				Name:              "test detector",
				Description:       "An example detector response",
				AuthorizedWriters: &detector.AuthorizedWriters{},
				TimeZone:          "Australia/Sydney",
				MaxDelay:          common.AsPointer[int32](100000),
				MinDelay:          common.AsPointer[int32](100000),
				ProgramText:       `detect(when(data('*').count() < 1)).publish('no data')`,
				OverMTSLimit:      true,
				Rules: []*detector.Rule{
					{
						DetectLabel: "no data",
						Notifications: []*notification.Notification{
							{Type: "Team", Value: &notification.TeamNotification{Type: "Team", Team: "awesome-team"}},
						},
					},
				},
				Tags:                 []string{"tag-01"},
				Teams:                []string{"team-01"},
				DetectorOrigin:       "Standard",
				VisualizationOptions: &detector.Visualization{},
			},
			Issues: diag.Diagnostics{
				{Severity: diag.Warning, Summary: "detector is over mts limit"},
			},
		},
	} {
		tc.TestRead(t)
	}
}

func TestResourceUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[detector.Detector]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &detector.Detector{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
	} {
		tc.TestUpdate(t)
	}
}

func TestResourceDelete(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[detector.Detector]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  tfext.NopDecodeTerraform[detector.Detector],
			Input:    &detector.Detector{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "successful delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/detector/detector-01": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					w.WriteHeader(http.StatusNoContent)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  tfext.NopDecodeTerraform[detector.Detector],
			Input: &detector.Detector{
				Id: "detector-01",
			},
			Expect: nil,
			Issues: nil,
		},
		{
			Name: "failed delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/detector/detector-01": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "invalid detector", http.StatusBadRequest)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  tfext.NopDecodeTerraform[detector.Detector],
			Input: &detector.Detector{
				Id: "detector-01",
			},
			Expect: nil,
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "Unexpected status code: 400: invalid detector\n"},
			},
		},
	} {
		tc.TestDelete(t)
	}
}
