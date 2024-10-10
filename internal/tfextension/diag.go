// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import (
	"slices"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"go.uber.org/multierr"
)

// AppendDiagnostics is to be used similar to combining errors together
// so they are reported at the same time to provide all details at once.
func AppendDiagnostics(diags diag.Diagnostics, values ...diag.Diagnostic) diag.Diagnostics {
	return slices.Concat(diags, values)
}

// AsErrorDiagnostics is the same as `diag.FromErr`, however, it allow allows
// adding the attribute values that are provided in CRUD operations.
func AsErrorDiagnostics(err error, path ...cty.Path) (issues diag.Diagnostics) {
	return newUnwrapErrors(diag.Error, err, path...)
}

// AsWarnDiagnostics is the same as `diag.FromErr`, however, it sets the severity as Warning
// and allows for appending the attribute path as part of the values.
func AsWarnDiagnostics(err error, path ...cty.Path) (issues diag.Diagnostics) {
	return newUnwrapErrors(diag.Warning, err, path...)
}

func newUnwrapErrors(sev diag.Severity, err error, path ...cty.Path) (issues diag.Diagnostics) {
	if err == nil {
		return nil
	}

	// Checking to see if there is any joined errors
	// so it can be unpacked into separate reported issues.
	// This useses the unpublished errors' [interface{ Unwrap() []error }]
	// and if that is unset it then checks the uber's implementation.
	var errs []error
	if v, ok := err.(interface{ Unwrap() []error }); ok {
		errs = v.Unwrap()
	}

	if len(errs) == 0 {
		errs = multierr.Errors(err)
	}

	for _, err := range errs {
		issues = AppendDiagnostics(issues, diag.Diagnostic{
			Severity: sev, Summary: err.Error(), AttributePath: slices.Concat(path...),
		})
	}

	return issues
}
