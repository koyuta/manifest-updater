package updater

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/koyuta/manifest-updater/pkg/repository"
	"golang.org/x/sync/semaphore"
)

var timeout = 20 * time.Second

type UpdateLooper struct {
	entries       []*Entry
	checkInterval time.Duration
	logger        logr.Logger

	token string

	queue <-chan *Entry

	done         chan struct{}
	shuttingDown *atomic.Value
}

func NewUpdateLooper(queue <-chan *Entry, c time.Duration, logger logr.Logger, token string) *UpdateLooper {
	return &UpdateLooper{
		queue:         queue,
		checkInterval: c,
		logger:        logger,
		token:         token,
		done:          make(chan struct{}),
		shuttingDown:  &atomic.Value{},
	}
}

func (u *UpdateLooper) Loop(stop <-chan struct{}) error {
	if v := u.shuttingDown.Load(); v != nil {
		return errors.New("Looper is shutting down")
	}

	ticker := time.NewTicker(u.checkInterval)
	defer ticker.Stop()

	var (
		wg         = sync.WaitGroup{}
		sem        = semaphore.NewWeighted(10)
		repoLocker = map[string]sync.Locker{}
	)

	for {
		select {
		case entry, ok := <-u.queue:
			if !ok {
				return errors.New("Queue was closed")
			}
			j, _ := json.Marshal(entry)
			u.logger.Info(string(j), "Recieved a entry")
			u.entries = append(u.entries, entry)
			repoLocker[entry.Git] = &sync.Mutex{}
		case <-stop:
			wg.Wait()
			return nil
		case <-ticker.C:
			for i := range u.entries {
				entry := u.entries[i]
				updater := NewUpdater(entry, u.token)

				var errch = make(chan error, 1)
				mux := repoLocker[entry.Git]
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
							u.logger.Info(string(j), "Image tag already up to date")
						case err != nil:
							u.logger.Error(err, "Updater")
						default:
							u.logger.Info(string(j), "Pull request was created")
						}
					}
				}()
			}
		}
	}
}
