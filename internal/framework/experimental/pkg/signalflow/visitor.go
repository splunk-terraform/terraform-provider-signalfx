// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

// Visitor is a visitor pattern for traversing the execution graph.
type Visitor interface {
	Visit(block ExecutionBlock) error
}

// VisitorFunc is a helper type for creating visitors from functions.
type VisitorFunc func(block ExecutionBlock) error

func (vf VisitorFunc) Visit(block ExecutionBlock) error {
	return vf(block)
}
