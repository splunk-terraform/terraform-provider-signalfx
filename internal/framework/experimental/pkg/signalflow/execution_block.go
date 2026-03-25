// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

// ExecutionBlock is a generic interface that is used to represent a unit of work within a signalflow program.
type ExecutionBlock interface {
	isBlock()

	// Type returns the name of block type
	// which should used to cast into its respective concrete type.
	Type() string
}

type embeddedBlock struct{}

func (embeddedBlock) isBlock() {}

// ExecutionBlockImport represents a block of the execution graph
// that is used for type `IMPORT`
type ExecutionBlockImport struct {
	embeddedBlock

	Typed  string `json:"type,omitempty"`
	Name   string `json:"name"`
	Module string `json:"module"`
	Alias  string `json:"alias,omitempty"`
}

func (ebi ExecutionBlockImport) Type() string {
	return ebi.Typed
}

// ExecutionBlockStream represents a block of the execution graph
// that is used for types `PLOT` and `DETECT`
type ExecutionBlockStream struct {
	embeddedBlock

	Typed          string                       `json:"type,omitempty"`
	UniqueKey      int                          `json:"uniqueKey"`
	ExpressionText string                       `json:"expressionText,omitempty"`
	Start          ExecutionBlockStreamMethod   `json:"start,omitempty"`
	Methods        []ExecutionBlockStreamMethod `json:"streamMethods,omitempty"`
}

func (ebs ExecutionBlockStream) Type() string {
	return ebs.Typed
}

type ExecutionBlockStreamMethod struct {
	FunctionName string                  `json:"functionName"`
	OriginalText string                  `json:"originalText,omitempty"`
	Position     int                     `json:"position,omitempty"`
	Type         string                  `json:"type,omitempty"`
	Arguments    ExecutionBlockArguments `json:"args,omitempty"`
}
