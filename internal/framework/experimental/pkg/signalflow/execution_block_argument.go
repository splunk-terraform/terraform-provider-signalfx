// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ExecutionBlockArgumentValue interface {
	isArgumentValue()

	Expression() string
}

type ExecutionBlockArguments map[string]ExecutionBlockArgumentValue

type executionBlockArgumentJSONProbe struct {
	Type string `json:"type"`
}

var (
	_ json.Unmarshaler = (*ExecutionBlockArguments)(nil)
)

func (eba *ExecutionBlockArguments) UnmarshalJSON(data []byte) error {
	if *eba == nil {
		(*eba) = make(ExecutionBlockArguments)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for k, v := range raw {
		var typeProbe executionBlockArgumentJSONProbe
		if err := json.Unmarshal(v, &typeProbe); err != nil {
			return err
		}

		var arg ExecutionBlockArgumentValue
		switch typeProbe.Type {
		case "filter":
			arg = &ExecutionBlockArgumentFilter{}
		case "binary_expression":
			arg = &ExecutionBlockArgumentOperation{}
		default:
			arg = &ExecutionBlockArgumentLiteral{}
		}

		if err := json.Unmarshal(v, arg); err != nil {
			return err
		}
		(*eba)[k] = arg
	}
	return nil
}

type executionBlockArgumentEmbedded struct{}

func (executionBlockArgumentEmbedded) isArgumentValue() {}

type ExecutionBlockArgumentLiteral struct {
	executionBlockArgumentEmbedded

	Type         string `json:"type,omitempty"`
	OriginalText string `json:"originalText,omitempty"`
	Position     int    `json:"position,omitempty"`
	Value        any    `json:"value,omitempty"`
}

func (ebal *ExecutionBlockArgumentLiteral) Expression() string {
	switch v := ebal.Value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	default:
		return fmt.Sprint(v)
	}
}

type ExecutionBlockArgumentFilter struct {
	executionBlockArgumentEmbedded

	Type         string                          `json:"type,omitempty"`
	OriginalText string                          `json:"originalText,omitempty"`
	Position     int                             `json:"position,omitempty"`
	Field        ExecutionBlockArgumentLiteral   `json:"field,omitempty"`
	Value        *ExecutionBlockArgumentLiteral  `json:"value,omitempty"`
	Values       []ExecutionBlockArgumentLiteral `json:"values,omitempty"`
}

func (ebaf *ExecutionBlockArgumentFilter) Expression() string {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "filter(%q,", ebaf.Field.Value)

	var values []string
	if ebaf.Value != nil {
		values = append(values, ebaf.Value.Expression())
	}
	for _, v := range ebaf.Values {
		values = append(values, v.Expression())
	}
	_, _ = fmt.Fprint(&sb, strings.Join(values, " , "))
	_, _ = fmt.Fprint(&sb, ")")
	return sb.String()
}

type ExecutionBlockArgumentOperation struct {
	executionBlockArgumentEmbedded

	Type         string                      `json:"type,omitempty"`
	OriginalText string                      `json:"originalText,omitempty"`
	Position     int                         `json:"position,omitempty"`
	Operation    string                      `json:"op,omitempty"`
	LeftRaw      json.RawMessage             `json:"left,omitempty"`
	RightRaw     json.RawMessage             `json:"right,omitempty"`
	Left         ExecutionBlockArgumentValue `json:"-"`
	Right        ExecutionBlockArgumentValue `json:"-"`
}

func (ebao *ExecutionBlockArgumentOperation) Expression() string {
	return fmt.Sprintf("(%s %s %s)", ebao.Left.Expression(), ebao.Operation, ebao.Right.Expression())
}

func (ebao *ExecutionBlockArgumentOperation) UnmarshalJSON(data []byte) error {
	type Alias ExecutionBlockArgumentOperation
	var result Alias
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	(*ebao) = (ExecutionBlockArgumentOperation)(result)

	{
		var probe executionBlockArgumentJSONProbe
		if err := json.Unmarshal(ebao.LeftRaw, &probe); err != nil {
			return err
		}

		var arg ExecutionBlockArgumentValue
		switch probe.Type {
		case "filter":
			arg = &ExecutionBlockArgumentFilter{}
		case "binary_expression":
			arg = &ExecutionBlockArgumentOperation{}
		default:
			arg = &ExecutionBlockArgumentLiteral{}
		}

		if err := json.Unmarshal(ebao.LeftRaw, arg); err != nil {
			return err
		}
		ebao.Left = arg
	}
	{
		var probe executionBlockArgumentJSONProbe
		if err := json.Unmarshal(ebao.RightRaw, &probe); err != nil {
			return err
		}

		var arg ExecutionBlockArgumentValue
		switch probe.Type {
		case "filter":
			arg = &ExecutionBlockArgumentFilter{}
		case "binary_expression":
			arg = &ExecutionBlockArgumentOperation{}
		default:
			arg = &ExecutionBlockArgumentLiteral{}
		}

		if err := json.Unmarshal(ebao.RightRaw, arg); err != nil {
			return err
		}
		ebao.Right = arg
	}

	return nil
}
