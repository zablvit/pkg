package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ocraviotto/go-scm/scm"
	"github.com/ocraviotto/go-scm/scm/factory"
	"github.com/ocraviotto/pkg/test"
	"gopkg.in/h2non/gock.v1"
)

var _ GitClient = (*SCMClient)(nil)

type ghCommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ghContentS struct {
	Branch    string         `json:"branch"`
	Message   string         `json:"message"`
	Content   *string        `json:"content"`
	Sha       string         `json:"sha"`
	Author    ghCommitAuthor `json:"author"`
	Committer ghCommitAuthor `json:"committer"`
}

func TestGetFile(t *testing.T) {
	gock.New("https://api.github.com").
		Get("/repos/Codertocat/Hello-World/contents/config/my/file.yaml").
		MatchParam("ref", "master").
		Reply(http.StatusOK).
		Type("application/json").
		File("testdata/content.json")
	defer gock.Off()

	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	body, err := client.GetFile(context.TODO(), "Codertocat/Hello-World", "master", "config/my/file.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := mustParseJSONAsContent(t, "testdata/content.json")
	if diff := cmp.Diff(want, body); diff != "" {
		t.Fatalf("got a different body back: %s\n", diff)
	}
}

func TestGetFileWithErrorResponse(t *testing.T) {
	gock.New("https://api.github.com").
		Get("/repos/Codertocat/Hello-World/contents/config/my/file.yaml").
		MatchParam("ref", "master").
		Reply(http.StatusInternalServerError).
		BodyString("not found")
	defer gock.Off()

	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	_, err = client.GetFile(context.TODO(), "Codertocat/Hello-World", "master", "config/my/file.yaml")
	if !test.MatchError(t, `failed to get file.*(500)`, err) {
		t.Fatalf("failed to match error: %s", err)
	}
}

func TestGetFileWithNoServer(t *testing.T) {
	scmClient, err := factory.NewClient("github", "https://localhost:2000", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	_, err = client.GetFile(context.TODO(), "Codertocat/Hello-World", "master", "config/my/file.yaml")
	if !test.MatchError(t, `connect: connection refused`, err) {
		t.Fatalf("failed to match error: %s", err)
	}
}

func TestUpdateFile(t *testing.T) {
	message := "just a test message"
	content := []byte("testing")
	branch := "my-test-branch"
	sha := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	signature := scm.Signature{
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}

	encode := func(b []byte) string {
		return base64.StdEncoding.EncodeToString(b)
	}

	c := encode(content)
	r := ghContentS{
		Branch:  branch,
		Message: message,
		Content: &c,
		Sha:     sha,
		Author: ghCommitAuthor{
			Name:  signature.Name,
			Email: signature.Email,
		},
		Committer: ghCommitAuthor{
			Name:  signature.Name,
			Email: signature.Email,
		},
	}

	gock.New("https://api.github.com").
		Put("/repos/Codertocat/Hello-World/contents/config/my/file.yaml").
		MatchType("json").
		JSON(r).
		Reply(http.StatusCreated).
		Type("application/json").
		File("testdata/content.json")
	defer gock.Off()

	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	err = client.UpdateFile(context.TODO(), "Codertocat/Hello-World", branch,
		"config/my/file.yaml", message, "980a0d5f19a64b4b30a87d4206aade58726b60e3",
		signature, []byte(`testing`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFile(t *testing.T) {
	message := "just another message"
	branch := "my-test-branch"
	sha := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	signature := scm.Signature{
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}

	r := ghContentS{
		Branch:  branch,
		Message: message,
		Sha:     sha,
		Author: ghCommitAuthor{
			Name:  signature.Name,
			Email: signature.Email,
		},
		Committer: ghCommitAuthor{
			Name:  signature.Name,
			Email: signature.Email,
		},
	}

	gock.Observe(gock.DumpRequest)
	gock.New("https://api.github.com").
		Delete("/repos/Codertocat/Hello-World/contents/config/my/file.yaml").
		MatchType("json").
		JSON(r).
		Reply(http.StatusCreated).
		Type("application/json").
		File("testdata/content.json")
	defer gock.Off()

	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	err = client.DeleteFile(context.TODO(), "Codertocat/Hello-World", branch,
		"config/my/file.yaml", message, "980a0d5f19a64b4b30a87d4206aade58726b60e3",
		signature, []byte{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateFileWithNoConnection(t *testing.T) {
	message := "just a test message"
	branch := "my-test-branch"
	signature := scm.Signature{
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}

	scmClient, err := factory.NewClient("github", "https://localhost:2000", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	err = client.UpdateFile(context.TODO(), "Codertocat/Hello-World", branch,
		"config/my/file.yaml", message, "980a0d5f19a64b4b30a87d4206aade58726b60e3",
		signature, []byte(`testing`))
	if !test.MatchError(t, `connect: connection refused`, err) {
		t.Fatalf("failed to match error: %s", err)
	}
}

func TestUpdateFileToPrivateRepositoryWithBadCreds(t *testing.T) {
	message := "just a test message"
	branch := "master"
	repository := "Codertocat/Hello-World"
	signature := scm.Signature{
		Name:  "John Doe",
		Email: "john.doe@example.com",
	}

	scmClient, err := factory.NewClient("github", "", "myfaketoken")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	err = client.UpdateFile(context.TODO(), repository, branch,
		"./README.md", message, "6113728f27ae82c7b1a177c8d03f9e96e0adf246",
		signature, []byte(`testing`))
	if !test.MatchError(t, `failed to update file.*\(401\)$`, err) {
		t.Fatalf("failed to match error: %s", err)
	}
}

func TestCreateBranchInGitHub(t *testing.T) {
	sha := "aa218f56b14c9653891f9e74264a383fa43fefbd"

	gock.New("https://api.github.com").
		Post("/repos/Codertocat/Hello-World/git/refs").
		MatchType("json").
		JSON(map[string]string{"ref": "refs/heads/new-feature", "sha": sha}).
		Reply(http.StatusCreated).
		Type("application/json").
		File("testdata/content.json")
	defer gock.Off()

	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	err = client.CreateBranch(context.Background(), "Codertocat/Hello-World", "new-feature", sha)
	if err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Fatal("branch was not created")
	}
}

func TestCreateBranchInGitLab(t *testing.T) {
	sha := "aa218f56b14c9653891f9e74264a383fa43fefbd"
	branch := "new-feature"

	gock.New("https://gitlab.com").
		Post("/api/v4/projects/Codertocat/Hello-World/repository/branches").
		JSON(map[string]string{"branch": branch, "ref": sha}).
		Reply(http.StatusCreated).
		Type("application/json").
		File("testdata/content.json")
	defer gock.Off()

	scmClient, err := factory.NewClient("gitlab", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	err = client.CreateBranch(context.Background(), "Codertocat/Hello-World", branch, sha)
	if err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Fatal("branch was not created")
	}
}

func TestCreatePullRequest(t *testing.T) {
	title := "Amazing new feature"
	body := "Please pull these awesome changes in!"
	head := "octocat:new-feature"
	base := "master"

	gock.New("https://api.github.com").
		Post("/repos/Codertocat/Hello-World/pulls").
		MatchType("json").
		JSON(map[string]string{"title": title, "body": body, "head": head, "base": base}).
		Reply(http.StatusCreated).
		Type("application/json").
		File("testdata/pr_create.json")
	defer gock.Off()

	input := &scm.PullRequestInput{
		Title:  title,
		Body:   "Please pull these awesome changes in!",
		Source: "octocat:new-feature",
		Target: "master",
	}
	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	_, err = client.CreatePullRequest(context.Background(), "Codertocat/Hello-World", input)
	if err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Fatal("pull request was not created")
	}
}

func TestGetBranchHead(t *testing.T) {
	sha := "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d"
	gock.New("https://api.github.com").
		Get("/repos/Codertocat/Hello-World/branches/master").
		Reply(http.StatusOK).
		Type("application/json").
		File("testdata/github_get_branch.json")
	defer gock.Off()

	scmClient, err := factory.NewClient("github", "", "")
	if err != nil {
		t.Fatal(err)
	}
	client := New(scmClient)

	rSha, err := client.GetBranchHead(context.Background(), "Codertocat/Hello-World", "master")
	if err != nil {
		t.Fatal(err)
	}
	if !gock.IsDone() {
		t.Fatal("ref was not fetched")
	}
	if rSha != sha {
		t.Fatalf("got a different sha back: %s != %s\n", rSha, sha)
	}

}

func mustParseJSONAsContent(t *testing.T, filename string) *scm.Content {
	t.Helper()
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]interface{}{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		t.Fatal(err)
	}
	content, err := base64.StdEncoding.DecodeString(data["content"].(string))
	if err != nil {
		t.Fatal(err)
	}
	return &scm.Content{
		Path:   data["path"].(string),
		BlobID: data["sha"].(string),
		Data:   content,
	}
}
