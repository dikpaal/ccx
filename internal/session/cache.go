package session

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"time"
)

// cachedSession is the on-disk representation of a cached session.
type cachedSession struct {
	ModTime time.Time
	Sess    Session
}

// sessionCache maps file path → cached session metadata.
type sessionCache struct {
	path    string
	entries map[string]cachedSession
	dirty   bool
}

func cacheFilePath(claudeDir string) string {
	return filepath.Join(claudeDir, ".ccx-cache.gob")
}

func loadCache(claudeDir string) *sessionCache {
	sc := &sessionCache{
		path:    cacheFilePath(claudeDir),
		entries: make(map[string]cachedSession),
	}

	f, err := os.Open(sc.path)
	if err != nil {
		return sc
	}
	defer f.Close()

	var entries map[string]cachedSession
	if err := gob.NewDecoder(f).Decode(&entries); err != nil {
		return sc
	}
	sc.entries = entries
	return sc
}

func (sc *sessionCache) lookup(path string, modTime time.Time) (Session, bool) {
	cached, ok := sc.entries[path]
	if !ok || !cached.ModTime.Equal(modTime) {
		return Session{}, false
	}
	return cached.Sess, true
}

func (sc *sessionCache) store(path string, modTime time.Time, sess Session) {
	sc.entries[path] = cachedSession{ModTime: modTime, Sess: sess}
	sc.dirty = true
}

func (sc *sessionCache) save() {
	if !sc.dirty {
		return
	}
	f, err := os.Create(sc.path)
	if err != nil {
		return
	}
	defer f.Close()
	gob.NewEncoder(f).Encode(sc.entries)
}

// prune removes entries for files that no longer exist.
func (sc *sessionCache) prune(validPaths map[string]bool) {
	for p := range sc.entries {
		if !validPaths[p] {
			delete(sc.entries, p)
			sc.dirty = true
		}
	}
}
