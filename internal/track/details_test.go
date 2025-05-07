// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package track

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDetailsFromGit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		get    func(tb testing.TB) *git.Repository
		expect *Details
		errVal string
	}{
		{
			name: "empty repo",
			get: func(tb testing.TB) *git.Repository {
				repo, err := git.PlainInit(tb.TempDir(), false)
				require.NoError(tb, err, "Must not error creating project")
				return repo
			},
			expect: nil,
			errVal: "remote not found",
		},
		{
			name: "clean history",
			get: func(tb testing.TB) *git.Repository {
				repo, err := git.PlainInit(tb.TempDir(), false)
				require.NoError(tb, err, "Must not error creating project")
				_, err = repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{"localhost/my-project.git"},
				})
				require.NoError(tb, err, "Must not error creating remote config")
				return repo
			},
			expect: nil,
			errVal: "unable to read head: reference not found",
		},
		{
			name: "repo configured",
			get: func(tb testing.TB) *git.Repository {
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
				return repo
			},
			expect: &Details{name: "localhost/my-project", branch: "main"},
			errVal: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := NewDetailsFromGit(tc.get(t))
			if tc.errVal != "" {
				assert.EqualError(t, err, tc.errVal, "Must match the expected value")
			} else {
				assert.NoError(t, err, "Must match the expected error")
			}
			assert.Equal(t, tc.expect, actual)
		})
	}
}

func TestDetailsTags(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		details Details
		expect  []string
	}{
		{
			name:    "zero value",
			details: Details{},
			expect: []string{
				"project:",
				"branch:",
				"experimental:false",
			},
		},
		{
			name: "set values",
			details: Details{
				name:   "my-awesome-project",
				branch: "main",
				dirty:  false,
			},
			expect: []string{
				"project:my-awesome-project",
				"branch:main",
				"experimental:false",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expect, tc.details.Tags())
		})
	}
}

func TestCleanGitURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		url    string
		expect string
	}{
		{
			name:   "empty",
			url:    "",
			expect: "",
		},
		{
			name:   "git connection",
			url:    "git@git:my-awesome-project.git",
			expect: "my-awesome-project",
		},
		{
			name:   "http connection",
			url:    "https://localhost/my-awesome-project.git",
			expect: "my-awesome-project",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual := cleanGitURL(tc.url)
			assert.Equal(t, tc.expect, actual, "Must match the expected value")
		})
	}
}
