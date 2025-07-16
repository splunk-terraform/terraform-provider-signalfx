// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package autoarchivesettings

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

	for _, tc := range []tftest.ResourceOperationTestCase[automated_archival.AutomatedArchivalSettings]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &automated_archival.AutomatedArchivalSettings{},
			Expect:   &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Failed preconditions", http.StatusPreconditionFailed)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &automated_archival.AutomatedArchivalSettings{},
			Expect:   &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/automated-archival/settings\" had issues with status code 412"},
			},
		},
		{
			Name: "Successful create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					err := json.NewEncoder(w).Encode(&automated_archival.AutomatedArchivalSettings{
						Enabled:        true,
						LookbackPeriod: "P30D",
						GracePeriod:    "P30D",
					})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
			},
			Expect: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
			},
			Issues: nil,
		},
	} {
		tc.TestCreate(t)
	}
}

func TestResourceRead(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[automated_archival.AutomatedArchivalSettings]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &automated_archival.AutomatedArchivalSettings{},
			Expect:   &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Settings not available", http.StatusNotFound)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "1",
			Input: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
				Version:        1,
			},
			Expect: &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/automated-archival/settings\" had issues with status code 404"},
			},
		},
		{
			Name: "Successful read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					err := json.NewEncoder(w).Encode(&automated_archival.AutomatedArchivalSettings{
						Enabled:        true,
						LookbackPeriod: "P30D",
						GracePeriod:    "P30D",
						Version:        1,
					})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "1",
			Input: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
			},
			Expect: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
				Version:        1,
			},
			Issues: nil,
		},
	} {
		tc.TestRead(t)
	}
}

func TestResourceUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[automated_archival.AutomatedArchivalSettings]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &automated_archival.AutomatedArchivalSettings{},
			Expect:   &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed update",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"PUT /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Failed preconditions", http.StatusPreconditionFailed)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "1",
			Input: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
				Version:        1,
			},
			Expect: &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/automated-archival/settings\" had issues with status code 412"},
			},
		},
		{
			Name: "Successful update",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"PUT /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					err := json.NewEncoder(w).Encode(&automated_archival.AutomatedArchivalSettings{
						Enabled:        true,
						LookbackPeriod: "P30D",
						GracePeriod:    "P30D",
						Version:        1,
					})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "1",
			Input: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
			},
			Expect: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
				Version:        1,
			},
			Issues: nil,
		},
	} {
		tc.TestUpdate(t)
	}
}

func TestResourceDelete(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[automated_archival.AutomatedArchivalSettings]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &automated_archival.AutomatedArchivalSettings{},
			Expect:   &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					http.Error(w, "Failed preconditions", http.StatusPreconditionFailed)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "1",
			Input: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
				Version:        1,
			},
			Expect: &automated_archival.AutomatedArchivalSettings{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/automated-archival/settings\" had issues with status code 412"},
			},
		},
		{
			Name: "Successful delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/automated-archival/settings": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()
					w.WriteHeader(http.StatusNoContent)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "1",
			Input: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
				Version:        1,
			},
			Expect: &automated_archival.AutomatedArchivalSettings{
				Enabled:        true,
				LookbackPeriod: "P30D",
				GracePeriod:    "P30D",
				Version:        1,
			},
			Issues: nil,
		},
	} {
		tc.TestDelete(t)
	}
}
