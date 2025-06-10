// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package autoarchiveexemptmetric

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	automated_archival "github.com/signalfx/signalfx-go/automated-archival"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestNewResource(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewResource(), "Must return a valid value")
}

func TestResourceCreate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[[]automated_archival.ExemptMetric]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &[]automated_archival.ExemptMetric{},
			Expect:   &[]automated_archival.ExemptMetric{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST /v2/automated-archival/exempt-metrics": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Failed preconditions", http.StatusPreconditionFailed)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &[]automated_archival.ExemptMetric{
				{
					Name: "metric1",
				},
				{
					Name: "metric2",
				},
			},
			Expect: &[]automated_archival.ExemptMetric{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/automated-archival/exempt-metrics\" had issues with status code 412"},
			},
		},
		{
			Name: "Successful create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST /v2/automated-archival/exempt-metrics": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					id1 := "id-metric-1"
					id2 := "id-metric-2"
					err := json.NewEncoder(w).Encode(&[]automated_archival.ExemptMetric{
						{
							Name: "metric1",
							Id:   &id1,
						},
						{
							Name: "metric2",
							Id:   &id2,
						},
					})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &[]automated_archival.ExemptMetric{
				{
					Name: "metric1",
				},
				{
					Name: "metric2",
				},
			},
			Expect: &[]automated_archival.ExemptMetric{{Name: "metric1"}, {Name: "metric2"}},
			Issues: nil,
		},
	} {
		tc.TestCreate(t)
	}
}

func TestResourceRead(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[[]automated_archival.ExemptMetric]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &[]automated_archival.ExemptMetric{},
			Expect:   &[]automated_archival.ExemptMetric{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/automated-archival/exempt-metrics": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Failed preconditions", http.StatusPreconditionFailed)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &[]automated_archival.ExemptMetric{
				{
					Name: "metric1",
				},
			},
			Expect: &[]automated_archival.ExemptMetric{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/automated-archival/exempt-metrics\" had issues with status code 412"},
			},
		},
		{
			Name: "Successful read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/automated-archival/exempt-metrics": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					id1 := "id-metric-1"
					id2 := "id-metric-2"
					err := json.NewEncoder(w).Encode(&[]automated_archival.ExemptMetric{
						{
							Name: "metric1",
							Id:   &id1,
						},
						{
							Name: "metric2",
							Id:   &id2,
						},
					})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &[]automated_archival.ExemptMetric{},
			Expect:   &[]automated_archival.ExemptMetric{{Name: "metric1"}, {Name: "metric2"}},
			Issues:   nil,
		},
	} {
		tc.TestRead(t)
	}
}

func TestResourceDelete(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[[]automated_archival.ExemptMetric]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &[]automated_archival.ExemptMetric{},
			Expect:   &[]automated_archival.ExemptMetric{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/automated-archival/exempt-metrics": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Failed preconditions", http.StatusPreconditionFailed)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "id-metric-1,id-metric-2",
			Input: &[]automated_archival.ExemptMetric{
				{
					Name: "metric1",
				},
				{
					Name: "metric2",
				},
			},
			Expect: &[]automated_archival.ExemptMetric{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/automated-archival/exempt-metrics\" had issues with status code 412"},
			},
		},
		{
			Name: "Successful delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/automated-archival/exempt-metrics": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					w.WriteHeader(http.StatusNoContent)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "id-metric-1,id-metric-2",
			Input: &[]automated_archival.ExemptMetric{
				{
					Name: "metric1",
				},
				{
					Name: "metric2",
				},
			},
			Expect: &[]automated_archival.ExemptMetric{
				{Name: "metric1"},
				{Name: "metric2"},
			},
			Issues: nil,
		},
	} {
		tc.TestDelete(t)
	}
}
