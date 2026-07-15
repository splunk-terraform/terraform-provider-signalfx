// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalframework "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework"
	"github.com/splunk-terraform/terraform-provider-signalfx/signalfx"
)

func TestMuxProviderTypeOwnership(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	framework := internalframework.NewProvider("test")
	legacy := signalfx.Provider()

	resourceNames := make(map[string]struct{})
	for _, factory := range framework.Resources(ctx) {
		resp := &resource.MetadataResponse{}
		factory().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)
		require.NotEmpty(t, resp.TypeName)
		assert.NotContains(t, resourceNames, resp.TypeName, "Framework resource type must be unique")
		assert.NotContains(t, legacy.ResourcesMap, resp.TypeName, "Resource type cannot be owned by both mux providers")
		resourceNames[resp.TypeName] = struct{}{}
	}

	dataSourceNames := make(map[string]struct{})
	for _, factory := range framework.DataSources(ctx) {
		resp := &datasource.MetadataResponse{}
		factory().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "signalfx"}, resp)
		require.NotEmpty(t, resp.TypeName)
		assert.NotContains(t, dataSourceNames, resp.TypeName, "Framework data source type must be unique")
		assert.NotContains(t, legacy.DataSourcesMap, resp.TypeName, "Data source type cannot be owned by both mux providers")
		dataSourceNames[resp.TypeName] = struct{}{}
	}
}
