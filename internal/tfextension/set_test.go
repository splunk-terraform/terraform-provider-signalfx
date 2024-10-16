// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfext

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestNewSchemaSet(t *testing.T) {
	t.Parallel()

	vector := []any{3, 1, 5, 4, 2, 6}

	s := NewSchemaSet(schema.HashInt, vector)
	assert.Equal(t, 6, s.Len(), "Must have all the values defined")
	assert.Equal(t, vector, s.List(), "Must match the expected values")
}
