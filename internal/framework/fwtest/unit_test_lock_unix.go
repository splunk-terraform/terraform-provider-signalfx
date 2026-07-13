// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package fwtest

import (
	"os"
	"path/filepath"
	"syscall"

	gotesting "github.com/mitchellh/go-testing-interface"
)

func lockTerraformUnitTest(t gotesting.T) func() {
	t.Helper()

	file, err := os.OpenFile(
		filepath.Join(os.TempDir(), "terraform-provider-signalfx-unit-test.lock"),
		os.O_CREATE|os.O_RDWR,
		0o600,
	)
	if err != nil {
		t.Fatalf("unable to open Terraform unit test lock: %s", err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		t.Fatalf("unable to acquire Terraform unit test lock: %s", err)
	}

	return func() {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
			t.Fatalf("unable to release Terraform unit test lock: %s", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("unable to close Terraform unit test lock: %s", err)
		}
	}
}
