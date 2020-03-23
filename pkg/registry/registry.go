package registry

import "context"

type Registry interface {
	FetchLatestTag(context.Context) (string, error)
}
