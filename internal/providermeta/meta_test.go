// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package pmeta

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadClient(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		meta any
		err  error
	}{
		{
			name: "meta not set",
			meta: nil,
			err:  ErrMetaNotProvided,
		},
		{
			name: "meta defined",
			meta: &Meta{},
			err:  nil,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := LoadClient(context.Background(), tc.meta)
			require.ErrorIs(t, err, tc.err, "Must match the expected error value")
		})
	}
}

func TestLoadApplicationURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		meta      any
		fragments []string
		url       string
		errVal    string
	}{
		{
			name:      "no meta set",
			meta:      nil,
			fragments: []string{},
			url:       "",
			errVal:    "expected to implement type Meta",
		},
		{
			name: "custom domain set",
			meta: &Meta{
				CustomAppURL: "http://custom.signalfx.com",
			},
			fragments: []string{},
			url:       "http://custom.signalfx.com/",
			errVal:    "",
		},
		{
			name: "custom domain with fragments",
			meta: &Meta{
				CustomAppURL: "http://custom.signalfx.com",
			},
			fragments: []string{
				"detector",
				"aaaa",
				"edit",
			},
			url:    "http://custom.signalfx.com/#detector/aaaa/edit",
			errVal: "",
		},
		{
			name: "invalid domain set",
			meta: &Meta{
				CustomAppURL: "domain",
			},
			fragments: []string{},
			url:       "",
			errVal:    "parse \"domain\": invalid URI for request",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u, err := LoadApplicationURL(context.Background(), tc.meta, tc.fragments...)
			require.Equal(t, tc.url, u, "Must match the expected url")
			if tc.errVal != "" {
				require.EqualError(t, err, tc.errVal, "Must match expected error message")
			} else {
				require.NoError(t, err, "Must not error when loading url")
			}
		})
	}
}

func TestMetaValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		meta   Meta
		errVal string
	}{
		{
			name:   "meta not set",
			meta:   Meta{},
			errVal: "auth token not set; api url is not set",
		},
		{
			name: "state valid",
			meta: Meta{
				AuthToken: "aaa",
				APIURL:    "http://api.signalfx.com",
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.meta.Validate(); tc.errVal != "" {
				require.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				require.NoError(t, err, "Must not error when validation")
			}
		})
	}
}
