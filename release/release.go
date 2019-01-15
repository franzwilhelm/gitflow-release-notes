package release

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/franzwilhelm/gitflow-release-notes/githubutil"
	"github.com/franzwilhelm/gitflow-release-notes/slack"
	"github.com/google/go-github/github"
	version "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// Release is a wrapper of github data containing merged prs and commits between
// two tags. The tag of the release is the one to create release notes for
type Release struct {
	Tag          githubutil.Tag
	Repository   githubutil.Repository
	Commits      []github.RepositoryCommit
	PullRequests []github.PullRequest
}

// Filename returns an appropriate filename based on the git tag of the release
// For instance tag 'v1.2.3' returns 'v1_2_3.[fileExt]'
func (r *Release) Filename(fileExt string) string {
	dotsRemoved := strings.Replace(r.TagName(), ".", "_", -1)
	return fmt.Sprintf("%s.%s", dotsRemoved, fileExt)
}

// TagName returns the git tag for a release
func (r *Release) TagName() string {
	return r.Tag.Name
}

// GithubURL returns the Github URL for the release
func (r *Release) GithubURL() string {
	return fmt.Sprintf("https://www.github.com/%s/releases/tag/%s", r.Repository.Full(), r.TagName())
}

// GetPullRequestSections returns the different pull request groups of a release
func (r *Release) GetPullRequestSections() (
	feature []github.PullRequest,
	bugfix []github.PullRequest,
	hotfix []github.PullRequest,
	other []github.PullRequest,
) {

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
	return
}

// GenerateMarkdownChangelog writes a markdown changelog file to the provided writer
func (r *Release) GenerateMarkdownChangelog(w io.Writer) error {
	feature, bugfix, hotfix, other := r.GetPullRequestSections()

	if err := writeMarkdownSection(w, "Features", feature); err != nil {
		return err
	}
	if err := writeMarkdownSection(w, "Bug fixes", bugfix); err != nil {
		return err
	}
	if err := writeMarkdownSection(w, "Hotfixes", hotfix); err != nil {
		return err
	}
	if err := writeMarkdownSection(w, "Other", other); err != nil {
		return err
	}
	return nil
}

func writeMarkdownSection(w io.Writer, title string, prs []github.PullRequest) error {
	if prs == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "## %s:\n", title); err != nil {
		return err
	}
	for _, pr := range prs {
		if _, err := fmt.Fprintf(w, "#### [#%v](%s): %s\n%s\n\n", pr.GetNumber(), pr.GetHTMLURL(), pr.GetTitle(), pr.GetBody()); err != nil {
			return err
		}
	}
	return nil
}

// PushToGithub pushes a release to github. If the release already exists,
// it won't be pushed if the overwrite argument is not present
func (r *Release) PushToGithub(overwrite bool) error {
	buf := new(bytes.Buffer)
	if err := r.GenerateMarkdownChangelog(buf); err != nil {
		return err
	}
	release, err := githubutil.GetRelease(r.TagName())
	if err != nil {
		logrus.Infof("Pusing release %s to Github", r.TagName())
		return githubutil.CreateRelease(r.TagName(), buf.String())
	} else if overwrite {
		logrus.Warnf("Overwriting release %s in Github", r.TagName())
		if err := githubutil.DeleteRelease(*release.ID); err != nil {
			return err
		}
		return githubutil.CreateRelease(r.TagName(), buf.String())
	} else {
		logrus.Warnf("Skipping push of existing release %s. Use --overwrite to ignore", r.TagName())
	}
	return nil
}

// PushToSlack pushes release notes to the slack channel specified
func (r *Release) PushToSlack(channel, iconURL string) error {
	feature, bugfix, hotfix, other := r.GetPullRequestSections()

	var attachments []slack.Attachment
	addSlackAttachment(&attachments, "Features", "#315cfd", feature)
	addSlackAttachment(&attachments, "Bug fixes", "#d80f5c", bugfix)
	addSlackAttachment(&attachments, "Hotfixes", "#d80f5c", hotfix)
	addSlackAttachment(&attachments, "", "#2a284f", other)

	return slack.PostWebhook(&slack.WebhookMessage{
		Channel:     channel,
		IconURL:     iconURL,
		Username:    "Release Notes",
		Text:        fmt.Sprintf("New release: <%s|%s@%s> :tada:", r.GithubURL(), r.Repository.Name, r.TagName()),
		Attachments: attachments,
	})
}

func addSlackAttachment(attachments *[]slack.Attachment, title, color string, prs []github.PullRequest) {
	if prs != nil {
		attachment := slack.Attachment{Title: title, Color: color}
		attachment.UsePullRequests(prs)
		*attachments = append(*attachments, attachment)
	}
}

// GenerateReleasesBetweenTags generates a release array containing all releases
// between two tags. For instance sending in v1.10.0 and v1.10.4 will generate
// a release array containing v1.10.0, v1.10.1, v1.10.2, v1.10.3 and v1.10.4
func GenerateReleasesBetweenTags(base, head string) ([]Release, error) {
	// Fetch the latest 100 tags and pull requests
	tags, err := githubutil.GetTags()
	if err != nil {
		return nil, fmt.Errorf("could not fetch tags: %v", err)
	}
	prMap, err := githubutil.GetPullRequests()
	if err != nil {
		return nil, fmt.Errorf("could not fetch pull requests: %v", err)
	}

	for i, tag := range tags {
		if tag.Name == base {
			base = tags[i+1].Name
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
	for i := len(tags) - 1; i >= 0; i-- {
		tagEdge := tags[i]
		release := Release{
			Tag:        tagEdge,
			Repository: githubutil.Repo,
		}
		version, err := version.NewVersion(tagEdge.Name)
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
			if commit.GetSHA() == tagEdge.Target.Sha {
				found = true
				commitIndex++
				break
			}
		}
		if !found {
			logrus.Fatalf("Could not find the commit for tag %v. Is the original tag commit deleted?", tagEdge)
		}
		releases = append(releases, release)
	}
	return releases, nil
}
