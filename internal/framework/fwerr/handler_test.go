// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwerr

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/signalfx/signalfx-go"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected diag.Diagnostics
	}{
		{
			name: "nil error",
			err:  nil,
		},
		{
			name: "non api issue error",
			err:  fmt.Errorf("issue reaching required host"),
			expected: diag.Diagnostics{
				diag.NewErrorDiagnostic("Issue handling request", "issue reaching required host"),
			},
		},
		{
			name: "signalfx response error",
			err:  &signalfx.ResponseError{},
			expected: diag.Diagnostics{
				diag.NewErrorDiagnostic(`route "" had issues with status code 0`, ""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ErrorHandler(context.TODO(), tfsdk.State{}, tt.err)
			assert.Equal(t, tt.expected, result, "Must match expected diagnostics")
		})
	}
}
