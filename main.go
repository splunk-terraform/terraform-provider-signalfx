// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/splunk-terraform/terraform-provider-signalfx/signalfx"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: signalfx.Provider,
	})
}
