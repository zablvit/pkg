# common

This is a fork of https://github.com/gitops-tools/image-updater, a shared repository for tooling for interacting with Git.

Until merged, this is being maintained as a separated project. The main reason for the fork was the need to support additional operations (key and file removal) and drivers (mainly bitbucket), plus some minor changes for additional parsing of options from the env etc.

In more detail, besides the added functionality for file and key deletion (and the general usage as a yaml updater rather than just an image updater), the main change was swapping jenkins-x/go-scm with a fork of drone/go-scm. The [fork](https://github.com/ocraviotto/go-scm) merged the client factory interface from jenkins-x into drone to make it easier to swap it in here, and was necessary to provide better support for bitbucket and bitbucket cloud (the original project was mainly supporting gitlab and guthub). 

This is used as a library with the also forked and modified [yaml-updater](https://github.com/ocraviotto/yaml-updater) (originally named image-updater)

## Original notice

This is alpha code, just extracted from another project for reuse.

## updater

This provides functionality for updating YAML files with a single call,
including updating the file and optionally opening a PR for the change.

```go
package main

import (
	"context"
	"log"

	"github.com/gitops-tools/common/pkg/client"
	"github.com/gitops-tools/common/pkg/updater"
	"github.com/ocraviotto/go-scm/scm/factory"
	"go.uber.org/zap"
)

func main() {
	cli, err := factory.NewClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	scmClient := client.New(cli)

	logger, _ := zap.NewProduction()
	sugar := logger.Sugar()

	u := updater.New(sugar, scmClient)

	input := updater.Input{
		Repo:               "my-org/my-repo",
		Filename:           "service/deployment.yaml",
		Branch:             "main",
		Key:                "metadata.annotations.reviewed",
		NewValue:           "test-user",
		BranchGenerateName: "test-branch-",
		CommitMessage:      "testing a common component library",
		PullRequest: updater.PullRequestInput{
			Title: "This is a test",
			Body:  "No, really, this is just a test",
		},
	}

	pr, err := u.UpdateYAML(context.Background(), &input)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("pr.Link = %s", pr.Link)
}
```
