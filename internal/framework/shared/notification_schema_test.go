// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNotificationResourceAttribute(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewNotificationResourceAttribute())
}
