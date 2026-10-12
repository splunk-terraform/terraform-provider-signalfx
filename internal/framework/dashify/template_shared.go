// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package fwdashify

import (
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/signalfx/signalfx-go"
	"github.com/signalfx/signalfx-go/template"
)

const (
	// DashifyImportNamespace is the namespace shared by Dashify import annotations.
	DashifyImportNamespace               = "$import"
	ObservabilityTemplateRecordType      = "#/dashify/v1/templates/Record"
	observabilityTemplateImportPrefix    = DashifyImportNamespace + ":"
	observabilityTemplateReferencePrefix = signalfx.TemplateAPIURL + "/"
)

func ValidateObservabilityTitle(resp *resource.ValidateConfigResponse, titlePath path.Path, title types.String) {
	if title.IsUnknown() {
		return
	}
	if title.IsNull() || strings.TrimSpace(title.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(titlePath, "Missing required value", "title must contain at least one non-whitespace character")
	}
}

func ObservabilityTemplateFromResult(result *template.Result) (*template.Template, error) {
	if result == nil || result.Data == nil {
		return nil, errors.New("template API returned no template record")
	}
	if result.Data.ID == "" {
		return nil, errors.New("template API returned a template record without an ID")
	}
	return result.Data, nil
}

func ObservabilityTemplateIDValid(id string) bool {
	return id != "" && id != "." && id != ".." && !strings.Contains(id, "/")
}

func ObservabilityTemplateReference(id string) string {
	return observabilityTemplateReferencePrefix + id
}

func ObservabilityTemplateIDFromReference(reference string) (string, bool) {
	id, ok := strings.CutPrefix(reference, observabilityTemplateReferencePrefix)
	return id, ok && ObservabilityTemplateIDValid(id)
}
