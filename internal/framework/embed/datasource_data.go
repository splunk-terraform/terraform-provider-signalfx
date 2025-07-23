// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwembed

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

// DatasourceData is an embeddable struct that provides common functionality for Datasource,
// since it implements the extended method required for [Datasource.DatasourceWithConfigure].
type DatasourceData struct {
	meta *pmeta.Meta
}

func (dd *DatasourceData) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if meta, ok := req.ProviderData.(*pmeta.Meta); !ok {
		resp.Diagnostics.AddAttributeError(
			path.Empty(),
			"Missing Provider Data",
			"Provider data must be configured before using the datasource.",
		)
	} else {
		dd.meta = meta
	}
}

func (dd *DatasourceData) Details() *pmeta.Meta {
	return dd.meta
}
