// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package dimension

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchema(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, newSchema(), "Must have a defined schema returned")
}
