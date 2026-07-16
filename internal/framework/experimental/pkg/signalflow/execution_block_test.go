// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutionBlockTypes(t *testing.T) {
	t.Parallel()

	blocks := []ExecutionBlock{
		ExecutionBlockImport{Typed: "IMPORT"},
		ExecutionBlockStream{Typed: "DETECT"},
	}

	assert.Equal(t, "IMPORT", blocks[0].Type())
	assert.Equal(t, "DETECT", blocks[1].Type())
	for _, block := range blocks {
		block.isBlock()
	}
}
