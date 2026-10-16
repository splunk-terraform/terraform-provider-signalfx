// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package contracts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/splunk-terraform/terraform-provider-signalfx/internal/framework/dashify/charts/interpreter"
)

var chartTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// ScaffoldDefinition derives a draft chart identity from a product JSON Schema.
// Selection and mapping of Terraform fields remain the chart author's work.
func ScaffoldDefinition(raw []byte) (string, []byte, error) {
	document, err := interpreter.DecodeObject(string(raw))
	if err != nil {
		return "", nil, fmt.Errorf("product schema: %w", err)
	}
	title, ok := document["title"].(string)
	if !ok || !strings.HasPrefix(title, "o11y:") || len(title) == len("o11y:") {
		return "", nil, fmt.Errorf("product schema needs an o11y: element title")
	}
	if document["type"] != "object" {
		return "", nil, fmt.Errorf("product schema root must be an object")
	}
	chartType := chartName(strings.TrimPrefix(title, "o11y:"))
	required, _ := document["required"].([]any)
	for _, item := range required {
		if item == "datasource" && !strings.HasPrefix(chartType, "metrics_") {
			chartType = "metrics_" + chartType
		}
	}
	if !chartTypePattern.MatchString(chartType) {
		return "", nil, fmt.Errorf("cannot derive a chart type from %q", title)
	}
	return chartType, fmt.Appendf(nil, `# Copyright Splunk, Inc.
# SPDX-License-Identifier: MPL-2.0

# Starter only: review the UI behavior and replace TODOs before supporting this chart.
type: %s
version: 1
description: "TODO: describe this chart's Terraform interface"
element: %s
extends:
  - common/widget.yml

# Replace {} with the selected chart properties and their full wire paths.
properties: {}
`, chartType, strconv.Quote(title)), nil
}
