// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

// ToAny provides a way to convert back to the generic terraform values
func ToAny[T any](in T) any {
	return in
}
