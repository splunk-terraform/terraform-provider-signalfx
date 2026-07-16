// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwshared

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestNewNotificationResourceAttribute(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewNotificationResourceAttribute())

	objectType := NewNotificationObjectType()
	assert.Len(t, objectType.AttrTypes, 12)
	assert.Equal(t, types.StringType, objectType.AttrTypes["type"])
}
