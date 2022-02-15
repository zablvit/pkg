package updater

import (
	"context"

	"github.com/ocraviotto/go-scm/scm"
)

// GitUpdater defines the way to apply changes to files in Git.
type GitUpdater interface {
	ApplyUpdateToFile(ctx context.Context, input CommitInput, f ContentUpdater) (string, error)
	CreatePR(ctx context.Context, input PullRequestInput) (*scm.PullRequest, error)
}
