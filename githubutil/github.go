package githubutil

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	Repo             Repository
	githubDateFormat = "2006-01-02"
	ctx              = context.Background()
	client           *github.Client
	clientv4         *githubv4.Client
)

// Initialize initializes the github and githubv4 clients
// If a Github authorized httpClient is passed as an argument, the github
// clients will be able to fetch data from private resources
func Initialize(accessToken string, r Repository) {
	var httpClient *http.Client
	if accessToken == "" {
		logrus.Warn("GITHUB_ACCESS_TOKEN not set, using Github Client without auth")
	} else {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		)
		httpClient = oauth2.NewClient(ctx, ts)
	}

	client = github.NewClient(httpClient)
	clientv4 = githubv4.NewClient(httpClient)
	Repo = r
}

// GetPullRequestIssuesBetween fetches all pull request issues between two timestamps,
// using the Github Search api. Returns a map of the merge commit SHAs and the issues.
func GetPullRequestIssuesBetween(start, end time.Time) (map[string]github.Issue, error) {
	startFormatted := start.Format(githubDateFormat)
	endFormatted := end.Format(githubDateFormat)
	logrus.Infof("Fetching all pull requests between %s and %s. This may take a while...", startFormatted, endFormatted)
	query := searchQuery(map[string]interface{}{
		"repo":   fmt.Sprintf("%s/%s", Repo.Owner, Repo.Name),
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

		query := graphqlQuery(map[string]interface{}{
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

// GetPullRequests fetches the 100 most recent pull requests
func GetPullRequests() (map[string]*github.PullRequest, error) {
	logrus.Infof("Fetching latest 100 pull requests")
	prs, _, err := client.PullRequests.List(ctx, Repo.Owner, Repo.Name, &github.PullRequestListOptions{
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

// Tag holds data about a git tag and it's target commit sha / url
type Tag struct {
	Name   string
	Target struct {
		Sha       string `graphql:"oid"`
		CommitURL string
	}
}

// GetTags fetches the 100 most recent tags
func GetTags() ([]Tag, error) {
	logrus.Info("Fetching the latest 100 tags")
	var graphqlResult struct {
		Repository struct {
			Tags struct {
				Edges []struct {
					Node Tag
				}
			} `graphql:"refs(refPrefix: \"refs/tags/\", first: 100, direction: DESC)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}
	err := clientv4.Query(ctx, &graphqlResult, graphqlQuery(nil))
	if err != nil {
		return nil, err
	}
	edges := graphqlResult.Repository.Tags.Edges
	logrus.Warnf("Only tags between %s and %s will generate changelogs", edges[len(edges)-1].Node.Name, edges[0].Node.Name)

	var tags []Tag
	for _, edge := range edges {
		urlSplit := strings.Split(edge.Node.Target.CommitURL, "/")
		edge.Node.Target.Sha = urlSplit[len(urlSplit)-1]
		tags = append(tags, edge.Node)
	}
	return tags, err
}

// CompareCommits returns all commits between two github tags or hashes
func CompareCommits(base, head string) ([]github.RepositoryCommit, error) {
	logrus.Infof("Fetching all commits between %s and %s", base, head)
	comparision, _, err := client.Repositories.CompareCommits(ctx, Repo.Owner, Repo.Name, base, head)
	if err != nil {
		return nil, err
	}
	return comparision.Commits, nil
}

// CreateRelease creates a release in Github
func CreateRelease(tagName, body string) error {
	_, _, err := client.Repositories.CreateRelease(ctx, Repo.Owner, Repo.Name, &github.RepositoryRelease{
		TagName: &tagName,
		Name:    &tagName,
		Body:    &body,
	})
	return err
}

// GetRelease fetches a release in Github by tag
func GetRelease(tag string) (*github.RepositoryRelease, error) {
	release, _, err := client.Repositories.GetReleaseByTag(ctx, Repo.Owner, Repo.Name, tag)
	return release, err
}

// DeleteRelease deletes a release in Github
func DeleteRelease(id int64) error {
	_, err := client.Repositories.DeleteRelease(ctx, Repo.Owner, Repo.Name, id)
	return err
}

func searchQuery(searchMap map[string]interface{}) (query string) {
	for key, value := range searchMap {
		query += fmt.Sprintf(" %s:%v", key, value)
	}
	return query
}

func graphqlQuery(query map[string]interface{}) map[string]interface{} {
	if query == nil {
		query = make(map[string]interface{})
	}
	query["owner"] = githubv4.String(Repo.Owner)
	query["repo"] = githubv4.String(Repo.Name)
	return query
}
