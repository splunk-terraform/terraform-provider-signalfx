// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build windows

package fwtest

import (
	"sync"

	gotesting "github.com/mitchellh/go-testing-interface"
)

var terraformUnitTestMu sync.Mutex

func lockTerraformUnitTest(t gotesting.T) func() {
	t.Helper()

	terraformUnitTestMu.Lock()
	return terraformUnitTestMu.Unlock
}
