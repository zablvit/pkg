package updater

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-logr/logr"
	"github.com/ocraviotto/go-scm/scm"

	"github.com/ocraviotto/pkg/client"
	"github.com/ocraviotto/pkg/names"
)

// ContentUpdater takes an existing body, it should transform it, and return the
// updated body.
type ContentUpdater func([]byte) ([]byte, error)

// UpdaterFunc is an option for for creating new Updaters.
type UpdaterFunc func(u *Updater)

// CommitInput is used to configure the commit and pull request.
type CommitInput struct {
	Repo               string        // e.g. my-org/my-repo
	Filename           string        // relative path to the file in the repository
	Branch             string        // e.g. main
	BranchGenerateName string        // e.g. gitops-
	DisablePRCreation  bool          // Whether to disable PR creation
	CreateMissing      bool          // Whether to create the target file if it's missing
	RemoveFile         bool          // Whether to remove the target file
	CommitMessage      string        // This is used for the commit when updating the file
	Signature          scm.Signature // This identifies a git commit creator
}

// PullRequestInput provides configuration for the PullRequest to be opened.
type PullRequestInput struct {
	SourceBranch string // e.g. 'main'
	NewBranch    string
	Repo         string // e.g. my-org/my-repo
	Title        string
	Body         string
}

var timeSeed = rand.New(rand.NewSource(time.Now().UnixNano()))

// NameGenerator is an option func for the Updater creation function.
func NameGenerator(g names.Generator) UpdaterFunc {
	return func(u *Updater) {
		u.nameGenerator = g
	}
}

// New creates and returns a new Updater.
func New(l logr.Logger, c client.GitClient, opts ...UpdaterFunc) *Updater {
	u := &Updater{gitClient: c, nameGenerator: names.New(timeSeed), log: l}
	for _, o := range opts {
		o(u)
	}
	return u
}

// Updater can update a Git repo with an updated version of a file.
type Updater struct {
	gitClient     client.GitClient
	nameGenerator names.Generator
	log           logr.Logger
}

// ApplyUpdateToFile does the job of fetching a file, passing it to a
// user-provided function if not deleting it, and optionally creating a PR.
func (u *Updater) ApplyUpdateToFile(ctx context.Context, input CommitInput, f ContentUpdater) (string, error) {
	var (
		updated         []byte
		isNotFoundError bool
		currentSHA      string
	)

	isFileOp := input.RemoveFile || input.CreateMissing
	current, err := u.gitClient.GetFile(ctx, input.Repo, input.Branch, input.Filename)
	if err != nil {
		isNotFoundError = client.IsNotFound(err)
		if !isNotFoundError || (isNotFoundError && !isFileOp) {
			u.log.Info("failed to get file from repo", "err", err)
			return "", err
		}
	}
	if isNotFoundError && input.RemoveFile {
		return "", fmt.Errorf("removing a non-existing file %s in branch %s is not necessary", input.Filename, input.Branch)
	}
	if current.Sha != "" {
		currentSHA = current.Sha
		u.log.Info("got existing file", "sha", current.Sha)
	} else if isNotFoundError {
		currentSHA, err = u.gitClient.GetBranchHead(ctx, input.Repo, input.Branch)
		if err != nil {
			u.log.Info("unable to get parent sha for branch, if branch is main, it may still succeed", "err", err, "branch", input.Branch)
		}
	}
	updated, err = f(current.Data)
	if err != nil {
		return "", fmt.Errorf("failed to apply update: %v", err)
	}

	return u.applyUpdate(ctx, input, currentSHA, updated)
}

func (u *Updater) applyUpdate(ctx context.Context, input CommitInput, currentSHA string, newBody []byte) (string, error) {
	branchRef, err := u.gitClient.GetBranchHead(ctx, input.Repo, input.Branch)
	if err != nil {
		return "", fmt.Errorf("failed to get branch head: %v", err)
	}
	newBranchName, err := u.createBranchIfNecessary(ctx, input, branchRef)
	if err != nil {
		return "", err
	}

	if input.RemoveFile {
		err = u.gitClient.DeleteFile(ctx, input.Repo, newBranchName, input.Filename, input.CommitMessage, currentSHA, input.Signature, newBody)
		if err != nil {
			return "", fmt.Errorf("failed to delete file: %w", err)
		}
		u.log.Info("deleted file", "filename", input.Filename)
		return newBranchName, nil
	}

	err = u.gitClient.UpdateFile(ctx, input.Repo, newBranchName, input.Filename, input.CommitMessage, currentSHA, input.Signature, newBody)
	if err != nil {
		return "", fmt.Errorf("failed to update file: %w", err)
	}
	u.log.Info("updated file", "filename", input.Filename)
	return newBranchName, nil

}

func (u *Updater) createBranchIfNecessary(ctx context.Context, input CommitInput, sourceRef string) (string, error) {
	if input.DisablePRCreation {
		u.log.Info("DisablePRCreation set, committing directly to source branch", "branch", input.Branch)
		return input.Branch, nil
	}

	newBranchName := u.nameGenerator.PrefixedName(input.BranchGenerateName)
	u.log.Info("generating new branch", "name", newBranchName)
	err := u.gitClient.CreateBranch(ctx, input.Repo, newBranchName, sourceRef)
	if err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}
	u.log.Info("created branch", "branch", newBranchName, "ref", sourceRef)
	return newBranchName, nil
}

func (u *Updater) CreatePR(ctx context.Context, input PullRequestInput) (*scm.PullRequest, error) {
	pr, err := u.gitClient.CreatePullRequest(ctx, input.Repo, &scm.PullRequestInput{
		Title:  input.Title,
		Body:   input.Body,
		Source: input.NewBranch,
		Target: input.SourceBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a pull request: %w", err)
	}
	u.log.Info("created PullRequest", "number", pr.Number)
	return pr, nil
}
