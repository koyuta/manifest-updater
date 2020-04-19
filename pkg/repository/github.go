package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

var nowFunc = time.Now

var (
	BranchName      = "feature/update-tag"
	DefaultCloneDir = "/tmp"
)

type GitHubRepository struct {
	URL    string     `json:"url"`
	Branch string     `json:"branch"`
	Path   string     `json:"path,omitempty"`
	Auth   GithubAuth `json:"-"`
}

type GithubAuth struct {
	User  string
	Token string
}

func NewGitHubRepository(url, branch, path string, auth GithubAuth) *GitHubRepository {
	if branch == "" {
		branch = "master"
	}
	if path == "" {
		path = "/"
	}
	return &GitHubRepository{
		URL:    url,
		Branch: branch,
		Path:   path,
		Auth:   auth,
	}
}

func (g *GitHubRepository) PushReplaceTagCommit(ctx context.Context, image, tag string) error {
	endpoint, err := transport.NewEndpoint(g.URL)
	if err != nil {
		return err
	}
	clonepath := filepath.Join(
		DefaultCloneDir,
		g.extractOwnerFromEndpoint(endpoint),
		g.extractRepositoryFromEndpoint(endpoint),
	)

	var auth transport.AuthMethod
	if endpoint.Protocol == "https" && g.Auth.User != "" && g.Auth.Token != "" {
		auth = &http.BasicAuth{Username: g.Auth.User, Password: g.Auth.Token}
	}

	branch := plumbing.NewBranchReferenceName(g.Branch)

	var repository *git.Repository
	if _, err := os.Stat(clonepath); os.IsNotExist(err) {
		opts := &git.CloneOptions{
			Auth:          auth,
			URL:           g.URL,
			SingleBranch:  true,
			ReferenceName: branch,
		}
		repository, err = git.PlainCloneContext(ctx, clonepath, false, opts)
		if err != nil {
			return err
		}
	} else {
		repository, err = git.PlainOpen(clonepath)
		if err != nil {
			return err
		}
	}

	worktree, err := repository.Worktree()
	if err != nil {
		return err
	}
	err = worktree.PullContext(ctx, &git.PullOptions{
		Force:         true,
		SingleBranch:  true,
		ReferenceName: branch,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}

	defer func() {
		worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.Master})
		repository.Storer.RemoveReference(plumbing.NewBranchReferenceName(BranchName))
	}()

	checkoutOpts := &git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(BranchName),
	}
	if _, err = repository.Branch(BranchName); errors.Is(err, git.ErrBranchNotFound) {
		checkoutOpts.Create = true
	}
	if err := worktree.Checkout(checkoutOpts); err != nil {
		return err
	}

	re := regexp.MustCompile(fmt.Sprintf(`%s:(?P<tag>\w[\w-\.]{0,127})`, image))
	if err = filepath.Walk(
		filepath.Join(clonepath, g.Path),
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if strings.HasPrefix(path, filepath.Join(clonepath, ".git")) {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			replacedContent := re.ReplaceAll(content, []byte(fmt.Sprintf("%s:%s", image, tag)))
			if err := ioutil.WriteFile(path, replacedContent, 0); err != nil {
				return err
			}
			if !bytes.Equal(content, replacedContent) {
				prefix := fmt.Sprintf("%s/", clonepath)
				if _, err := worktree.Add(strings.TrimPrefix(path, prefix)); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
		return err
	}

	status, err := worktree.Status()
	if err != nil {
		return err
	}

	// To prevent non-fast-forward error, do not commit and push
	// if no file was modified.
	if len(status) == 0 {
		return ErrTagNotReplaced
	}

	msg := "Update image tag names"
	if _, err := worktree.Commit(msg, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name: "manifest-updater",
			When: nowFunc(),
		},
	}); err != nil {
		return err
	}

	err = repository.PushContext(ctx, &git.PushOptions{Auth: auth})
	if err != nil {
		if errors.Is(err, git.ErrNonFastForwardUpdate) {
			return ErrTagAlreadyUpToDate
		}

		// Because of:
		// https://github.com/src-d/go-git/blob/d6c4b113c17a011530e93f179b7ac27eb3f17b9b/remote.go#L784
		// https://github.com/src-d/go-git/blob/d6c4b113c17a011530e93f179b7ac27eb3f17b9b/remote.go#L793
		if strings.Contains(err.Error(), git.ErrNonFastForwardUpdate.Error()) {
			return nil
		}
	}
	return err
}

func (g *GitHubRepository) CreatePullRequest(ctx context.Context) error {
	endpoint, err := transport.NewEndpoint(g.URL)
	if err != nil {
		return err
	}

	owner := g.extractOwnerFromEndpoint(endpoint)
	repoistory := g.extractRepositoryFromEndpoint(endpoint)

	client := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: g.Auth.Token},
	)))

	prs, _, err := client.PullRequests.List(ctx, owner, repoistory, &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:%s", owner, BranchName),
		Base: g.Branch,
	})
	if err != nil {
		return err
	}
	if len(prs) > 0 {
		return ErrPullRequestAlreadyExists
	}

	_, _, err = client.PullRequests.Create(ctx, owner, repoistory, &github.NewPullRequest{
		Title:               github.String("Automaticaly update image tags"),
		Head:                github.String(BranchName),
		Base:                github.String(g.Branch),
		Body:                github.String(""),
		MaintainerCanModify: github.Bool(true),
	})
	return err
}

func (g *GitHubRepository) extractOwnerFromEndpoint(endpoint *transport.Endpoint) string {
	path := strings.Split(strings.TrimPrefix(endpoint.Path, "/"), "/")
	return path[0]
}

func (g *GitHubRepository) extractRepositoryFromEndpoint(endpoint *transport.Endpoint) string {
	path := strings.Split(strings.TrimPrefix(endpoint.Path, "/"), "/")
	return strings.TrimSuffix(path[1], ".git")
}
