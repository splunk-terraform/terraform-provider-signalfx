package tfext

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
)

func TestAppendDiagnostics(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		orig    diag.Diagnostics
		entries diag.Diagnostics
		expect  diag.Diagnostics
	}{
		{
			name:    "no values",
			orig:    nil,
			entries: nil,
			expect:  nil,
		},
		{
			name: "no additional values",
			orig: diag.Diagnostics{
				{Severity: diag.Error, Summary: "Issue creating value"},
			},
			entries: nil,
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "Issue creating value"},
			},
		},
		{
			name: "appending values",
			orig: nil,
			entries: diag.Diagnostics{
				{Severity: diag.Error, Summary: "Issue creating value"},
			},
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "Issue creating value"},
			},
		},
		{
			name: "combined values",
			orig: diag.Diagnostics{
				{Severity: diag.Error, Summary: "a"},
			},
			entries: diag.Diagnostics{
				{Severity: diag.Warning, Summary: "b"},
			},
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "a"},
				{Severity: diag.Warning, Summary: "b"},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				tc.expect,
				AppendDiagnostics(tc.orig, tc.entries...),
				"Must match the expected values",
			)
		})
	}
}

func TestAsErrorDiagnostics(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		value  diag.Diagnostics
		expect diag.Diagnostics
	}{
		{
			name:   "nil error",
			value:  AsErrorDiagnostics(nil),
			expect: nil,
		},
		{
			name:  "defined error",
			value: AsErrorDiagnostics(errors.New("boo")),
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "boo"},
			},
		},
		{
			name:  "error with path",
			value: AsErrorDiagnostics(errors.New("bad entry"), cty.IndexStringPath("attr")),
			expect: diag.Diagnostics{
				{Severity: diag.Error, Summary: "bad entry", AttributePath: cty.IndexStringPath("attr")},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, tc.value)
		})
	}
}

func TestAsWarnDiagnostics(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		value  diag.Diagnostics
		expect diag.Diagnostics
	}{
		{
			name:   "nil error",
			value:  AsWarnDiagnostics(nil),
			expect: nil,
		},
		{
			name:  "defined error",
			value: AsWarnDiagnostics(errors.New("boo")),
			expect: diag.Diagnostics{
				{Severity: diag.Warning, Summary: "boo"},
			},
		},
		{
			name:  "error with path",
			value: AsWarnDiagnostics(errors.New("bad entry"), cty.IndexStringPath("attr")),
			expect: diag.Diagnostics{
				{Severity: diag.Warning, Summary: "bad entry", AttributePath: cty.IndexStringPath("attr")},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, tc.value)
		})
	}
}