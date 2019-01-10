package githubutil

import (
	"context"

	"github.com/google/go-github/github"
)

var (
	ctx    = context.Background()
	client *github.Client
	owner  string
	repo   string
)

func Initialize(c *github.Client, repository, repositoryOwner string) {
	client = c
	repo = repository
	owner = repositoryOwner
}

func GetPullRequests() ([]*github.PullRequest, error) {
	prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State:     "closed",
		Head:      "develop",
		Sort:      "updated",
		Direction: "desc",
	})
	return prs, err
}

func GetCommitsBetweenTags(base, head string) ([]github.RepositoryCommit, error) {
	comparision, _, err := client.Repositories.CompareCommits(ctx, owner, repo, base, head)
	return comparision.Commits, err
}
