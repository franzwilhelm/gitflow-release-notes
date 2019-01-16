// Copyright © 2019 Franz von der Lippe franz.vonderlippe@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/franzwilhelm/gitflow-release-notes/release"
	"github.com/franzwilhelm/gitflow-release-notes/slack"
	version "github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	overwrite       bool
	pushToGithub    bool
	saveMarkdown    bool
	slackChannel    string
	slackWebhookURL string
	slackIconURL    string
)

func parseTagInput(input string) (baseVersion, headVersion *version.Version, versionPrefix string, err error) {
	tags := strings.Split(input, "..")
	baseVersion, err = version.NewVersion(tags[0])
	if err != nil {
		return nil, nil, "", err
	}
	switch len(tags) {
	case 1:
		headVersion = baseVersion
	case 2:
		headVersion, err = version.NewVersion(tags[1])
		if err != nil {
			return nil, nil, "", err
		}
	default:
		return nil, nil, "", errors.New("input argument should only contain one double dot (..)")
	}

	return baseVersion, headVersion, strings.Replace(tags[0], baseVersion.String(), "", 1), nil
}

// changelogCmd represents the changelog command
var changelogCmd = &cobra.Command{
	Use:   "changelog [base-tag..head-tag]",
	Short: "Generates changelogs for the specified tag or tag range",
	Args:  cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if slackChannel != "" && slackWebhookURL == "" {
			return errors.New("--slack-webhook is needed to post to slack")
		} else if slackChannel != "" {
			slack.Initialize(slackWebhookURL)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		baseVersion, headVersion, tagPrefix, err := parseTagInput(args[0])
		if err != nil {
			logrus.WithError(err).Fatal("Could not parse tag input")
		}

		repoOwnerLog := logrus.WithFields(logrus.Fields{
			"repo":  repo.Name,
			"owner": repo.Owner,
		})
		if baseVersion.Equal(headVersion) {
			repoOwnerLog.Infof("Generating changelog for %s", tagPrefix+baseVersion.String())
		} else {
			repoOwnerLog.Infof("Generating changelog for tags between %s and %s", tagPrefix+baseVersion.String(), tagPrefix+headVersion.String())
		}

		releases, err := release.GenerateReleasesBetweenTags(baseVersion, headVersion, tagPrefix)
		if err != nil {
			logrus.WithError(err).Fatalf("Could not generate releases")
		}

		pushToSlack := slackWebhookURL != "" && slackChannel != ""
		for _, release := range releases {
			log := logrus.WithField("release", release.TagName())
			if pushToGithub {
				if err := release.PushToGithub(overwrite); err != nil {
					log.WithError(err).Error("Could not push release to Github")
				}
			}
			if pushToSlack {
				log.Info("Pusing release to slack")
				if err := release.PushToSlack(slackChannel, slackIconURL); err != nil {
					log.WithError(err).Error("Could not push release to slack")
				}
			}
			if saveMarkdown {
				filename := release.Filename("md")
				if f, err := os.Create(filename); err != nil {
					log.WithError(err).Error("Could not create file for changelog")
				} else {
					defer f.Close()
					if err := release.GenerateMarkdownChangelog(f); err != nil {
						log.WithError(err).Error("Could not generate markdown changelog")
					} else {
						log.Infof("Wrote changelog to %s", filename)
					}
				}
			} else if !pushToGithub && !pushToSlack {
				buf := new(bytes.Buffer)
				if err := release.GenerateMarkdownChangelog(buf); err != nil {
					log.WithError(err).Error("Could not generate markdown changelog")
				} else {
					fmt.Println(buf.String())
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(changelogCmd)
	changelogCmd.Flags().BoolVar(&pushToGithub, "push", false, "Push changelog to Github instead of saving it locally")
	changelogCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing tags in Github if necessary")
	changelogCmd.Flags().BoolVarP(&saveMarkdown, "save", "s", false, "Save the release notes to files")
	changelogCmd.Flags().StringVarP(&slackChannel, "slack-channel", "c", "", "Post release notes to a slack channel")
	changelogCmd.Flags().StringVarP(&slackWebhookURL, "slack-webhook", "w", "", "A slack webhook URL")
	changelogCmd.Flags().StringVarP(&slackIconURL, "slack-icon", "i", "", "A URL containing the icon which will appear in the slack message")
}
