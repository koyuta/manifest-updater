package updater

import (
	"context"

	"github.com/koyuta/manifest-updater/pkg/registry"
	"github.com/koyuta/manifest-updater/pkg/repository"
)

type Entry struct {
	DockerHub string `json:"dockerHub"`
	Filter    string `json:"filter,omitempty"`
	Git       string `json:"git"`
	Branch    string `json:"branch,omitempty"`
	Path      string `json:"path,omitempty"`
}

type Updater struct {
	Registry   registry.Registry
	Repository repository.Repository
}

func NewUpdater(entry *Entry, token string) *Updater {
	return &Updater{
		Registry: registry.NewDockerHubRegistry(
			entry.DockerHub,
			entry.Filter,
		),
		Repository: repository.NewGitHubRepository(
			entry.Git,
			entry.Branch,
			entry.Path,
			entry.DockerHub,
			token,
		),
	}
}

func (u *Updater) Run(ctx context.Context) error {
	tag, err := u.Registry.FetchLatestTag(ctx)
	if err != nil {
		return err
	}
	if err := u.Repository.PushReplaceTagCommit(ctx, tag); err != nil {
		return err
	}
	return u.Repository.CreatePullRequest(ctx)
}
