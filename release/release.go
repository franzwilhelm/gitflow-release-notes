package release

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/franzwilhelm/gitflow-release-notes/gitflow"
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
	return r.Tag.Data.Name
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
		case gitflow.Feature:
			feature = append(feature, pr)
		case gitflow.Bugfix:
			bugfix = append(bugfix, pr)
		case gitflow.Hotfix:
			hotfix = append(hotfix, pr)
		case gitflow.Release:
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
	return writeMarkdownSection(w, "Other", other)
}

func writeMarkdownSection(w io.Writer, title string, prs []github.PullRequest) error {
	if prs == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "## %s:\n", title); err != nil {
		return err
	}
	for _, pr := range prs {
		if _, err := fmt.Fprintf(w, "#### [#%v](%s): %s\n%s\n\n",
			pr.GetNumber(),
			pr.GetHTMLURL(),
			gitflow.RemovePrefixes(pr.GetTitle()),
			pr.GetBody()); err != nil {
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
func GenerateReleasesBetweenTags(baseVersion, headVersion *version.Version, tagPrefix string) ([]Release, error) {
	// Fetch the latest 100 tags and pull requests
	tags, err := githubutil.GetTags()
	if err != nil {
		return nil, fmt.Errorf("could not fetch tags: %v", err)
	}
	prMap, err := githubutil.GetPullRequests()
	if err != nil {
		return nil, fmt.Errorf("could not fetch pull requests: %v", err)
	}

	// Find the tag before the base version and use it as the new base
	for i, tag := range tags {
		if tag.Version.Equal(baseVersion) {
			baseVersion = tags[i+1].Version
			break
		}
	}
	baseTag := tagPrefix + baseVersion.String()
	headTag := tagPrefix + headVersion.String()

	// Fetch all commits between the two tags we're interested in
	commits, err := githubutil.CompareCommits(baseTag, headTag)
	if err != nil {
		return nil, fmt.Errorf("could not get commits between tag '%s' and '%s': %v", baseTag, headTag, err)
	}

	var releases []Release
	j := 0
	for i := len(tags) - 1; i >= 0; i-- {
		release := Release{
			Tag:        tags[i],
			Repository: githubutil.Repo,
		}
		if !tags[i].IsBetween(baseVersion, headVersion) {
			continue
		}
		found := false
		for ; j < len(commits); j++ {
			commit := commits[j]
			release.Commits = append(release.Commits, commit)
			if pr, ok := prMap[commit.GetSHA()]; ok {
				release.PullRequests = append(release.PullRequests, *pr)
			}
			if commit.GetSHA() == tags[i].Data.Target.Sha {
				found = true
				j++
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Could not find the commit for tag %v. Is the original tag commit deleted?", tags[i])
		}
		releases = append(releases, release)
	}
	return releases, nil
}
