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
func LoadApplicationURL(ctx context.Context, meta any, fragments ...string) (string, error) {
	m, ok := meta.(*Meta)
	if !ok {
		return "", ErrMetaNotProvided
	}
	u, err := url.ParseRequestURI(m.CustomAppURL)
	if err != nil {
		return "", err
	}
	// In order to currently set that fragment,
	// the path needs to end with `/`
	// to ensure the URL is valid once built
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u.Fragment = path.Join(fragments...)
	return u.String(), nil
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
