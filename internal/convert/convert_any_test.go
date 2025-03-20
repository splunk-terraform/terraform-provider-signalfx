// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToAny(t *testing.T) {
	t.Parallel()

	const v int = 0
	assert.IsType(t, (any)(int(0)), ToAny(v))
}
