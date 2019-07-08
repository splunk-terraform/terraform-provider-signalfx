package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/terraform-providers/terraform-provider-signalfx/signalfx"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: signalfx.Provider,
	})
}
