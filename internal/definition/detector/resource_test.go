// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResource(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, NewResource(), "Must have a valid resource defined")
}

func TestResourceCreate(t *testing.T) {
	t.Parallel()

}

func TestResourceRead(t *testing.T) {
	t.Parallel()

}

func TestResourceUpdate(t *testing.T) {
	t.Parallel()

}

func TestResourceDelete(t *testing.T) {
	t.Parallel()

}
