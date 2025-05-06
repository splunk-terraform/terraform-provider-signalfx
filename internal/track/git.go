// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package track

import (
	"context"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tfext "github.com/splunk-terraform/terraform-provider-signalfx/internal/tfextension"
)

// GitRepositoryFunc is used to allow for mocking the underlying calls to check the git repo
// to make it easier to validate the work being done.
type GitRepositoryFunc func(path string, opts *git.PlainOpenOptions) (*git.Repository, error)

// ReadGitDetails is a convenience function that correctly calls the mockable type
// to use the concrete implementation
func ReadGitDetails(ctx context.Context) (*Details, error) {
	return (GitRepositoryFunc)(nil).ReadDetails(ctx)
}

// Open will read the git repo details, and if there is a mock set, then it will call that implementation
// otherwise, it will call the git implementation.
func (fn GitRepositoryFunc) Open(path string, opts *git.PlainOpenOptions) (*git.Repository, error) {
	if fn != nil {
		return fn(path, opts)
	}
	return git.PlainOpenWithOptions(path, opts)
}

func (fn GitRepositoryFunc) ReadDetails(ctx context.Context) (*Details, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "Checking working directory for git details", tfext.NewLogFields().
		Field("cwd", cwd),
	)

	repo, err := fn.Open(cwd, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, err
	}

	return NewDetailsFromGit(repo)
}
