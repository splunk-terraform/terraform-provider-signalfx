// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/signalfx/signalfx-go"
	"github.com/stretchr/testify/assert"
)

func TestAsErrorDiagnostics_ResponseError_Basic(t *testing.T) {
	t.Parallel()

	// Zero-value ResponseError still exercises the ResponseError branch.
	diags := AsErrorDiagnostics(&signalfx.ResponseError{})

	assert.Equal(t, diag.Diagnostics{
		{
			Severity: diag.Error,
			Summary:  "route \"\" had issues with status code 0",
			Detail:   "route \"\" had issues with status code 0",
		},
	}, diags)
}

func TestAsErrorDiagnostics_ResponseError_Wrapped(t *testing.T) {
	t.Parallel()

	// Wrap to ensure errors.As still finds the ResponseError and that
	// detail falls back to the wrapped error string when Details() is empty.
	base := &signalfx.ResponseError{}
	wrapped := fmt.Errorf("wrap: %w", base)
	// Sanity check that we actually wrapped the error
	assert.ErrorIs(t, wrapped, base)

	diags := AsErrorDiagnostics(wrapped)

	assert.Equal(t, diag.Diagnostics{
		{
			Severity: diag.Error,
			Summary:  "route \"\" had issues with status code 0",
			Detail:   "wrap: route \"\" had issues with status code 0",
		},
	}, diags)
}
