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
	"github.com/franzwilhelm/gitflow-release-notes/githubutil"
	"github.com/franzwilhelm/gitflow-release-notes/release"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	fromTag string
	toTag   string
)

// changelogCmd represents the changelog command
var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		if fromTag == "" {
			fromTag = toTag
		}
		githubutil.Initialize(httpClient, repo, owner)
		releases, err := release.GenerateReleasesBetweenTags(fromTag, toTag)
		if err != nil {
			logrus.WithError(err).Fatalf("Could not generate releases")
		}
		for _, release := range releases {
			release.GenerateMarkdown()
		}
	},
}

func init() {
	rootCmd.AddCommand(changelogCmd)
	changelogCmd.Flags().StringVarP(&fromTag, "from-tag", "f", "", "If specified, changelogs are generated from all releases between this tag, and the other tag specified.")
	changelogCmd.Flags().StringVarP(&toTag, "tag", "t", "", "Tag to generate changelog for")
	changelogCmd.MarkFlagRequired("tag")
}
