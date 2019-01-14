package githubutil

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/github"
	version "github.com/hashicorp/go-version"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
)

var (
	githubDateFormat = "2006-01-02"
	ctx              = context.Background()
	client           *github.Client
	clientv4         *githubv4.Client
	owner            string
	repo             string
)

type Release struct {
	TagEdge      TagEdge
	Commits      []github.RepositoryCommit
	PullRequests []github.PullRequest
}

type TagEdge struct {
	Node struct {
		Name   string
		Target struct {
			Sha       string `graphql:"oid"`
			CommitURL string
		}
	}
}

func Initialize(httpClient *http.Client, repository, repositoryOwner string) {
	client = github.NewClient(httpClient)
	clientv4 = githubv4.NewClient(httpClient)
	repo = repository
	owner = repositoryOwner
}

func SearchQuery(searchMap map[string]interface{}) (query string) {
	for key, value := range searchMap {
		query += fmt.Sprintf(" %s:%v", key, value)
	}
	return query
}

func GraphqlQuery(query map[string]interface{}) map[string]interface{} {
	if query == nil {
		query = make(map[string]interface{})
	}
	query["owner"] = githubv4.String(owner)
	query["repo"] = githubv4.String(repo)
	return query
}

func GetPullRequestsBetween(start, end time.Time) (map[string]github.Issue, error) {
	startFormatted := start.Format(githubDateFormat)
	endFormatted := end.Format(githubDateFormat)
	logrus.Infof("Fetching all pull requests between %s and %s. This may take a while...", startFormatted, endFormatted)
	query := SearchQuery(map[string]interface{}{
		"repo":   fmt.Sprintf("%s/%s", owner, repo),
		"type":   "pr",
		"merged": fmt.Sprintf("%s..%s", startFormatted, endFormatted),
	})
	result, _, err := client.Search.Issues(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	prMap := make(map[string]github.Issue)

	for _, issue := range result.Issues {
		var graphqlResult struct {
			Repository struct {
				PullRequest struct {
					MergeCommit struct {
						Sha string `graphql:"oid"`
					}
				} `graphql:"pullRequest(number: $number)"`
			} `graphql:"repository(owner: $owner, name: $repo)"`
		}

		query := GraphqlQuery(map[string]interface{}{
			"number": githubv4.Int(issue.GetNumber()),
		})
		err = clientv4.Query(ctx, &graphqlResult, query)
		if err != nil {
			return nil, err
		}
		mergeCommitSha := graphqlResult.Repository.PullRequest.MergeCommit.Sha
		prMap[mergeCommitSha] = issue
	}
	logrus.Info("Done fetching all pull requests")
	return prMap, nil
}

func GetPullRequests() (map[string]*github.PullRequest, error) {
	logrus.Infof("Fetching latest 100 pull requests")
	prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State:     "closed",
		Head:      "develop",
		Sort:      "created",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100, // Maximum limit
		},
	})
	if err != nil {
		return nil, err
	}
	prMap := make(map[string]*github.PullRequest)

	for _, pr := range prs {
		prMap[pr.GetMergeCommitSHA()] = pr
	}
	logrus.Warnf("Only prs between %v and %v will be added to changelogs", prs[len(prs)-1].GetNumber(), prs[0].GetNumber())
	return prMap, nil
}

func CompareCommits(base, head string) ([]github.RepositoryCommit, error) {
	logrus.Infof("Fetching all commits between %s and %s", base, head)
	comparision, _, err := client.Repositories.CompareCommits(ctx, owner, repo, base, head)
	if err != nil {
		return nil, err
	}
	return comparision.Commits, nil
}

func ListTags() ([]TagEdge, error) {
	logrus.Info("Fetching the latest 100 tags")
	var graphqlResult struct {
		Repository struct {
			Tags struct {
				Edges []TagEdge
			} `graphql:"refs(refPrefix: \"refs/tags/\", first: 100, direction: DESC)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}
	err := clientv4.Query(ctx, &graphqlResult, GraphqlQuery(nil))
	if err != nil {
		return nil, err
	}
	edges := graphqlResult.Repository.Tags.Edges
	logrus.Warnf("Only tags between %s and %s will generate changelogs", edges[len(edges)-1].Node.Name, edges[0].Node.Name)
	for i := range edges {
		urlSplit := strings.Split(edges[i].Node.Target.CommitURL, "/")
		edges[i].Node.Target.Sha = urlSplit[len(urlSplit)-1]
	}
	return edges, err
}

func GetReleasesBetweenTags(base, head string) ([]Release, error) {
	// Fetch the latest 100 tags and pull requests
	tags, err := ListTags()
	if err != nil {
		return nil, fmt.Errorf("could not fetch tags: %v", err)
	}
	prMap, err := GetPullRequests()
	if err != nil {
		return nil, fmt.Errorf("could not fetch pull requests: %v", err)
	}

	for i, tag := range tags {
		if tag.Node.Name == base {
			base = tags[i+1].Node.Name
			break
		}
	}
	baseVersion, err := version.NewVersion(base)
	if err != nil {
		return nil, err
	}
	headVersion, err := version.NewVersion(head)
	if err != nil {
		return nil, err
	}

	// Fetch all commits between the two tags we're interested in
	commits, err := CompareCommits(base, head)
	if err != nil {
		return nil, fmt.Errorf("could not get commits between tag '%s' and '%s': %v", base, head, err)
	}

	var releases []Release
	commitIndex := 0
	for i := len(tags) - 1; i > 0; i-- {
		tagEdge := tags[i]
		release := Release{
			TagEdge: tagEdge,
		}
		version, err := version.NewVersion(tagEdge.Node.Name)
		if err != nil {
			continue
		}
		if version.LessThan(baseVersion) || version.Equal(baseVersion) || version.GreaterThan(headVersion) {
			continue
		}
		found := false
		for ; commitIndex < len(commits); commitIndex++ {
			commit := commits[commitIndex]
			release.Commits = append(release.Commits, commit)
			if pr, ok := prMap[commit.GetSHA()]; ok {
				release.PullRequests = append(release.PullRequests, *pr)
			}
			if commit.GetSHA() == tagEdge.Node.Target.Sha {
				found = true
				commitIndex++
				break
			}
		}
		if !found {
			logrus.Fatalf("Could not find the commit for tag %v. Is the original tag commit deleted?", tagEdge.Node)
		}
		releases = append(releases, release)
	}
	return releases, nil
}
