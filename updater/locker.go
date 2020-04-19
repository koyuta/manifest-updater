package updater

import "sync"

type repoLocker struct {
	m sync.Map
}

func (r *repoLocker) Load(reponame string) *sync.Mutex {
	mux, ok := r.m.Load(reponame)
	if !ok {
		return nil
	}
	return mux.(*sync.Mutex)
}

func (r *repoLocker) Store(reponame string, mux *sync.Mutex) {
	r.m.Store(reponame, mux)
}

func (r *repoLocker) Delete(reponame string) {
	r.m.Delete(reponame)
}
