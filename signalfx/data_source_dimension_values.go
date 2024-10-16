// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package signalfx

import (
	"github.com/splunk-terraform/terraform-provider-signalfx/internal/definition/dimension"
)

// Deprecated: Use `dimension.NewDataSource` instead
var dataSourceDimensionValues = dimension.NewDataSource
