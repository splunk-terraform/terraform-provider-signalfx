// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	internalframework "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework"
	"github.com/splunk-terraform/terraform-provider-signalfx/signalfx"
)

const (
	ProviderRegistry = "registry.terraform.io/splunk-terraform/signalfx"
)

var (
	Version = "dev"

	debug = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	upgrader, err := tf5to6server.UpgradeServer(
		context.Background(),
		signalfx.Provider().GRPCProvider, // Provider to be sunset during the migration of 10.x
	)

	if err != nil {
		log.Fatal(err)
	}

	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(internalframework.NewProvider(Version)),
		func() tfprotov6.ProviderServer {
			return upgrader
		},
	}

	mux, err := tf6muxserver.NewMuxServer(context.Background(), providers...)
	if err != nil {
		log.Fatal(err)
	}

	var opts []tf6server.ServeOpt
	if *debug {
		opts = append(opts, tf6server.WithManagedDebug())
	}

	if err = tf6server.Serve(ProviderRegistry, mux.ProviderServer, opts...); err != nil {
		log.Fatal(err)
	}
}
