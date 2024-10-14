// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func TestNewHTTPMockStat(t *testing.T) {
	t.Parallel()

	fn := NewTestHTTPMockMeta(map[string]http.HandlerFunc{
		"/v2/team/001": func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()

			w.WriteHeader(http.StatusNoContent)
		},
	})

	meta := fn(t)

	client, err := pmeta.LoadClient(context.Background(), meta)
	require.NoError(t, err, "Must not error loading client")

	require.NoError(t, client.DeleteTeam(context.Background(), "001"), "Must not error removing team")
	require.Error(t, client.DeleteTeam(context.Background(), "002"), "Must error trying to make request to endpoint not defined in mock")
}

func TestNewAcceptanceConfigure(t *testing.T) {

	for _, tc := range []struct {
		name   string
		envs   map[string]string
		issues diag.Diagnostics
	}{
		{
			name: "no values set",
			envs: map[string]string{},
			issues: diag.Diagnostics{
				{Severity: diag.Error, Summary: "auth token not set"},
				{Severity: diag.Error, Summary: "api url is not set"},
			},
		},
		{
			name: "configured environment vars",
			envs: map[string]string{
				"SFX_AUTH_TOKEN": "mytoken",
				"SFX_API_URL":    "https://localhost",
			},
			issues: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			CleanEnvVars(t)

			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			actual, issues := newAcceptanceConfigure(context.Background(), &schema.ResourceData{})
			if len(tc.issues) == 0 {
				assert.NotNil(t, actual, "Must have a valid meta object returned")
				assert.Empty(t, issues)
			} else {
				assert.Nil(t, actual, "Must have a valid meta object returned")
				assert.Equal(t, tc.issues, issues, "Must match the expected value")
			}
		})
	}
}
