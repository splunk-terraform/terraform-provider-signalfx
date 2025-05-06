// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package organization

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/signalfx/signalfx-go/organization"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestNewDataSource(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewDataSource(), "Must have a valid resource returned")
}

func TestDataSourceRead(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		meta   func(tb testing.TB) any
		values []any
		diags  diag.Diagnostics
	}{
		{
			name: "no provider",
			meta: func(_ testing.TB) any {
				return nil
			},
			values: nil,
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			name: "not authorized",
			meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/organization/member": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "failed to read", http.StatusUnauthorized)
				},
			}),
			values: []any{
				"user-01@example.com",
			},
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/organization/member\" had issues with status code 401"},
			},
		},
		{
			name: "no emails",
			meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/organization/member": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&organization.MemberSearchResults{
						Count: 0,
					})
				},
			}),
			values: []any{
				"user-01@example.com",
			},
			diags: nil,
		},
		{
			name: "no emails",
			meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/organization/member": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&organization.MemberSearchResults{
						Count: 1,
						Results: []*organization.Member{
							{FullName: "user 01", Id: "AAAAAAAA", Email: "user-01@example.com"},
						},
					})
				},
			}),
			values: []any{
				"user-01@example.com",
			},
			diags: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := NewDataSource()

			rd := data.TestResourceData()
			assert.NoError(t, rd.Set("emails", tc.values))

			actual := data.ReadContext(t.Context(), rd, tc.meta(t))
			assert.Equal(t, tc.diags, actual, "Must match the expected results")
		})
	}
}
