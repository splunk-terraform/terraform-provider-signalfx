// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"fmt"
	"time"
	_ "time/tzdata" // Importing time zone database to ensure there is failover option

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

func TimeZoneLocation() schema.SchemaValidateDiagFunc {
	return func(i interface{}, p cty.Path) diag.Diagnostics {
		tz, ok := i.(string)
		if !ok {
			return tfext.AsErrorDiagnostics(
				fmt.Errorf("expected %v as string", i),
				p,
			)
		}
		_, err := time.LoadLocation(tz)
		return tfext.AsErrorDiagnostics(err, p)

	}
}
