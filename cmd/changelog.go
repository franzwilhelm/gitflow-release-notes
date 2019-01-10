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
	"fmt"
	"regexp"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/franzwilhelm/gitflow-release-notes/githubutil"
	"github.com/google/go-github/github"
)

// changelogCmd represents the changelog command
var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		githubutil.Initialize(client, repo, owner)

		prs, err := githubutil.GetPullRequests()
		if err != nil {
			logrus.WithError(err).Error("Could not fetch pull requests")
		}
		prMap := make(map[int]*github.PullRequest)

		for _, pr := range prs {
			prMap[*pr.Number] = pr
		}

		commits, err := githubutil.GetCommitsBetweenTags("v1.11.0", "v1.11.25")
		if err != nil {
			logrus.WithError(err).Error("Could not get commits between tags")
		}

		for _, commit := range commits {
			msg := *commit.Commit.Message
			match := regexp.
				MustCompile(`(\(|request )#(\d*)`).
				FindStringSubmatch(msg)

			if len(match) != 0 {
				issueNumber, _ := strconv.Atoi(match[len(match)-1])
				pr := prMap[issueNumber]
				fmt.Println("PR NUMBER:", issueNumber)
				fmt.Println("TITLE:", *pr.Title)
				fmt.Println("BODY:",*pr.Body)
				fmt.Println("")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(changelogCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// changelogCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// changelogCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
