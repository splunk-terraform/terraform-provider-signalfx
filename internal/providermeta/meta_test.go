// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package pmeta

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/signalfx/signalfx-go/sessiontoken"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
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
	}{
		{
			name:      "no meta set",
			meta:      nil,
			fragments: []string{},
			url:       "",
		},
		{
			name: "custom domain set",
			meta: &Meta{
				CustomAppURL: "http://custom.signalfx.com",
			},
			fragments: []string{},
			url:       "http://custom.signalfx.com/",
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
			url: "http://custom.signalfx.com/#detector/aaaa/edit",
		},
		{
			name: "invalid domain set",
			meta: &Meta{
				CustomAppURL: "domain",
			},
			fragments: []string{},
			url:       "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u := LoadApplicationURL(context.Background(), tc.meta, tc.fragments...)
			require.Equal(t, tc.url, u, "Must match the expected url")
		})
	}
}

func TestLoadPreviewRegistry(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		meta   any
		expect *feature.Registry
	}{
		{
			name:   "no meta set",
			meta:   nil,
			expect: feature.GetGlobalRegistry(),
		},
		{
			name:   "no local registry set",
			meta:   &Meta{},
			expect: feature.GetGlobalRegistry(),
		},
		{
			name: "empty local registry",
			meta: &Meta{
				Registry: &feature.Registry{},
			},
			expect: &feature.Registry{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := LoadPreviewRegistry(context.Background(), tc.meta)
			assert.Equal(t, tc.expect, r, "Must match the expected registry")
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
			errVal: "missing auth token or email and password; api url is not set",
		},
		{
			name: "state valid",
			meta: Meta{
				AuthToken: "aaa",
				APIURL:    "http://api.signalfx.com",
			},
		},
		{
			name: "Email only provided",
			meta: Meta{
				Email:  "example@com",
				APIURL: "http://api.signalfx.com",
			},
			errVal: "missing auth token or email and password",
		},
		{
			name: "password only provided",
			meta: Meta{
				Password: "derp",
				APIURL:   "http://api.signalfx.com",
			},
			errVal: "missing auth token or email and password",
		},
	} {

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

func TestMetaToken(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		token    string
		handler  http.HandlerFunc
		email    string
		password string
		expect   string
		errVal   string
	}{
		{
			name:  "missing values",
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				http.Error(w, "failed auth", http.StatusBadRequest)
			},
			email:    "",
			password: "",
			expect:   "",
			errVal:   "route \"/v2/session\" had issues with status code 400",
		},
		{
			name:  "token already provided",
			token: "aaccbbb",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				http.Error(w, "should not be called", http.StatusBadRequest)
			},
			email:    "",
			password: "",
			expect:   "aaccbbb",
			errVal:   "",
		},
		{
			name:  "username password provided",
			token: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				_ = json.NewEncoder(w).Encode(&sessiontoken.Token{AccessToken: "secret"})
			},
			email:    "user@example",
			password: "notsosecret",
			expect:   "secret",
			errVal:   "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := httptest.NewServer(tc.handler)
			t.Cleanup(s.Close)

			m := &Meta{
				APIURL:    s.URL,
				AuthToken: tc.token,
				Email:     tc.email,
				Password:  tc.password,
			}

			if token, err := m.LoadSessionToken(context.Background()); tc.errVal != "" {
				assert.Equal(t, tc.expect, token, "Must match the expected value")
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.Equal(t, tc.expect, token, "Must match the expected value")
				assert.NoError(t, err, "Must not error")
			}
		})
	}
}

func TestLoadProviderTags(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		meta   any
		expect []string
	}{
		{
			name:   "no provider set",
			meta:   nil,
			expect: nil,
		},
		{
			name:   "incorrect provider type set",
			meta:   "provider",
			expect: nil,
		},
		{
			name: "disabled provider",
			meta: &Meta{
				Registry: func() *feature.Registry {
					r := feature.NewRegistry()
					r.MustRegister(feature.PreviewProviderTags)
					return r
				}(),
				Tags: []string{
					"example",
					"test",
				},
			},
			expect: nil,
		},
		{
			name: "disabled provider",
			meta: &Meta{
				Registry: func() *feature.Registry {
					r := feature.NewRegistry()
					r.MustRegister(feature.PreviewProviderTags, feature.WithPreviewGlobalAvailable())
					return r
				}(),
				Tags: []string{
					"example",
					"test",
				},
			},
			expect: []string{
				"example",
				"test",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				tc.expect,
				LoadProviderTags(t.Context(), tc.meta),
			)
		})
	}
}

func TestMergeProviderTeams(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		meta   any
		teams  []string
		expect []string
	}{
		{
			name:   "no details provided",
			meta:   nil,
			teams:  nil,
			expect: nil,
		},
		{
			name: "provide has details, preview not enabled",
			meta: &Meta{
				Registry: func() *feature.Registry {
					r := feature.NewRegistry()
					_ = r.MustRegister(feature.PreviewProviderTeams)
					return r
				}(),
				Teams: []string{"team-00", "team-02", "team-03"},
			},
			teams:  []string{"team-01", "team-02"},
			expect: []string{"team-01", "team-02"},
		},
		{
			name: "provider has details, preview enabled",
			meta: &Meta{
				Registry: func() *feature.Registry {
					r := feature.NewRegistry()
					_ = r.MustRegister(feature.PreviewProviderTeams, feature.WithPreviewGlobalAvailable())
					return r
				}(),
				Teams: []string{"team-00", "team-02", "team-03"},
			},
			teams:  []string{"team-01", "team-02"},
			expect: []string{"team-00", "team-02", "team-03", "team-01"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				tc.expect,
				MergeProviderTeams(t.Context(), tc.meta, tc.teams),
				"Must match the expected details",
			)
		})
	}
}

func TestDetectCustomAPPURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		expect  string
		errVal  string
	}{
		{
			name: "not authorized",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				http.Error(w, "not authorized", http.StatusUnauthorized)
			},
			expect: "",
			errVal: "failed fetching organization details",
		},
		{
			name: "missing content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				_ = json.NewEncoder(w).Encode(map[string]any{})
			},
			expect: "",
			errVal: "",
		},
		{
			name: "custom domain configured",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				_ = json.NewEncoder(w).Encode(map[string]any{
					"url": "https://custom.signalfx.com",
				})
			},
			expect: "https://custom.signalfx.com",
			errVal: "",
		},
		{
			name: "invalid json content suppplied",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()

				_, _ = w.Write([]byte("{"))
			},
			expect: "",
			errVal: "unexpected EOF",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := httptest.NewServer(tc.handler)
			t.Cleanup(s.Close)

			m := &Meta{
				APIURL: s.URL,
			}

			domain, err := m.DetectCustomAPPURL(t.Context())
			assert.Equal(t, tc.expect, domain, "Must match the expected value")
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected error")
			} else {
				assert.NoError(t, err, "Must not error when detecting custom app URL")
			}
		})
	}
}

func TestMetaDetectorCustomAppURL_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("invalid api name", func(t *testing.T) {
		m := &Meta{
			APIURL: "\tinvalid",
		}

		_, err := m.DetectCustomAPPURL(context.Background())
		assert.Error(t, err, "Must return an error when api URL is invalid")
	})
}
