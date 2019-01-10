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
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	cfgFile string
	owner   string
	repo    string
	ctx     context.Context
	client  *github.Client
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitflow-release-notes",
	Short: "A brief description of your application",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gitflow-release-notes.yaml)")
	rootCmd.PersistentFlags().StringVarP(&owner, "owner", "o", "", "Github owner")
	rootCmd.PersistentFlags().StringVarP(&repo, "repository", "r", "", "Github repository")
	rootCmd.MarkPersistentFlagRequired("owner")
	rootCmd.MarkPersistentFlagRequired("repository")

	accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")

	if accessToken == "" {
		logrus.Warn("GITHUB_ACCESS_TOKEN not set, using Github Client without auth")
		client = github.NewClient(nil)
	} else {
		ctx = context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gitflow-release-notes" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".gitflow-release-notes")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
