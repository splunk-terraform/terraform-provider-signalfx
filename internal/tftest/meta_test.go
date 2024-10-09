// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"context"
	"io"
	"net/http"
	"testing"

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
