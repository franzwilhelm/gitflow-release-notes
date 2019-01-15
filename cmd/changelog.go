// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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
	"os"
	"strings"

	"github.com/franzwilhelm/gitflow-release-notes/release"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	base         string
	head         string
	overwrite    bool
	push         bool
	slackChannel string
)

// changelogCmd represents the changelog command
var changelogCmd = &cobra.Command{
	Use:   "changelog [base-tag..head-tag]",
	Short: "Generates changelogs for the specified tag or tag range",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		versionSpec := args[0]
		tags := strings.Split(versionSpec, "..")
		base := tags[0]

		repoOwnerLog := logrus.WithFields(logrus.Fields{
			"repo":  repo.Name,
			"owner": repo.Owner,
		})
		if len(tags) == 2 {
			head = tags[1]
			repoOwnerLog.Infof("Generating changelog for tags between %s and %s", base, head)
		} else if len(tags) == 1 {
			head = tags[0]
			repoOwnerLog.Infof("Generating changelog for %s", base)
		} else {
			logrus.Fatal("Bad input argument format")
		}

		releases, err := release.GenerateReleasesBetweenTags(base, head)
		if err != nil {
			logrus.WithError(err).Fatalf("Could not generate releases")
		}
		for _, release := range releases {
			if push {
				if err := release.PushToGithub(overwrite); err != nil {
					logrus.WithError(err).Fatal("Could not push release to Github")
				}
			} else {
				filename := release.Filename("md")
				f, err := os.Create(filename)
				if err != nil {
					logrus.WithError(err).Error("Could not create file for changelog")
				}
				defer f.Close()
				release.GenerateMarkdownChangelog(f)
				logrus.Infof("Wrote changelog to %s", filename)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(changelogCmd)
	changelogCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing tags in Github if necessary")
	changelogCmd.Flags().BoolVar(&push, "push", false, "Push changelog to github instead of saving it locally")
	changelogCmd.Flags().StringVar(&slackChannel, "slack-channel", "", "Post release notes to a slack channel")
	changelogCmd.MarkFlagRequired("tag")
}
