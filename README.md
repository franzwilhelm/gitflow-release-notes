# gitflow-release-notes

Automatically generate release notes based on pull requests.

## Background

In order for this tool to work properly, you must follow the GitFlow workflow, at least to some degree. If you're not familiar with the GitFlow workflow, you could have a look at these great articles, giving some great insight:

* https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow
* https://datasift.github.io/gitflow/IntroducingGitFlow.html

GitFlow is a way of structuring pull requests and branches, that intends to simplify the branching workflow in a larger project. This tool assumes you merge branches as PRs with the following setup:

#### Static branhes:
* `master` - Where tags are pushed - represents the stable environment
* `develop` - Where new additions and non-urgent fixes are added

#### Branches based from `develop`
* `feature/[name]` - Branches with this prefix contains features (merge to `develop`)
* `bugfix/[name]` - Branches with this prefix contains bug fixes (merge to `develop`)
* `release/[name]` - Branches with this prefix are created to prepare an upcoming release (merge to `develop` and `master`)

#### Branches based from `master`
* `hotfix/[name]` - Branches with this prefix are urgent to get to the stable environment to fix bugs (merge to `develop` and `master`)

## Installation & Usage

```
go get -u franzwilhelm/gitflow-release-notes
go install $GOPATH/src/github.com/franzwilhelm/gitflow-release-notes
```

To use the tool and get available commands, simply run `gitflow-release-notes -h`.

#### Example
This example generates changelog for release v1.2.3 and pushes is to Github, and to a slack channel.
```shell
gitflow-release-notes changelog v1.2.3 \
  --repository franzwilhelm/gitflow-release-notes \
  --push \ # Push to Github
  --slack-channel $slack_channel \
  --slack-icon $slack_icon \
  --slack-webhook $slack_webhook_url
```

#### Private repositories
For use in private repositories, make sure to [generate a personal access token on Github](https://github.com/settings/tokens). For the tool to work correctly, it needs the following permissions:
* `repo` - _Full control of private repositories_
* `repo:status` - _Access commit status_
* `repo_deployment` - _Access deployment status_
* `public_repo` - _Access public repositories_
* `repo:invite` - _Access repository invitations_
* `read:gpg_key` - _Read user gpg keys_

Then export it as an environment variable `GITHUB_ACCESS_TOKEN`, before running the tool.

## Features

`gitflow-release-notes` already has some great built-in features, and there are more to come!
- [x] Automatic grouping of changelogs by branch name (`feature`/`bugfix`/`hotfix`/`other`)
- [x] Write beautiful changelogs for a single or multiple tags to disk
- [x] Push or overwrite release notes directly to Github
- [x] Push structured release notes to a Slack channel
- [ ] Possible to use a config file instead of flags
- [ ] Possible to customize markdown formatting
- [ ] Use commit messages as backup when no PRs are found for a release

## Contributing

If you would like to contribute, all pull requests are welcome! This repo is currently not GitFlow enabled, so just open PRs directly to master, and I'll drop a review!
