// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

func ToString(in any) string {
	return ToStringLike[string](in)
}

func ToStringLike[T ~string](in any) T {
	return in.(T)
}
