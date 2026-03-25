// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package flow

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"slices"
)

// ExecutionGraph is the generic representation of the complete signalflow program
// as represented by their concrete execution blocks.
type ExecutionGraph []ExecutionBlock

type executionGraphTyped struct {
	Type string `json:"type"`
}

var (
	_ json.Unmarshaler = (*ExecutionGraph)(nil)
)

func NewExecutionGraphFromJSON(r io.Reader) (ExecutionGraph, error) {
	var eg ExecutionGraph
	if err := json.NewDecoder(r).Decode(&eg); err != nil {
		return nil, err
	}
	return eg, nil
}

func (eg ExecutionGraph) All() iter.Seq[ExecutionBlock] {
	return slices.Values(eg)
}

func (eg ExecutionGraph) Visit(visitor Visitor) error {
	for block := range eg.All() {
		if err := visitor.Visit(block); err != nil {
			return err
		}
	}
	return nil
}

func (eg *ExecutionGraph) UnmarshalJSON(data []byte) error {
	var rawBlocks []json.RawMessage
	if err := json.Unmarshal(data, &rawBlocks); err != nil {
		return fmt.Errorf("unable to split blocks: %w", err)
	}

	for _, raw := range rawBlocks {
		var blockType executionGraphTyped
		if err := json.Unmarshal(raw, &blockType); err != nil {
			return err
		}

		switch t := blockType.Type; t {
		case "IMPORT":
			var block ExecutionBlockImport
			if err := json.Unmarshal(raw, &block); err != nil {
				return err
			}
			*eg = append(*eg, block)
		case "PLOT", "DETECT":
			var block ExecutionBlockStream
			if err := json.Unmarshal(raw, &block); err != nil {
				return err
			}
			*eg = append(*eg, block)
		default:
			return fmt.Errorf("unknown block type %q", t)
		}
	}

	return nil
}
