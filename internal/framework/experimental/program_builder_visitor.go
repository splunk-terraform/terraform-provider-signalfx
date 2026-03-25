// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package experimental

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	flow "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/signalflow"
)

type ProgramBuilderVisitor struct {
	program   strings.Builder
	filterKey string
	filters   map[string][]string
	inputs    map[string]any
}

var _ flow.Visitor = (*ProgramBuilderVisitor)(nil)

func NewProgramBuilderVisitor(opts ...func(*ProgramBuilderVisitor)) *ProgramBuilderVisitor {
	v := &ProgramBuilderVisitor{
		filterKey: "filter",
		filters:   make(map[string][]string),
		inputs:    make(map[string]any),
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func (v *ProgramBuilderVisitor) Visit(block flow.ExecutionBlock) error {
	switch b := block.(type) {
	case flow.ExecutionBlockImport:
		if b.Name != "" {
			_, _ = fmt.Fprintf(&v.program, "from %s import %s", b.Module, b.Name)
		} else {
			_, _ = fmt.Fprintf(&v.program, "import %s", b.Module)
		}
		if b.Alias != "" {
			_, _ = fmt.Fprintf(&v.program, " as %s", b.Alias)
		}
	case flow.ExecutionBlockStream:
		if b.Type() != "DETECT" {
			// Leave it unmodified, but enforce new line for readability
			_, _ = fmt.Fprintln(&v.program, b.Start.OriginalText)
			return nil
		}
		_, _ = fmt.Fprintf(&v.program, "%s(", b.Start.FunctionName)
		if len(v.filters) > 0 {
			_, _ = fmt.Fprintf(&v.program, "%s=", v.filterKey)
			for idx, key := range slices.Sorted(maps.Keys(v.filters)) {
				if idx > 0 {
					_, _ = fmt.Fprint(&v.program, " and ")
				}
				values := v.filters[key]
				_, _ = fmt.Fprintf(&v.program, "filter('%s'", key)
				for _, value := range values {
					_, _ = fmt.Fprintf(&v.program, ", '%s'", value)
				}
				_, _ = fmt.Fprint(&v.program, ")")
			}
		}
		for idx, arg := range slices.Sorted(maps.Keys(v.inputs)) {
			var (
				value    = v.inputs[arg]
				valueStr string
			)
			switch vv := value.(type) {
			case int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64,
				float32, float64:
				valueStr = fmt.Sprint(vv)
			case bool:
				stmt := "False"
				if vv {
					stmt = "True"
				}
				valueStr = stmt
			case string:
				valueStr = fmt.Sprintf("'%s'", vv)
			default:
				// For unsupported types, we can choose to either skip or return an error. Here we skip.
			}
			if idx != 0 || len(v.filters) > 0 {
				_, _ = fmt.Fprint(&v.program, ", ")
			}
			_, _ = fmt.Fprintf(&v.program, "%s=%s", arg, valueStr)
		}
		_, _ = fmt.Fprint(&v.program, ")")
		for _, method := range b.Methods {
			_, _ = fmt.Fprintf(&v.program, ".%s(", method.FunctionName)
			// Assuming method arguments are in the form of key-value pairs
			for idx, argKey := range slices.Sorted(maps.Keys(method.Arguments)) {
				argValue := method.Arguments[argKey]
				if idx > 0 {
					_, _ = fmt.Fprint(&v.program, ", ")
				}
				_, _ = fmt.Fprintf(&v.program, "%s=%s", argKey, argValue.Expression())
			}
			_, _ = fmt.Fprint(&v.program, ")")
		}
	}

	_, _ = fmt.Fprintln(&v.program) // Enforce new line between block definition
	return nil
}

func (v *ProgramBuilderVisitor) WithFilterKey(key string) *ProgramBuilderVisitor {
	v.filterKey = key
	return v
}

func (v *ProgramBuilderVisitor) WithFilter(name string, values ...string) *ProgramBuilderVisitor {
	v.filters[name] = append(v.filters[name], values...)
	return v
}

func (v *ProgramBuilderVisitor) WithInput(name string, value any) *ProgramBuilderVisitor {
	v.inputs[name] = value
	return v
}

func (v *ProgramBuilderVisitor) BuildProgramText() string {
	return v.program.String()
}
