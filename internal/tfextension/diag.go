// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import (
	"slices"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

// AppendDiagnostics is to be used similar to combining errors together
// so they are reported at the same time to provide all details at once.
func AppendDiagnostics(diags diag.Diagnostics, values ...diag.Diagnostic) diag.Diagnostics {
	return slices.Concat(diags, values)
}

// AsErrorDiagnostics is the same as `diag.FromErr`, however, it allow allows
// adding the attribute values that are provided in CRUD operations.
func AsErrorDiagnostics(err error, path ...cty.Path) diag.Diagnostics {
	return newDiagnostics(diag.Error, err, path...)
}

// AsWarnDiagnostics is the same as `diag.FromErr`, however, it sets the severity as Warning
// and allows for appending the attribute path as part of the values.
func AsWarnDiagnostics(err error, path ...cty.Path) diag.Diagnostics {
	return newDiagnostics(diag.Warning, err, path...)
}

func newDiagnostics(sev diag.Severity, summary error, path ...cty.Path) diag.Diagnostics {
	if summary == nil {
		return nil
	}
	return diag.Diagnostics{
		{Severity: sev, Summary: summary.Error(), AttributePath: slices.Concat(path...)},
	}
}
