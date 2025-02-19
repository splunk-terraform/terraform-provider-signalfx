// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

// Func is the operation that converts from the original value
// that is stored as part of the schema set value
// and will convert it to the expected out type.
type Func[In any, Out any] func(s In) Out
