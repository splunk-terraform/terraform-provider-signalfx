// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/signalfx/signalfx-go"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

type MockProvider struct {
	data *pmeta.Meta

	resources   []func() resource.Resource
	datasources []func() datasource.DataSource
}

var (
	_ provider.Provider = (*MockProvider)(nil)
)

func WithMockResources(resources ...func() resource.Resource) func(*MockProvider) {
	return func(mp *MockProvider) {
		mp.resources = resources
	}
}

func WithMockDataSources(datasources ...func() datasource.DataSource) func(*MockProvider) {
	return func(mp *MockProvider) {
		mp.datasources = datasources
	}
}

func NewMockProviderFactory(tb testing.TB, endpoints map[string]http.Handler, opts ...func(*MockProvider)) map[string]func() (tfprotov5.ProviderServer, error) {
	mux := http.NewServeMux()
	for path, handler := range endpoints {
		mux.Handle(path, handler)
	}
	// The pattern matchers will match based on the longest prefix matching
	// so this acts to help identify unmatched paths and will force the test
	// to fail so it the behaviour is not dependant on.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tb.Log("Unhandled request:", r.Method, r.URL.Path)

		http.Error(w, "Internal Test Error: "+tb.Name(), http.StatusInternalServerError)

		tb.Fail()
	})

	s := httptest.NewServer(mux)
	tb.Cleanup(s.Close)

	client, _ := signalfx.NewClient(
		tb.Name(),
		signalfx.HTTPClient(s.Client()),
		signalfx.APIUrl(s.URL),
	)

	mock := &MockProvider{
		data: &pmeta.Meta{
			Client:       client,
			APIURL:       s.URL,
			CustomAppURL: s.URL,
		},
	}

	for _, opt := range opts {
		opt(mock)
	}

	return map[string]func() (tfprotov5.ProviderServer, error){
		"signalfx": func() (tfprotov5.ProviderServer, error) { //nolint:unparam // required signature for provider server
			return providerserver.NewProtocol5(mock)(), nil
		},
	}
}

func (mp MockProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "signalfx"
	resp.Version = "1.0.0"
}

func (mp MockProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This is a mock provider for testing purposes.",
	}
}

func (mp MockProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = mp.data
	resp.DataSourceData = mp.data
	resp.EphemeralResourceData = mp.data
}

func (mp MockProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return mp.datasources
}

func (mp MockProvider) Resources(ctx context.Context) []func() resource.Resource {
	return mp.resources
}
