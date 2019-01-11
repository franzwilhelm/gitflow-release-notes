package githubutil

import (
	"context"
	"fmt"

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

func GetPullRequests() (map[string]*github.PullRequest, error) {
	prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State:     "closed",
		Head:      "develop",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100, // Maximum limit
		},
	})
	prMap := make(map[string]*github.PullRequest)

	for _, pr := range prs {
		prMap[pr.GetMergeCommitSHA()] = pr
	}
	return prMap, err
}

func GetCommitsBetweenTags(base, head string) ([]github.RepositoryCommit, error) {
	comparision, _, err := client.Repositories.CompareCommits(ctx, owner, repo, base, head)
	return comparision.Commits, err
}

func GetPrsBetweenTags(base, head string) ([]github.PullRequest, error) {
	prMap, err := GetPullRequests()
	if err != nil {
		return nil, fmt.Errorf("Could not fetch pull requests: %v", err)
	}

	commits, err := GetCommitsBetweenTags(base, head)
	if err != nil {
		return nil, fmt.Errorf("Could not get commits between tag '%s' and '%s': %v", base, head, err)
	}

	var prs []github.PullRequest
	for _, commit := range commits {
		pr, ok := prMap[*commit.SHA]
		if ok {
			prs = append(prs, *pr)
		}
	}
	return prs, nil
}
