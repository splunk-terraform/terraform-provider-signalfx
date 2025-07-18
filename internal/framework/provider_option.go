// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package internalframework

import "github.com/splunk-terraform/terraform-provider-signalfx/internal/feature"

type ProviderOption func(*ollyProvider)

func WithProviderFeatureRegistry(reg *feature.Registry) ProviderOption {
	return func(p *ollyProvider) {
		p.features = reg
	}
}
