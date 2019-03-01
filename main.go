package main

import (
	"github.com/signalfx/terraform-provider-signalfx/signalfx"

	"github.com/hashicorp/terraform/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: signalfx.Provider,
	})
}
