// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build tools

package tools

// This file follows the recommendation at
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// on how to pin tooling dependencies to a go.mod file.
// This ensures that all systems use the same version of tools in addition to regular dependencies.

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/google/addlicense"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
