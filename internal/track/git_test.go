// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package track

import (
	"errors"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadGitDetails(t *testing.T) {
	t.Parallel()

	// Since nil is explicitly being used, need to ensure that it does not cause
	// a panic during runtime.
	assert.NotPanics(t, func() {
		_, _ = ReadGitDetails(t.Context())
	})
}

func TestGitRepositoryFuncOpen(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		fn       GitRepositoryFunc
		expected *git.Repository
		errVal   string
	}{} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo, err := tc.fn.Open(t.Name(), &git.PlainOpenOptions{})
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must not error")
			}
			assert.Equal(t, tc.expected, repo, "Must match the expected value")
		})
	}
}

func TestGitRepositoryReadDetails(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		get    func(tb testing.TB) (*git.Repository, error)
		expect *Details
		errVal string
	}{
		{
			name: "failed to load",
			get: func(_ testing.TB) (*git.Repository, error) {
				return nil, errors.New("failed to load")
			},
			expect: nil,
			errVal: "failed to load",
		},
		{
			name: "successful project load",
			get: func(tb testing.TB) (*git.Repository, error) {
				repo, err := git.PlainInitWithOptions(tb.TempDir(), &git.PlainInitOptions{
					InitOptions: git.InitOptions{
						DefaultBranch: plumbing.Main,
					},
					Bare: false,
				})
				require.NoError(tb, err, "Must not error creating project")

				_, err = repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{"localhost/my-project.git"},
				})
				require.NoError(tb, err, "Must not error creating remote config")

				wt, err := repo.Worktree()
				require.NoError(tb, err, "Must not error loading working tree")

				_, err = wt.Commit("bare commit", &git.CommitOptions{
					AllowEmptyCommits: true,
					Author: &object.Signature{
						Name:  "test",
						Email: "test@localhost",
						When:  time.Now(),
					},
				})
				require.NoError(tb, err, "Must not error creating commit")
				return repo, nil
			},
			expect: &Details{
				name:   "localhost/my-project",
				branch: "main",
			},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fn := GitRepositoryFunc(func(path string, opts *git.PlainOpenOptions) (*git.Repository, error) {
				return tc.get(t)
			})

			actual, err := fn.ReadDetails(t.Context())
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expecting error")
			} else {
				assert.NoError(t, err, "Must not error leading read details")
			}

			assert.Equal(t, tc.expect, actual, "Must match the expected details")
		})
	}
}
