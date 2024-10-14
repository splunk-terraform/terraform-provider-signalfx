// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package tftest

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.uber.org/multierr"
)

// AcceptanceHandler is used to abstract some of the more raw
// terraform test functionality and standardise how acceptance tests are invoked.
type AcceptanceHandler struct {
	beforeAll func()
	provider  *schema.Provider
}

// AcceptanceHandlerOption is used to supply additional values to a test case.
type AcceptanceHandlerOption func(*AcceptanceHandler)

func WithAcceptanceResources(resources map[string]*schema.Resource) AcceptanceHandlerOption {
	return func(ah *AcceptanceHandler) {
		ah.provider.ResourcesMap = resources
	}
}

func WithAcceptanceDataSources(data map[string]*schema.Resource) AcceptanceHandlerOption {
	return func(ah *AcceptanceHandler) {
		ah.provider.DataSourcesMap = data
	}
}

func WithAcceptanceBeforeAll(before func()) AcceptanceHandlerOption {
	return func(ah *AcceptanceHandler) {
		ah.beforeAll = before
	}
}

func NewAcceptanceHandler(opts []AcceptanceHandlerOption) *AcceptanceHandler {
	ah := &AcceptanceHandler{
		provider: &schema.Provider{
			Schema:               make(map[string]*schema.Schema),
			ConfigureContextFunc: newAcceptanceConfigure,
		},
	}

	for _, opt := range opts {
		opt(ah)
	}

	return ah
}

func (ah *AcceptanceHandler) Validate() (errs error) {
	if len(ah.provider.DataSourcesMap) == 0 && len(ah.provider.ResourcesMap) == 0 {
		errs = multierr.Append(errs, errors.New("missing resource and datasource defintions"))
	}

	return multierr.Append(errs, ah.provider.InternalValidate())
}

func (ah *AcceptanceHandler) Test(t *testing.T, steps []resource.TestStep) {
	var msgs []string
	if _, set := os.LookupEnv("SFX_AUTH_TOKEN"); !set {
		msgs = append(msgs, fmt.Sprintf("missing environment variable %q", "SFX_AUTH_TOKEN"))
	}
	if _, set := os.LookupEnv("SFX_API_URL"); !set {
		msgs = append(msgs, fmt.Sprintf("missing environment variable %q", "SFX_API_URL"))
	}
	if len(msgs) != 0 {
		t.Skip(
			"Missing required environment variables to run tests, Please set the listed variables below:\n",
			strings.Join(msgs, "\n"),
		)
		return
	}

	// Due to how the terraform library works,
	// if this is globally set for each test case,
	// it will cause some functions to panic instead of returning an error.
	// Therefore, this will set the environment variable for the test and will
	// be unset once the test is completed.
	//
	// See https://github.com/hashicorp/terraform-plugin-sdk/issues/1384 for more details.
	t.Setenv("TF_ACC", "1")

	tc := resource.TestCase{
		IsUnitTest: false,
		ProviderFactories: map[string]func() (*schema.Provider, error){
			"signalfx": func() (*schema.Provider, error) { //nolint:unparam // Required signature
				return ah.provider, nil
			},
		},
		PreCheck: ah.beforeAll,
		Steps:    steps,
	}

	resource.Test(t, tc)
}
