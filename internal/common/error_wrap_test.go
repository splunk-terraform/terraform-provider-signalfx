// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/signalfx/signalfx-go"
	"github.com/stretchr/testify/assert"
)

func TestWrapResponseError_PreservesIdentityAndMessage(t *testing.T) {
	t.Parallel()

	base := &signalfx.ResponseError{}

	wrapped := WrapResponseError(base)

	// Identity preserved for errors.Is/As
	assert.ErrorIs(t, wrapped, base)

	// Message should include outer context and base error
	assert.Equal(t,
		"route \"\" had issues with status code 0: route \"\" had issues with status code 0",
		wrapped.Error(),
	)
}
