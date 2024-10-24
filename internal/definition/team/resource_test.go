// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package team

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/signalfx/signalfx-go/notification"
	"github.com/signalfx/signalfx-go/team"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestResourceCreate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[team.Team]{
		{
			Name: "no provider set",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &team.Team{},
			Expect:   nil,
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "failed create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST /v2/team": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Failed to create", http.StatusBadRequest)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Name:        "test",
				Description: "test team",
			},
			Expect: nil,
			Issues: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "route \"/v2/team\" had issues with status code 400",
				},
			},
		},
		{
			Name: "successful create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST	/v2/team": func(w http.ResponseWriter, r *http.Request) {
					var req team.CreateUpdateTeamRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					_ = r.Body.Close()
					_ = json.NewEncoder(w).Encode(&team.Team{
						Id:          "0001",
						Name:        "test",
						Description: "test team",
					})
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Name:        "test",
				Description: "test team",
			},
			Expect: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Issues: nil,
		},
		{
			Name: "successful create with members",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST	/v2/team": func(w http.ResponseWriter, r *http.Request) {
					var req team.CreateUpdateTeamRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					_ = r.Body.Close()
					_ = json.NewEncoder(w).Encode(&team.Team{
						Id:                "0001",
						Name:              req.Name,
						Description:       req.Description,
						Members:           req.Members,
						NotificationLists: req.NotificationLists,
					})
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Name:        "test",
				Description: "test team",
				Members: []string{
					"0001",
					"0002",
				},
				NotificationLists: team.NotificationLists{
					Default: []*notification.Notification{
						{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
					},
				},
			},
			Expect: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
				Members: []string{
					"0001",
					"0002",
				},
				NotificationLists: team.NotificationLists{
					Default: []*notification.Notification{
						{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
					},
				},
			},
			Issues: nil,
		},
	} {
		tc.TestCreate(t)
	}
}

func TestResourceRead(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[team.Team]{
		{
			Name: "no provider set",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &team.Team{},
			Expect:   nil,
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "failed read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/team/0001": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Failed to read", http.StatusBadRequest)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Expect: nil,
			Issues: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "route \"/v2/team/0001\" had issues with status code 400",
				},
			},
		},
		{
			Name: "successful read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/team/0001": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&team.Team{
						Id:          "0001",
						Name:        "test",
						Description: "test team",
					})
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Id: "0001",
			},
			Expect: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Issues: nil,
		},
		{
			Name: "successful read with extended details",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/team/0001": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&team.Team{
						Id:          "0001",
						Name:        "test",
						Description: "test team",
						Members:     []string{"a", "b", "c"},
						NotificationLists: team.NotificationLists{
							Default: []*notification.Notification{
								{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
							},
						},
					})
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Id: "0001",
			},
			Expect: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
				Members: []string{
					"a", "b", "c",
				},
				NotificationLists: team.NotificationLists{
					Default: []*notification.Notification{
						{Type: "Email", Value: &notification.EmailNotification{Type: "Email", Email: "example@com"}},
					},
				},
			},
			Issues: nil,
		},
	} {
		tc.TestRead(t)
	}
}

func TestResourceUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[team.Team]{
		{
			Name: "no provider set",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &team.Team{},
			Expect:   nil,
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "failed update",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"PUT /v2/team/0001": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Failed to update", http.StatusBadRequest)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Expect: nil,
			Issues: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "route \"/v2/team/0001\" had issues with status code 400",
				},
			},
		},
		{
			Name: "successful update",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"PUT /v2/team/0001": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&team.Team{
						Id:          "0001",
						Name:        "test",
						Description: "test team",
						Members: []string{
							"a", "b", "c",
						},
					})
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Expect: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
				Members: []string{
					"a", "b", "c",
				},
			},
			Issues: nil,
		},
	} {
		tc.TestUpdate(t)
	}
}

func TestResourceDelete(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[team.Team]{
		{
			Name: "no provider set",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &team.Team{},
			Expect:   nil,
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "failed delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/team/0001": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Failed to delete", http.StatusBadRequest)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Expect: nil,
			Issues: diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "route \"/v2/team/0001\" had issues with status code 400",
				},
			},
		},
		{
			Name: "successful delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/team/0001": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					w.WriteHeader(http.StatusNoContent)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Expect: &team.Team{
				Id:          "0001",
				Name:        "test",
				Description: "test team",
			},
			Issues: nil,
		},
	} {
		tc.TestDelete(t)
	}
}
