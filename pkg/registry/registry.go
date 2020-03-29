package registry

import (
	"context"
	"errors"
)

var (
	ErrNoTagsFound = errors.New("No tags found")
)

type Registry interface {
	FetchLatestTag(context.Context) (string, error)
}
