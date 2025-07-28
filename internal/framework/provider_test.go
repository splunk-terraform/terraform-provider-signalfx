// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

func NewTestConfig(p provider.Provider, values map[string]tftypes.Value) tfsdk.Config {
	schema := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, schema)
	data := map[string]tftypes.Value{
		"auth_token":             tftypes.NewValue(tftypes.String, nil),
		"api_url":                tftypes.NewValue(tftypes.String, nil),
		"custom_app_url":         tftypes.NewValue(tftypes.String, nil),
		"timeout_seconds":        tftypes.NewValue(tftypes.Number, nil),
		"retry_max_attempts":     tftypes.NewValue(tftypes.Number, nil),
		"retry_wait_min_seconds": tftypes.NewValue(tftypes.Number, nil),
		"retry_wait_max_seconds": tftypes.NewValue(tftypes.Number, nil),
		"email":                  tftypes.NewValue(tftypes.String, nil),
		"password":               tftypes.NewValue(tftypes.String, nil),
		"organization_id":        tftypes.NewValue(tftypes.String, nil),
		"feature_preview":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.Bool}, nil),
		"tags":                   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"teams":                  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
	}
	maps.Copy(data, values)
	return tfsdk.Config{
		Schema: schema.Schema,
		Raw: tftypes.NewValue(
			tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"auth_token":             tftypes.String,
					"api_url":                tftypes.String,
					"custom_app_url":         tftypes.String,
					"timeout_seconds":        tftypes.Number,
					"retry_max_attempts":     tftypes.Number,
					"retry_wait_min_seconds": tftypes.Number,
					"retry_wait_max_seconds": tftypes.Number,
					"email":                  tftypes.String,
					"password":               tftypes.String,
					"organization_id":        tftypes.String,
					"feature_preview":        tftypes.Map{ElementType: tftypes.Bool},
					"tags":                   tftypes.List{ElementType: tftypes.String},
					"teams":                  tftypes.List{ElementType: tftypes.String},
				},
				OptionalAttributes: map[string]struct{}{
					"auth_token":             {},
					"api_url":                {},
					"custom_app_url":         {},
					"timeout_seconds":        {},
					"retry_max_attempts":     {},
					"retry_wait_min_seconds": {},
					"retry_wait_max_seconds": {},
					"email":                  {},
					"password":               {},
					"organization_id":        {},
					"feature_preview":        {},
					"tags":                   {},
					"teams":                  {},
				},
			},
			data,
		),
	}
}

func TestNewProvider(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewProvider("1.0.0", WithProviderFeatureRegistry(feature.NewRegistry())), "NewProvider should not return nil")
}

func TestProviderMetadata(t *testing.T) {
	t.Parallel()

	p := NewProvider("1.0.0")
	resp := &provider.MetadataResponse{}

	p.Metadata(context.Background(), provider.MetadataRequest{}, resp)

	assert.Equal(t, "signalfx", resp.TypeName, "TypeName should be 'signalfx'")
	assert.Equal(t, "1.0.0", resp.Version, "Version should be '1.0.0'")
}

func TestProviderSchema(t *testing.T) {
	t.Parallel()

	p := NewProvider("1.0.0")
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), provider.SchemaRequest{}, resp)

	assert.NotNil(t, resp.Schema, "Schema should not be nil")
}

func TestProviderDataSources(t *testing.T) {
	t.Parallel()

	p := NewProvider("1.0.0")

	assert.Empty(t, p.DataSources(context.Background()), "Must not return any values")
}

func TestProviderResource(t *testing.T) {
	t.Parallel()

	p := NewProvider("1.0.0")

	assert.Empty(t, p.Resources(context.Background()), "Must not return any values")
}

func TestProviderFunctions(t *testing.T) {
	t.Parallel()

	p := NewProvider("1.0.0")
	if fp, ok := p.(provider.ProviderWithFunctions); ok {
		assert.NotNil(t, fp.Functions(context.Background()), "ProviderWithFunctions should return non-nil functions")
	} else {
		assert.Fail(t, "Provider does not implement ProviderWithFunctions")
	}
}

func TestProviderConfigure(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		data   func(t *testing.T) map[string]tftypes.Value
		issues diag.Diagnostics
		expect *pmeta.Meta
	}{
		{
			name: "No data set",
			data: func(_ *testing.T) map[string]tftypes.Value {
				return map[string]tftypes.Value{}
			},
			issues: diag.Diagnostics{
				diag.NewErrorDiagnostic(
					"Issue configuring provider",
					"missing auth token or email and password; api url is not set",
				),
			},
			expect: nil,
		},
		{
			name: "Sets minimal required values",
			data: func(_ *testing.T) map[string]tftypes.Value {
				return map[string]tftypes.Value{
					"api_url":                tftypes.NewValue(tftypes.String, "http://localhost"),
					"auth_token":             tftypes.NewValue(tftypes.String, "my-secret-token"),
					"custom_app_url":         tftypes.NewValue(tftypes.String, nil),
					"timeout_seconds":        tftypes.NewValue(tftypes.Number, nil),
					"retry_max_attempts":     tftypes.NewValue(tftypes.Number, nil),
					"retry_wait_min_seconds": tftypes.NewValue(tftypes.Number, nil),
					"retry_wait_max_seconds": tftypes.NewValue(tftypes.Number, nil),
					"email":                  tftypes.NewValue(tftypes.String, nil),
					"password":               tftypes.NewValue(tftypes.String, nil),
					"organization_id":        tftypes.NewValue(tftypes.String, nil),
					"feature_preview":        tftypes.NewValue(tftypes.Map{ElementType: tftypes.Bool}, nil),
					"tags":                   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
					"teams":                  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
				}
			},
			issues: nil,
			expect: &pmeta.Meta{
				APIURL:    "http://localhost",
				AuthToken: "my-secret-token",
			},
		},
		{
			name: "Sets operational values",
			data: func(_ *testing.T) map[string]tftypes.Value {
				return map[string]tftypes.Value{
					"api_url":         tftypes.NewValue(tftypes.String, "http://localhost"),
					"auth_token":      tftypes.NewValue(tftypes.String, "my-secret-token"),
					"feature_preview": tftypes.NewValue(tftypes.Map{ElementType: tftypes.Bool}, nil),
					"tags": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "tag1"),
						tftypes.NewValue(tftypes.String, "tag2"),
					}),
					"teams": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "team1"),
						tftypes.NewValue(tftypes.String, "team2"),
					}),
				}
			},
			expect: &pmeta.Meta{
				Registry:  feature.GetGlobalRegistry(),
				APIURL:    "http://localhost",
				AuthToken: "my-secret-token",
				Tags:      []string{"tag1", "tag2"},
				Teams:     []string{"team1", "team2"},
			},
		},
		{
			name: "Sets operational values and feature preview",
			data: func(_ *testing.T) map[string]tftypes.Value {
				return map[string]tftypes.Value{
					"api_url":    tftypes.NewValue(tftypes.String, "http://localhost"),
					"auth_token": tftypes.NewValue(tftypes.String, "my-secret-token"),
					"feature_preview": tftypes.NewValue(tftypes.Map{ElementType: tftypes.Bool}, map[string]tftypes.Value{
						"new_feature": tftypes.NewValue(tftypes.Bool, true),
					}),
					"tags": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "tag1"),
						tftypes.NewValue(tftypes.String, "tag2"),
					}),
					"teams": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "team1"),
						tftypes.NewValue(tftypes.String, "team2"),
					}),
				}
			},
			issues: []diag.Diagnostic{
				diag.WithPath(
					path.Root("feature_preview").AtMapKey("new_feature"),
					diag.NewWarningDiagnostic("Failed to load feature preview", "no preview with id \"new_feature\" found"),
				),
			},
			expect: &pmeta.Meta{
				Registry:  feature.GetGlobalRegistry(),
				APIURL:    "http://localhost",
				AuthToken: "my-secret-token",
				Tags:      []string{"tag1", "tag2"},
				Teams:     []string{"team1", "team2"},
			},
		},
		{
			name: "Custom Domain is set",
			data: func(_ *testing.T) map[string]tftypes.Value {
				mux := http.NewServeMux()
				mux.HandleFunc("GET /v2/organization", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"url": "https://custom-domain.example.com"}`))
				})

				s := httptest.NewServer(mux)
				t.Cleanup(s.Close)
				return map[string]tftypes.Value{
					"api_url":         tftypes.NewValue(tftypes.String, s.URL),
					"auth_token":      tftypes.NewValue(tftypes.String, "my-secret-token"),
					"feature_preview": tftypes.NewValue(tftypes.Map{ElementType: tftypes.Bool}, nil),
					"tags": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "tag1"),
						tftypes.NewValue(tftypes.String, "tag2"),
					}),
					"teams": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "team1"),
						tftypes.NewValue(tftypes.String, "team2"),
					}),
				}
			},
			issues: nil,
			expect: &pmeta.Meta{
				Registry:     feature.GetGlobalRegistry(),
				APIURL:       "http://localhost",
				AuthToken:    "my-secret-token",
				CustomAppURL: "https://custom-domain.example.com",
				Tags:         []string{"tag1", "tag2"},
				Teams:        []string{"team1", "team2"},
			},
		},
		{
			name: "Custom Domain is provided from user config",
			data: func(_ *testing.T) map[string]tftypes.Value {
				return map[string]tftypes.Value{
					"api_url":         tftypes.NewValue(tftypes.String, "http://localhost"),
					"auth_token":      tftypes.NewValue(tftypes.String, "my-secret-token"),
					"custom_app_url":  tftypes.NewValue(tftypes.String, "https://my-provided.domain.signalfx.com"),
					"feature_preview": tftypes.NewValue(tftypes.Map{ElementType: tftypes.Bool}, nil),
					"tags": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "tag1"),
						tftypes.NewValue(tftypes.String, "tag2"),
					}),
					"teams": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
						tftypes.NewValue(tftypes.String, "team1"),
						tftypes.NewValue(tftypes.String, "team2"),
					}),
				}
			},
			issues: nil,
			expect: &pmeta.Meta{
				Registry:     feature.GetGlobalRegistry(),
				APIURL:       "http://localhost",
				AuthToken:    "my-secret-token",
				CustomAppURL: "https://my-provided.domain.signalfx.com",
				Tags:         []string{"tag1", "tag2"},
				Teams:        []string{"team1", "team2"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewProvider(
				t.Name(),
			)

			schema := &provider.SchemaResponse{}
			p.Schema(context.Background(), provider.SchemaRequest{}, schema)
			assert.NotNil(t, schema.Schema, "Schema should not be nil")

			resp := &provider.ConfigureResponse{}
			p.Configure(
				context.Background(),
				provider.ConfigureRequest{
					TerraformVersion: "1.0.0",
					Config:           NewTestConfig(p, tc.data(t)),
				},
				resp,
			)
			if !assert.Equal(t, tc.issues, resp.Diagnostics, "Diagnostics should match expected issues") {
				for _, issue := range resp.Diagnostics {
					t.Logf("Issue: %s - %s", issue.Summary(), issue.Detail())
				}
			}
			if tc.expect == nil {
				assert.Nil(t, resp.DataSourceData, "DataSourceData should be nil when expect is nil")
				assert.Nil(t, resp.ResourceData, "ResourceData should be nil when expect is nil")
				assert.Nil(t, resp.EphemeralResourceData, "EphemeralResourceData should be nil when expect is nil")
				return
			}
			assert.NotNil(t, resp.DataSourceData, "DataSourceData should not be nil when expect is set")
			assert.NotNil(t, resp.ResourceData, "ResourceData should not be nil when expect is set")
			assert.NotNil(t, resp.EphemeralResourceData, "EphemeralResourceData should not be nil when expect is set")

			meta := resp.DataSourceData.(*pmeta.Meta)
			assert.NotEmpty(t, meta.APIURL, "APIURL should not be empty")
			assert.Equal(t, tc.expect.AuthToken, meta.AuthToken, "AuthToken should match expected")
			assert.Equal(t, tc.expect.CustomAppURL, meta.CustomAppURL, "CustomAppURL should match expected")
			assert.Equal(t, tc.expect.Email, meta.Email, "Email should match expected")
			assert.Equal(t, tc.expect.Password, meta.Password, "Password should match expected")
			assert.Equal(t, tc.expect.OrganizationID, meta.OrganizationID, "OrganizationID should match expected")
			assert.Equal(t, tc.expect.Tags, meta.Tags, "Tags should match expected")
			assert.Equal(t, tc.expect.Teams, meta.Teams, "Teams should match expected")
		})
	}
}

func TestProviderValidateConfig(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		data   func(t *testing.T) map[string]tftypes.Value
		issues diag.Diagnostics
	}{
		{
			name: "No data set",
			data: func(_ *testing.T) map[string]tftypes.Value {
				return map[string]tftypes.Value{}
			},
			issues: diag.Diagnostics{
				diag.NewAttributeErrorDiagnostic(
					path.Root("api_url"),
					"Missing API Endpoint",
					"Field must be set to a valid endpoint for the Splunk Observability Cloud provider.",
				),
				diag.NewAttributeErrorDiagnostic(
					path.Empty().
						AtName("auth_token").
						AtName("email").
						AtName("password"),
					"Missing Authentication Method",
					"Either 'auth_token' or both 'email' and 'password' must be set for authentication.",
				),
			},
		},
		{
			name: "Minimal required values",
			data: func(_ *testing.T) map[string]tftypes.Value {
				return map[string]tftypes.Value{
					"api_url":    tftypes.NewValue(tftypes.String, "http://localhost"),
					"auth_token": tftypes.NewValue(tftypes.String, "my-secret-token"),
				}
			},
			issues: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := NewProvider("1.0.0")

			schema := &provider.SchemaResponse{}
			p.Schema(context.Background(), provider.SchemaRequest{}, schema)
			assert.NotNil(t, schema.Schema, "Schema should not be nil")

			resp := &provider.ValidateConfigResponse{}
			validator, ok := p.(provider.ProviderWithValidateConfig)
			require.True(t, ok, "Provider should implement ProviderWithValidateConfig")

			validator.ValidateConfig(
				context.Background(),
				provider.ValidateConfigRequest{
					Config: NewTestConfig(p, tc.data(t)),
				},
				resp,
			)

			assert.Equal(t, tc.issues, resp.Diagnostics, "Diagnostics should match expected issues")
		})
	}
}
