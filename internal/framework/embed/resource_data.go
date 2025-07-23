// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwembed

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	pmeta "github.com/splunk-terraform/terraform-provider-signalfx/internal/providermeta"
)

// ResourceData is an embeddable struct that provides common functionality for resources,
// since it implements the extended method required for [resource.ResourceWithConfigure].
type ResourceData struct {
	meta *pmeta.Meta
}

func (rd *ResourceData) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if meta, ok := req.ProviderData.(*pmeta.Meta); !ok {
		resp.Diagnostics.AddAttributeError(
			path.Empty(),
			"Missing Provider Data",
			"Provider data must be configured before using the resource.",
		)
	} else {
		rd.meta = meta
	}
}

func (rd *ResourceData) Details() *pmeta.Meta {
	return rd.meta
}
