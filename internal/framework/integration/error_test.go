// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwintegration

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/signalfx/signalfx-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithAdminTokenHelp(t *testing.T) {
	t.Parallel()

	clientError := responseError(t, http.StatusUnauthorized)
	serverError := responseError(t, http.StatusBadGateway)
	plainError := errors.New("plain error")

	tests := []struct {
		name     string
		err      error
		expected error
		help     bool
	}{
		{name: "nil", err: nil, expected: nil},
		{name: "client response", err: clientError, expected: clientError, help: true},
		{name: "server response", err: serverError, expected: serverError},
		{name: "non response", err: plainError, expected: plainError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			actual := withAdminTokenHelp(test.err)
			assert.ErrorIs(t, actual, test.expected)
			if test.help {
				assert.ErrorContains(t, actual, adminTokenHelp)
			} else if actual != nil {
				assert.NotContains(t, actual.Error(), adminTokenHelp)
			}
		})
	}
}

func responseError(t *testing.T, status int) error {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(status), status)
	}))
	t.Cleanup(server.Close)

	client, err := signalfx.NewClient("token", signalfx.APIUrl(server.URL), signalfx.HTTPClient(server.Client()))
	require.NoError(t, err)

	_, err = client.GetOpsgenieIntegration(t.Context(), "test-id")
	require.Error(t, err)
	return err
}
