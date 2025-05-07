// Copyright Splunk, Inc.
// SPDX-License-Identifier: MPL-2.0

package track

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
)

type Details struct {
	name   string
	branch string
	dirty  bool
}

func NewDetailsFromGit(repo *git.Repository) (*Details, error) {
	origin, err := repo.Remote("origin")
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("unable to read head: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("unable to load worktree: %w", err)
	}

	st, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("unable to load current project status: %w", err)
	}
	return &Details{
		name:   cleanGitURL(origin.Config().URLs[0]),
		branch: head.Name().Short(),
		dirty:  !st.IsClean(),
	}, nil
}

func (d Details) Tags() []string {
	return []string{
		fmt.Sprint("project:", d.name),
		fmt.Sprint("branch:", d.branch),
		fmt.Sprint("experimental:", d.dirty),
	}
}

func cleanGitURL(connection string) string {
	switch {
	case strings.HasPrefix(connection, "http"):
		if u, err := url.Parse(connection); err == nil {
			connection = strings.TrimPrefix(u.Path, "/")
		}
	case strings.HasPrefix(connection, "git@"):
		_, project, _ := strings.Cut(connection, ":")
		connection = project
	}
	return strings.TrimSuffix(connection, ".git")
}
