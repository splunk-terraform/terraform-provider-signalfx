// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package feature

import (
	"errors"
	"fmt"
	"regexp"
)

type PreviewOption func(g *Preview) error

func (fn PreviewOption) apply(g *Preview) error {
	if fn == nil {
		return errors.New("function is nil")
	}
	return fn(g)
}

func WithPreviewGlobalAvailable() PreviewOption {
	return func(g *Preview) error {
		g.available = true
		g.enabled.Store(true)
		return nil
	}
}

func WithPreviewAddInVersion(version string) PreviewOption {
	return func(g *Preview) error {
		matched, err := regexp.MatchString(`^v[1-9]+\.[0-9]+`, version)
		if err != nil {
			return err
		}
		if !matched {
			return fmt.Errorf("version string %q needs to be in format vX.Y[.+]", version)
		}
		g.introduced = version
		return nil
	}
}

func WithPreviewDescription(description string) PreviewOption {
	return func(g *Preview) error {
		if description == "" {
			return errors.New("adding empty description")
		}
		g.description = description
		return nil
	}
}
