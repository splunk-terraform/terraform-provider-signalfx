// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dimension

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/signalfx/signalfx-go/metrics_metadata"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestDataSource(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		meta   func(t testing.TB) any
		values []any
		diags  diag.Diagnostics
	}{
		{
			name: "no provider set",
			meta: func(testing.TB) any {
				return nil
			},
			values: []any{},
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			name: "invalid request",
			meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "unable to read dimensions", http.StatusBadRequest)
				},
			}),
			values: []any{},
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/dimension\" had issues with status code 400"},
			},
		},
		{
			name: "invalid request",
			meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "unable to read dimensions", http.StatusBadRequest)
				},
			}),
			values: []any{},
			diags: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/dimension\" had issues with status code 400"},
			},
		},
		{
			name: "exceeds limit",
			meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&metrics_metadata.DimensionQueryResponseModel{
						Count: 3,
						Results: []*metrics_metadata.Dimension{
							{Value: "my-awesome-service"},
							{Value: "my-other-service"},
							{Value: "my-main-service"},
						},
					})
				},
			}),
			values: []any{"my-awesome-service", "my-other-service"},
			diags: diag.Diagnostics{
				{
					Severity: diag.Warning,
					Summary:  "Number of matched results exceeds allowed returned limit, values truncated",
					Detail:   "Adjust the query to be more selective or increase the limit to avoid this issue",
				},
			},
		},
		{
			name: "returns dimension",
			meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/dimension": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					_ = json.NewEncoder(w).Encode(&metrics_metadata.DimensionQueryResponseModel{
						Count: 1,
						Results: []*metrics_metadata.Dimension{
							{Value: "my-awesome-service"},
						},
					})
				},
			}),
			values: []any{"my-awesome-service"},
			diags:  nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := NewDataSource()

			rd := data.TestResourceData()
			_ = rd.Set("query", "service.name:*")
			_ = rd.Set("limit", 2)

			diags := data.ReadContext(context.Background(), rd, tc.meta(t))
			assert.Equal(t, tc.diags, diags, "Must match the expected values")

			values := rd.Get("values").([]any)
			assert.Equal(t, tc.values, values, "Must match the expected values")
		})
	}
}
