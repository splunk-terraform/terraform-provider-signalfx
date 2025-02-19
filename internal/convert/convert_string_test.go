// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToString(t *testing.T) {
	t.Parallel()

	const v = "my-string"

	assert.Equal(t, v, ToString(v), "Must match the expected value")
}

func TestToStringLike(t *testing.T) {
	t.Parallel()

	type secret string

	const s = secret("my-secret")
	assert.Equal(t, s, ToStringLike[secret](s), "Must match the expected value")
}
