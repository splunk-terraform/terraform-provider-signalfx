// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package orgtoken

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/signalfx/signalfx-go/orgtoken"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/tftest"
)

func TestNewResource(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewResource(), "Must return a valid value")
}

func TestResourceCreate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[orgtoken.Token]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &orgtoken.Token{},
			Expect:   &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST /v2/token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Failed preconditions", http.StatusPreconditionFailed)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &orgtoken.Token{},
			Expect:   &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/token\" had issues with status code 412"},
			},
		},
		{
			Name: "Successful create",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"POST /v2/token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					err := json.NewEncoder(w).Encode(&orgtoken.Token{
						Name: "simple-token",
					})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input: &orgtoken.Token{
				Name: "simple-token",
			},
			Expect: &orgtoken.Token{
				Name:   "simple-token",
				Limits: &orgtoken.Limit{},
			},
			Issues: nil,
		},
	} {
		tc.TestCreate(t)
	}
}

func TestResourceRead(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[orgtoken.Token]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &orgtoken.Token{},
			Expect:   &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/token/my-token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Token not available", http.StatusNotFound)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "my-token",
			Input: &orgtoken.Token{
				Name: "my-token",
			},
			Expect: &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/token/my-token\" had issues with status code 404"},
			},
		},
		{
			Name: "Successful read",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"GET /v2/token/my-token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					err := json.NewEncoder(w).Encode(&orgtoken.Token{
						Name:        "my-token",
						Description: "This is my token, no other will do",
						AuthScopes: []string{
							"API",
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
			ID:       "my-token",
			Input: &orgtoken.Token{
				Name: "my-token",
			},
			Expect: &orgtoken.Token{
				Name:        "my-token",
				Description: "This is my token, no other will do",
				AuthScopes:  []string{"API"},
				Limits:      &orgtoken.Limit{},
			},
			Issues: nil,
		},
	} {
		tc.TestRead(t)
	}
}

func TestResourceUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[orgtoken.Token]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &orgtoken.Token{},
			Expect:   &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed update",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"PUT /v2/token/my-token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Unable to progress", http.StatusInsufficientStorage)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "my-token",
			Input: &orgtoken.Token{
				Name: "my-token",
			},
			Expect: &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/token/my-token\" had issues with status code 507"},
			},
		},
		{
			Name: "Successful update",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"PUT /v2/token/my-token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					err := json.NewEncoder(w).Encode(&orgtoken.Token{
						Name:        "my-token",
						Description: "Everyone's token",
					})
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "my-token",
			Input: &orgtoken.Token{
				Name:        "my-token",
				Description: "My token...",
			},
			Expect: &orgtoken.Token{
				Name:        "my-token",
				Description: "Everyone's token",
				Limits:      &orgtoken.Limit{},
			},
			Issues: nil,
		},
	} {
		tc.TestUpdate(t)
	}
}

func TestResourceDelete(t *testing.T) {
	t.Parallel()

	for _, tc := range []tftest.ResourceOperationTestCase[orgtoken.Token]{
		{
			Name: "No provider",
			Meta: func(_ testing.TB) any {
				return nil
			},
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			Input:    &orgtoken.Token{},
			Expect:   &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "expected to implement type Meta"},
			},
		},
		{
			Name: "Failed delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/token/my-token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					http.Error(w, "Not Authed", http.StatusNotAcceptable)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "my-token",
			Input: &orgtoken.Token{
				Name: "my-token",
			},
			Expect: &orgtoken.Token{},
			Issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "route \"/v2/token/my-token\" had issues with status code 406"},
			},
		},
		{
			Name: "Succesful delete",
			Meta: tftest.NewTestHTTPMockMeta(map[string]http.HandlerFunc{
				"DELETE /v2/token/my-token": func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(io.Discard, r.Body)
					_ = r.Body.Close()

					w.WriteHeader(http.StatusNoContent)
				},
			}),
			Resource: NewResource(),
			Encoder:  encodeTerraform,
			Decoder:  decodeTerraform,
			ID:       "my-token",
			Input: &orgtoken.Token{
				Name: "my-token",
			},
			Expect: &orgtoken.Token{
				Name:   "my-token",
				Limits: &orgtoken.Limit{},
			},
			Issues: nil,
		},
	} {
		tc.TestDelete(t)
	}
}
