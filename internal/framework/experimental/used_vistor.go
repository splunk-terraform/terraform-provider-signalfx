// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package experimental

import (
	"container/list"
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/metadata"
	flow "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/signalflow"
)

type UsedVisitor struct {
	mem       map[string]string
	inputs    [][3]string
	arguments map[string]metadata.Argument
	filters   map[string][]string
}

var (
	_ flow.Visitor = (*UsedVisitor)(nil)
)

func NewUsedVisitor() *UsedVisitor {
	return &UsedVisitor{
		mem:       make(map[string]string),
		inputs:    make([][3]string, 0),
		arguments: make(map[string]metadata.Argument),
		filters:   make(map[string][]string),
	}
}

func (uv *UsedVisitor) Inputs() iter.Seq[[3]string] {
	return slices.Values(uv.inputs)
}

func (uv *UsedVisitor) Arguments() map[string]metadata.Argument {
	return uv.arguments
}

func (uv *UsedVisitor) Filters() map[string][]string {
	return uv.filters
}

func (uv *UsedVisitor) Visit(block flow.ExecutionBlock) error {
	switch b := block.(type) {
	case flow.ExecutionBlockImport:
		ident := b.Name
		if b.Alias != "" {
			ident = b.Alias
		}
		uv.mem[ident] = b.Module
	case flow.ExecutionBlockStream:
		if b.Type() != "DETECT" {
			return nil
		}
		parts := strings.Split(b.Start.FunctionName, ".")
		if modulePath, ok := uv.mem[parts[0]]; ok && len(parts) > 1 {
			uv.inputs = append(uv.inputs, [3]string{modulePath, parts[0], parts[1]})
		}
		for label, value := range b.Start.Arguments {
			switch val := value.(type) {
			case *flow.ExecutionBlockArgumentLiteral:
				uv.arguments[label] = metadata.Argument{Name: label, Type: val.Type, DefaultValue: val.Value}
			case *flow.ExecutionBlockArgumentFilter:
				var (
					field  = fmt.Sprint(val.Field.Value)
					values []string
				)
				if val.Value != nil {
					values = append(values, fmt.Sprint(val.Value.Value))
				}
				for _, v := range val.Values {
					values = append(values, fmt.Sprint(v.Value))
				}
				uv.filters[field] = append(uv.filters[field], values...)
			case *flow.ExecutionBlockArgumentOperation:
				queue := list.New()
				queue.PushBack(val.Left)
				queue.PushBack(val.Right)
				for queue.Len() > 0 {
					e := queue.Remove(queue.Front()).(flow.ExecutionBlockArgumentValue)
					if e == nil {
						continue
					}
					switch v := e.(type) {
					case *flow.ExecutionBlockArgumentLiteral:
						uv.arguments[label] = metadata.Argument{Name: label, Type: v.Type}
					case *flow.ExecutionBlockArgumentFilter:
						var (
							field  = fmt.Sprint(v.Field.Value)
							values []string
						)
						if v.Value != nil {
							values = append(values, fmt.Sprint(v.Value.Value))
						}
						for _, v := range v.Values {
							values = append(values, fmt.Sprint(v.Value))
						}
						uv.filters[field] = append(uv.filters[field], values...)
					case *flow.ExecutionBlockArgumentOperation:
						queue.PushBack(v.Left)
						queue.PushBack(v.Right)
					}
				}
			}
		}
	}
	return nil
}
