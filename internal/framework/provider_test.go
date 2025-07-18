// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/stretchr/testify/assert"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"
)

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

func TestProviderConfigure(t *testing.T) {
	t.Parallel()

	p := NewProvider("1.0.0")
	schema := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, schema)

	resp := &provider.ConfigureResponse{}

	p.Configure(
		context.Background(),
		provider.ConfigureRequest{
			Config: tfsdk.Config{
				Schema: schema.Schema,
			},
		},
		resp,
	)

	assert.NotEmpty(t, resp.Diagnostics, "ConfigureResponse should not have any diagnostics")
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
