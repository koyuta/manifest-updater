package registry

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type DockerHubRegistry struct {
	URL    string
	Filter string
}

func NewDockerHubRegistry(u, f string) *DockerHubRegistry {
	return &DockerHubRegistry{URL: u, Filter: f}
}

func (d *DockerHubRegistry) FetchLatestTag(ctx context.Context) (string, error) {
	registry, err := name.NewRepository(d.URL)
	if err != nil {
		return "", err
	}
	tags, err := remote.ListWithContext(ctx, registry)
	fmt.Printf("tags: %v\n", tags)
	fmt.Printf("err: %v\n", err)
	if len(tags) == 0 {
		return "", ErrNoTagsFound
	}

	return retrieveLatestTag(d.Filter, tags)
}

func retrieveLatestTag(filter string, tags []string) (string, error) {
	var tag = tags[0]
	if filter != "" {
		re, err := regexp.Compile(regexp.QuoteMeta(filter))
		if err != nil {
			return "", err
		}
		for _, t := range tags {
			if re.MatchString(t) {
				tag = t
				break
			}
		}
	}
	return tag, nil
}
