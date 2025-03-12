// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestGetGlobalRegistry(t *testing.T) {
	t.Parallel()

	reg := GetGlobalRegistry()
	assert.NotNil(t, reg, "Must have a valid registry value")
	assert.Equal(
		t,
		unsafe.Pointer(reg),
		unsafe.Pointer(GetGlobalRegistry()),
		"Must have the same pointer reference",
	)
}
