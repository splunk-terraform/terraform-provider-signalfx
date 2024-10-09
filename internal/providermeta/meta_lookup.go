// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package pmeta

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mitchellh/go-homedir"
)

// MetaLookupFunc allows for unified type to preconfigure the
// the state and allow it to be easer to ensure providers are working as expected.
type MetaLookupFunc func(ctx context.Context, s *Meta) error

func (fn MetaLookupFunc) Do(ctx context.Context, s *Meta) error {
	if fn == nil {
		return errors.New("not implemented")
	}
	return fn(ctx, s)
}

// NewDefaultProviderLookups returns the list of the expected default lookups.
func NewDefaultProviderLookups() []MetaLookupFunc {
	return []MetaLookupFunc{
		FileMetaLookupFunc("/etc/signalfx.conf"),
		UserMetaLookupFunc(user.Current),
		NetrcMetaLookupFunc(os.Getenv("NETRC")),
	}
}

func FileMetaLookupFunc(path string) MetaLookupFunc {
	return func(ctx context.Context, s *Meta) error {
		tflog.Debug(ctx, "Reading provider file", map[string]any{
			"path": path,
		})

		f, err := os.Open(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err = errors.New("file not found")
			}
			return err
		}

		tflog.Debug(ctx, "Reading file content")

		dec := json.NewDecoder(f)
		dec.DisallowUnknownFields()

		if err = dec.Decode(s); err != nil {
			if errors.Is(err, io.EOF) {
				err = errors.New("no file content")
			}
			return errors.Join(err, f.Close())
		}

		return f.Close()
	}
}

func UserMetaLookupFunc(current func() (*user.User, error)) MetaLookupFunc {
	return func(ctx context.Context, s *Meta) error {
		u, err := current()
		if err != nil {
			return err
		}
		return FileMetaLookupFunc(path.Join(u.HomeDir, ".signalfx.conf")).Do(ctx, s)
	}
}

func NetrcMetaLookupFunc(path string) MetaLookupFunc {
	return func(ctx context.Context, s *Meta) error {
		if path == "" {
			var err error
			path, err = homedir.Expand(filepath.Join("~/", NetrcFile))
			if err != nil {
				return err
			}
		}
		st, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
			return err
		}

		if st.IsDir() {
			return nil
		}

		tflog.Debug(ctx, "Reading netrc file", map[string]any{
			"path": path,
		})

		m, err := netrc.FindMachine(path, "api.signalfx.com")
		if err != nil {
			return err
		}
		if m != nil && !m.IsDefault() {
			s.AuthToken = m.Password
		}
		return nil
	}
}
