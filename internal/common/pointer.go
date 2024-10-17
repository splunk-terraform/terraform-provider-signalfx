// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

func AsPointer[T any](t T) *T {
	return &t
}
