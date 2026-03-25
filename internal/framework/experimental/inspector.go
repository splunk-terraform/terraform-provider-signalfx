// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package experimental

import (
	"context"
	"fmt"
	"net/url"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/metadata"
	flow "github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/experimental/pkg/signalflow"
)

type Inspector struct {
	meta *metadata.Client
	flow *flow.Client
}

func NewInspector(domain *url.URL, token string) *Inspector {
	return &Inspector{
		meta: metadata.NewClient(domain, token),
		flow: flow.NewClient(domain, token),
	}
}

func (inspect *Inspector) GetAutoDetectorArgumentsAndFilters(ctx context.Context, programText string) ([]metadata.Argument, map[string][]string, error) {
	graph, err := inspect.flow.GetExecutionGraph(ctx, programText)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load exec graph: %w", err)
	}

	used := NewUsedVisitor()
	if err := graph.Visit(used); err != nil {
		return nil, nil, fmt.Errorf("unable to visit exec graph: %w", err)
	}

	var (
		results   []metadata.Argument
		setValues = used.Arguments()
	)
	for inputs := range used.Inputs() {
		in, err := inspect.meta.GetModuleFunctionMetadata(ctx, inputs[0], inputs[1], inputs[2])
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get module function metadata: %w", err)
		}
		for _, r := range in.Arguments {
			if val, ok := setValues[r.Name]; ok {
				r.DefaultValue = val.DefaultValue
			}
			results = append(results, r)
		}
	}

	return results, used.Filters(), nil
}
