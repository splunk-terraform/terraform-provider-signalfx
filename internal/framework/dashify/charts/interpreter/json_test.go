// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package interpreter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeObjectPreservesNumbersAndRequiresObject(t *testing.T) {
	for name, raw := range map[string]string{
		"array root": `[]`,
		"null root":  `null`,
		"incomplete": `{"chart":[}`,
		"empty":      ``,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeObject(raw)
			assert.Error(t, err, "accepted malformed JSON or a non-object root")
		})
	}
	object, err := DecodeObject(`{"large":9007199254740993,"null":null,"empty":[]}`)
	require.NoError(t, err)
	assert.Equal(t, json.Number("9007199254740993"), object["large"], "lost numeric precision")
	empty, ok := object["empty"].([]any)
	require.True(t, ok, "empty array must decode as []any, got %T", object["empty"])
	assert.NotNil(t, empty, "empty array must not become null")
	assert.Empty(t, empty)
}
