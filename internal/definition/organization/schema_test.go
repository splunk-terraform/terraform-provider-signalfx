// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package organization

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchema(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, newSchema(), "Must have a valid schema")
}
