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
	PushReplaceTagCommit(context.Context, string) error
	CreatePullRequest(context.Context) error
}
