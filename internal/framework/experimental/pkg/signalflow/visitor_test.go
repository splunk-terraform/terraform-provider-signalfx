// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisitorImplements(t *testing.T) {
	t.Parallel()

	assert.Implements(t, (*Visitor)(nil), new(VisitorFunc), "Must implement the expected interface")
}

func TestVisitorFunc(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		visitor Visitor
		errVal  string
	}{
		{
			name: "no issue visitor",
			visitor: VisitorFunc(func(block ExecutionBlock) error {
				return nil
			}),
			errVal: "",
		},
		{
			name: "erroring visitor",
			visitor: VisitorFunc(func(block ExecutionBlock) error {
				return assert.AnError
			}),
			errVal: "assert.AnError general error for testing",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var block ExecutionBlock

			if err := tc.visitor.Visit(block); tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal)
			} else {
				assert.NoError(t, err, "Expected no error from visitor")
			}
		})
	}
}
