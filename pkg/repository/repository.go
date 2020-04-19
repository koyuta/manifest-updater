package repository

import (
	"context"
	"errors"
)

var (
	ErrTagAlreadyUpToDate       = errors.New("tag already up to date")
	ErrTagNotReplaced           = errors.New("tag not replaced")
	ErrPullRequestAlreadyExists = errors.New("pull request already exists")
)

type Repository interface {
	PushReplaceTagCommit(ctx context.Context, image, tag string) error
	CreatePullRequest(ctx context.Context) error
}
