// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsPointer(t *testing.T) {
	t.Parallel()

	val := 6
	assert.IsType(t, (*int)(nil), AsPointer(val), "Must match the expected type")
	assert.Equal(t, &val, AsPointer(val), "Must match the expected value")
}
