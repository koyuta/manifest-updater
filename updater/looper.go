package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"manifest-updater/pkg/registry"
	"manifest-updater/pkg/repository"

	"github.com/go-logr/logr"
	"golang.org/x/sync/semaphore"
)

var timeout = 20 * time.Second

type UpdateLooper struct {
	entries       []*Entry
	checkInterval time.Duration
	logger        logr.Logger

	user  string
	token string

	queue <-chan *Entry
}

func NewUpdateLooper(queue <-chan *Entry, c time.Duration, logger logr.Logger, user, token string) *UpdateLooper {
	return &UpdateLooper{
		queue:         queue,
		checkInterval: c,
		logger:        logger,
		user:          user,
		token:         token,
	}
}

func (u *UpdateLooper) addEntry(entry *Entry) {
	for _, entry := range u.entries {
		if entry.ID == entry.ID {
			return
		}
	}
	u.entries = append(u.entries, entry)
}

func (u *UpdateLooper) deleteEntry(uid string) {
	for i, entry := range u.entries {
		if entry.ID == uid {
			u.entries = append(u.entries[:i], u.entries[i+1:]...)
		}
	}
}

func (u *UpdateLooper) Loop(stop <-chan struct{}) error {
	ticker := time.NewTicker(u.checkInterval)
	defer ticker.Stop()

	var (
		wg      = sync.WaitGroup{}
		sem     = semaphore.NewWeighted(10)
		rlocker = repoLocker{m: sync.Map{}}
	)

	for {
		select {
		case entry, ok := <-u.queue:
			if !ok {
				return errors.New("Queue was closed")
			}
			j, _ := json.Marshal(entry)

			if entry.Deleted {
				u.deleteEntry(entry.ID)
				u.logger.Info(fmt.Sprintf("Deleted a entry: %v", string(j)))
			} else {
				u.addEntry(entry)
				u.logger.Info(fmt.Sprintf("Added a entry: %v", string(j)))
			}
		case <-stop:
			wg.Wait()
			return nil
		case <-ticker.C:
			for i := range u.entries {
				entry := u.entries[i]
				updater := NewUpdater(entry, u.user, u.token)

				rlocker.Store(entry.Git, &sync.Mutex{})

				var errch = make(chan error, 1)

				mux := rlocker.Load(entry.Git)
				sem.Acquire(context.Background(), 1)
				wg.Add(1)
				go func() {
					defer func() {
						mux.Unlock()
						sem.Release(1)
						wg.Done()
					}()

					mux.Lock()

					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					errch <- updater.Run(ctx)

					select {
					case <-ctx.Done():
						u.logger.Error(ctx.Err(), "Updater")
					case err := <-errch:
						j, _ := json.Marshal(entry)
						switch {
						case errors.Is(err, repository.ErrTagAlreadyUpToDate):
							u.logger.Info(fmt.Sprintf("Image tag already up to date: %s", string(j)))
						case errors.Is(err, repository.ErrPullRequestAlreadyExists):
							u.logger.Info(fmt.Sprintf("Pull request already exists: %s", string(j)))
						case errors.Is(err, repository.ErrTagNotReplaced):
							u.logger.Info(fmt.Sprintf("Image tag was not replaced: %s", string(j)))
						case errors.Is(err, registry.ErrNoTagsFound):
							u.logger.Info(fmt.Sprintf("Image tag was not found: %s", string(j)))
						case err != nil:
							u.logger.Error(err, "Updater")
						default:
							u.logger.Info(fmt.Sprintf("Pull request was created: %s", string(j)))
						}
					}
				}()
			}
		}
	}
}
