// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwtest

import (
	resourcetest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gotesting "github.com/mitchellh/go-testing-interface"
)

func UnitTest(t gotesting.T, c resourcetest.TestCase) {
	t.Helper()

	unlock := lockTerraformUnitTest(t)
	defer unlock()

	resourcetest.UnitTest(t, c)
}
