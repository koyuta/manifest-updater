package repository

import (
	"context"
	"errors"
)

var (
	ErrTagAlreadyUpToDate = errors.New("tag already up to date")
)

type Repository interface {
	PushReplaceTagCommit(context.Context, string) error
	CreatePullRequest(context.Context) error
}
