// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package pmeta

import (
	"context"
	"errors"
	"net/url"
	"path"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/signalfx/signalfx-go"
	"go.uber.org/multierr"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

var (
	ErrMetaNotProvided = errors.New("expected to implement type Meta")
)

// Meta is the result of `resource.Provider` being correctly configured
// and is returned as part of `provider.Meta()`.
//
// It is abstracted out from the provider definition to make it easier
// to test CRUD operations within unit tests.
type Meta struct {
	AuthToken    string           `json:"auth_token"`
	APIURL       string           `json:"api_url"`
	CustomAppURL string           `json:"custom_app_url"`
	Client       *signalfx.Client `json:"-"`
}

// LoadClient returns the configured [signalfx.Client] ready to use.
//
// Note that it is a shared instance so high amounts of parallelism could cause issues.
func LoadClient(ctx context.Context, meta any) (*signalfx.Client, error) {
	if m, ok := meta.(*Meta); ok {
		return m.Client, nil
	}
	tflog.Error(ctx, "Failed to load state from meta value", map[string]any{
		"meta": meta,
	})
	return nil, ErrMetaNotProvided
}

// LoadApplicationURL will generate the FQDN using the set CustomAppURL from the meta value.
func LoadApplicationURL(ctx context.Context, meta any, fragments ...string) string {
	m, ok := meta.(*Meta)
	if !ok {
		tflog.Error(ctx, "Unable to convert to expected type")
		return ""
	}
	u, err := url.ParseRequestURI(m.CustomAppURL)
	if err != nil {
		tflog.Error(ctx, "Issue trying to parse custom app url", tfext.NewLogFields().Error(err))
		return ""
	}
	// In order to currently set that fragment,
	// the path needs to end with `/`
	// to ensure the URL is valid once built
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u.Fragment = path.Join(fragments...)
	return u.String()
}

func (s *Meta) Validate() (errs error) {
	if s.AuthToken == "" {
		errs = multierr.Append(errs, errors.New("auth token not set"))
	}
	if s.APIURL == "" {
		errs = multierr.Append(errs, errors.New("api url is not set"))
	}
	return errs
}
