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

	"github.com/franzwilhelm/gitflow-release-notes/githubutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	fromTag string
	toTag string
)

// changelogCmd represents the changelog command
var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		githubutil.Initialize(client, repo, owner)
		prs, err := githubutil.GetPrsBetweenTags(fromTag, toTag)
		if err != nil {
			logrus.WithError(err).Fatalf("Could not fetch pull requests")
		}
		fmt.Println(prs)
	},
}

func init() {
	rootCmd.AddCommand(changelogCmd)
	changelogCmd.Flags().StringVarP(&fromTag, "from-tag", "f", "", "Tag to check from")
	changelogCmd.Flags().StringVarP(&toTag, "to-tag", "t", "", "Tag to check to")
	changelogCmd.MarkFlagRequired("from-tag")
	changelogCmd.MarkFlagRequired("to-tag")
}
