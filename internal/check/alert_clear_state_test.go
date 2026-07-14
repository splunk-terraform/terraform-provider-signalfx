// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestAlertClearState_ValidValues(t *testing.T) {
	validateFunc := AlertClearState()
	for _, value := range []string{"OK", "AUTO_RESOLVED", "STOPPED", "MANUALLY_RESOLVED"} {
		diagnostics := validateFunc(value, cty.Path{})
		if len(diagnostics) != 0 {
			t.Errorf("Expected no diagnostics for %q, got %v", value, diagnostics)
		}
	}
}

func TestAlertClearState_InvalidValue(t *testing.T) {
	validateFunc := AlertClearState()
	diagnostics := validateFunc("INVALID", cty.Path{})

	if len(diagnostics) == 0 {
		t.Errorf("Expected diagnostics, got none")
	}

	expected := `value "INVALID" is not allowed; must be one of: [OK AUTO_RESOLVED STOPPED MANUALLY_RESOLVED]`
	if diagnostics[0].Summary != expected {
		t.Errorf("Expected diagnostic message %q, got %q", expected, diagnostics[0].Summary)
	}
}

func TestAlertClearState_NonStringValue(t *testing.T) {
	validateFunc := AlertClearState()
	diagnostics := validateFunc(123, cty.Path{})

	if len(diagnostics) == 0 {
		t.Errorf("Expected diagnostics, got none")
	}

	expected := "expected 123 to be of type string"
	if diagnostics[0].Summary != expected {
		t.Errorf("Expected diagnostic message %q, got %q", expected, diagnostics[0].Summary)
	}
}
