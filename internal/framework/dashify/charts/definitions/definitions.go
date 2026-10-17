// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package definitions embeds the curated YAML chart definitions used by the interpreter.
package definitions

import "embed"

// Files is fixed at build time; definition loading never needs network access.
//
//go:embed *.yml common/*.yml
var Files embed.FS
