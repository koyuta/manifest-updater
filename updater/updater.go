package updater

import (
	"context"

	"manifest-updater/pkg/registry"
	"manifest-updater/pkg/repository"
)

type Updater struct {
	RepositoryName string                `json:"-"`
	ImageName      string                `json:"-"`
	Registry       registry.Registry     `json:"registry"`
	Repository     repository.Repository `json:"repository"`
}

func NewUpdater(entry *Entry, user, token string) *Updater {
	return &Updater{
		Registry: registry.NewDockerHubRegistry(
			entry.DockerHub,
			entry.Filter,
		),
		Repository: repository.NewGitHubRepository(
			entry.Git,
			entry.Branch,
			entry.Path,
			repository.GithubAuth{
				User:  user,
				Token: token,
			},
		),
	}
}

func (u *Updater) Run(ctx context.Context) error {
	tag, err := u.Registry.FetchLatestTag(ctx)
	if err != nil {
		return err
	}
	if err := u.Repository.PushReplaceTagCommit(ctx, u.ImageName, tag); err != nil {
		return err
	}
	return u.Repository.CreatePullRequest(ctx)
}
