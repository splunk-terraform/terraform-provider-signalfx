// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package check

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

func TestNotificationReminderType_ValidValue(t *testing.T) {
	validateFunc := NotificationReminderType()
	value := "TIMEOUT"
	diagnostics := validateFunc(value, cty.Path{})

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %v", diagnostics)
	}
}

func TestNotificationReminderType_InvalidValue(t *testing.T) {
	validateFunc := NotificationReminderType()
	value := "INVALID"
	diagnostics := validateFunc(value, cty.Path{})

	if len(diagnostics) == 0 {
		t.Errorf("Expected diagnostics, got none")
	}

	expectedMessage := `value "INVALID" is not allowed; must be one of: [TIMEOUT]`
	if diagnostics[0].Summary != expectedMessage {
		t.Errorf("Expected diagnostic message %q, got %q", expectedMessage, diagnostics[0].Summary)
	}
}

func TestNotificationReminderType_NonStringValue(t *testing.T) {
	validateFunc := NotificationReminderType()
	value := 123 // Non-string value
	diagnostics := validateFunc(value, cty.Path{})

	if len(diagnostics) == 0 {
		t.Errorf("Expected diagnostics, got none")
	}

	expectedMessage := "expected 123 to be of type string"
	if diagnostics[0].Summary != expectedMessage {
		t.Errorf("Expected diagnostic message %q, got %q", expectedMessage, diagnostics[0].Summary)
	}
}
