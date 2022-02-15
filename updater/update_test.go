package updater

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ocraviotto/go-scm/scm"
	"github.com/ocraviotto/pkg/client"
	"github.com/ocraviotto/pkg/client/mock"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	testGitHubRepo = "testorg/testrepo"
	testFilePath   = "environments/test/services/service-a/test.yaml"
	testBranch     = "main"
)

var _ GitUpdater = (*Updater)(nil)

func TestApplyUpdateToFile(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	updater := New(zap.New(), m, NameGenerator(stubNameGenerator{"a"}))
	newBody := []byte("new content")

	branch, err := updater.ApplyUpdateToFile(context.Background(), makeCommitInput(), func([]byte) ([]byte, error) {
		return newBody, nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if branch != "test-branch-a" {
		t.Fatalf("newly created branch, got %#v, want %#v", branch, "test-branch-a")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != string(newBody) {
		t.Fatalf("update failed, got %#v, want %#v", s, newBody)
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertNoPullRequestsCreated()
}

func TestApplyUpdateToFileMissingFile(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	updater := New(zap.New(), m, NameGenerator(stubNameGenerator{"a"}))
	testErr := errors.New("missing file")
	m.GetFileErr = testErr

	_, err := updater.ApplyUpdateToFile(context.Background(), makeCommitInput(), func([]byte) ([]byte, error) {
		return []byte("testing"), nil
	})

	if err != testErr {
		t.Fatalf("got %s, want %s", err, testErr)
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertNoBranchesCreated()
	m.AssertNoPullRequestsCreated()
}

func TestApplyUpdateToFileMissingWithCreate(t *testing.T) {
	createBranch := "test-branch-a"
	content := "testing"
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	updater := New(zap.New(), m, NameGenerator(stubNameGenerator{"a"}))
	testErr := client.SCMError{
		Msg:    fmt.Sprintf("failed to get file %s from repo %s ref %s", testFilePath, testGitHubRepo, testSHA),
		Status: 404,
	}
	m.GetFileErr = testErr
	input := makeCommitInput()
	input.CreateMissing = true

	newBranch, err := updater.ApplyUpdateToFile(context.Background(), input, func([]byte) ([]byte, error) {
		return []byte(content), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if s := newBranch; s != createBranch {
		t.Fatalf("failed to create branch, got %#v, want %#v", createBranch, newBranch)
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, createBranch)
	if s := string(updated); s != content {
		t.Fatalf("update failed, got %#v, want %#v", s, content)
	}
	m.AssertBranchCreated(testGitHubRepo, createBranch, testSHA)
}

func TestKeyRemoval(t *testing.T) {
	createBranch := "test-branch-a"
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	updater := New(zap.New(), m, NameGenerator(stubNameGenerator{"a"}))
	newBody := []byte("test: {}\n")
	input := makeCommitInput()

	branch, err := updater.ApplyUpdateToFile(context.Background(), input, RemoveYAMLKey("test.image"))
	if err != nil {
		t.Fatal(err)
	}
	if branch != createBranch {
		t.Fatalf("newly created branch, got %#v, want %#v", branch, createBranch)
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, createBranch)
	if s := string(updated); s != string(newBody) {
		t.Fatalf("update failed, got %#v, want %#v", s, string(newBody))
	}
	m.AssertBranchCreated(testGitHubRepo, createBranch, testSHA)
	m.AssertNoPullRequestsCreated()
}

func TestApplyUpdateToFileWithBranchCreationFailure(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	updater := New(zap.New(), m, NameGenerator(stubNameGenerator{"a"}))
	testErr := errors.New("can't create branch")
	m.CreateBranchErr = testErr

	_, err := updater.ApplyUpdateToFile(context.Background(), makeCommitInput(), func([]byte) ([]byte, error) {
		return []byte("testing"), nil
	})

	if err.Error() != "failed to create branch: can't create branch" {
		t.Fatalf("got %s, want %s", err, "failed to create branch: can't create branch")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertNoBranchesCreated()
	m.AssertNoPullRequestsCreated()
}

func TestCreatePullRequest(t *testing.T) {
	m := mock.New(t)
	updater := New(zap.New(), m, NameGenerator(stubNameGenerator{"a"}))
	input := makePullRequestInput()

	pr, err := updater.CreatePR(context.Background(), input)

	if err != nil {
		t.Fatal(err)
	}
	m.AssertPullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title:  input.Title,
		Body:   input.Body,
		Source: "test-branch-a",
		Target: testBranch,
	})
	if pr.Link != "https://example.com/pull-request/1" {
		t.Fatalf("link to PR is incorrect: got %#v, want %#v", pr.Link, "https://example.com/pull-request/1")
	}
}

func TestCreatePullRequestHandlingErrors(t *testing.T) {
	m := mock.New(t)
	updater := New(zap.New(), m, NameGenerator(stubNameGenerator{"a"}))
	input := makePullRequestInput()
	testErr := errors.New("can't create pull-request")
	m.CreatePullRequestErr = testErr

	_, err := updater.CreatePR(context.Background(), input)

	if err.Error() != "failed to create a pull request: can't create pull-request" {
		t.Fatalf("got %s, want %s", err, "failed to create a pull request: can't create pull-request")
	}
}

type stubNameGenerator struct {
	name string
}

func (s stubNameGenerator) PrefixedName(p string) string {
	return p + s.name
}

func makeCommitInput() CommitInput {
	return CommitInput{
		Repo:               testGitHubRepo,
		Filename:           testFilePath,
		Branch:             testBranch,
		BranchGenerateName: "test-branch-",
		CommitMessage:      "just a test commit",
	}
}

func makePullRequestInput() PullRequestInput {
	return PullRequestInput{
		Repo:         testGitHubRepo,
		NewBranch:    "test-branch-a",
		SourceBranch: testBranch,
		Title:        "This is a test PR",
		Body:         "This is the body",
	}
}
