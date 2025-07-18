// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"

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

	providers := []func() tfprotov5.ProviderServer{
		providerserver.NewProtocol5(internalframework.NewProvider(Version)),
		signalfx.Provider().GRPCProvider, // Provider to be sunset during the migration of 10.x
	}

	mux, err := tf5muxserver.NewMuxServer(context.Background(), providers...)
	if err != nil {
		log.Fatal(err)
	}

	var opts []tf5server.ServeOpt
	if *debug {
		opts = append(opts, tf5server.WithManagedDebug())
	}

	if err = tf5server.Serve(ProviderRegistry, mux.ProviderServer, opts...); err != nil {
		log.Fatal(err)
	}
}
