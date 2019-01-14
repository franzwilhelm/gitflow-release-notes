package release

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/franzwilhelm/gitflow-release-notes/githubutil"
	"github.com/google/go-github/github"
	version "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

type Release struct {
	TagEdge      githubutil.TagEdge
	Commits      []github.RepositoryCommit
	PullRequests []github.PullRequest
}

func (r *Release) GenerateMarkdown() {
	var (
		feature []github.PullRequest
		bugfix  []github.PullRequest
		hotfix  []github.PullRequest
		other   []github.PullRequest
	)
	for _, pr := range r.PullRequests {
		branchName := pr.Head.GetRef()
		prefix := strings.Split(branchName, "/")[0]
		switch prefix {
		case "feature":
			feature = append(feature, pr)
		case "bugfix":
			bugfix = append(bugfix, pr)
		case "hotfix":
			hotfix = append(hotfix, pr)
		case "release":
			// skip
		default:
			other = append(other, pr)
		}
	}
	document := generateSection("Features", feature) +
		generateSection("Bug fixes", bugfix) +
		generateSection("Hotfixes", hotfix) +
		generateSection("Other", other)

	err := ioutil.WriteFile(r.Filename()+".md", []byte(document), 0644)
	if err != nil {
		logrus.WithError(err).Errorf("Could not write changelog for %s", r.TagName())
	} else {
		logrus.Infof("Changelog for %s written to %s", r.TagName(), r.Filename()+".md")
	}
}

func (r *Release) Filename() string {
	return strings.Replace(r.TagName(), ".", "_", -1)
}

func (r *Release) TagName() string {
	return r.TagEdge.Node.Name
}

func GenerateReleasesBetweenTags(base, head string) ([]Release, error) {
	// Fetch the latest 100 tags and pull requests
	tags, err := githubutil.ListTags()
	if err != nil {
		return nil, fmt.Errorf("could not fetch tags: %v", err)
	}
	prMap, err := githubutil.GetPullRequests()
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
	commits, err := githubutil.CompareCommits(base, head)
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

func generateSection(title string, prs []github.PullRequest) (section string) {
	if prs == nil {
		return ""
	}
	section += fmt.Sprintf("## %s:\n", title)
	for _, pr := range prs {
		section += fmt.Sprintf("#### [#%v](%s): %s\n", pr.GetNumber(), pr.GetHTMLURL(), pr.GetTitle())
		section += pr.GetBody()
		section += "\n\n"
	}
	return section
}
