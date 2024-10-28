// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

func AsPointer[T any](t T) *T {
	return &t
}

func AsPointerOnCondition[T any](t T, cond func(val T) bool) *T {
	if !cond(t) {
		return nil
	}
	return AsPointer[T](t)
}
