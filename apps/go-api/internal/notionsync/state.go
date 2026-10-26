package notionsync

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type Entry struct {
	ID      string         `json:"id"`
	Base    map[string]any `json:"base"`
	Source  map[string]any `json:"source,omitempty"`
	Pending map[string]any `json:"pending,omitempty"`
}

type Journal struct {
	Version int               `json:"version"`
	Scope   string            `json:"scope"`
	Entries map[string]*Entry `json:"entries"`
	path    string
	lock    *os.File
}

// OpenJournal locks the state across preview and apply. The lock is released by
// the OS on a crash; the journal's pending write is then reconciled on restart.
func OpenJournal(path, scope string) (*Journal, error) {
	if path == "" {
		return nil, fmt.Errorf("a journal path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lock.Close()
		return nil, fmt.Errorf("another sync holds this journal: %w", err)
	}
	j := &Journal{Version: 1, Scope: scope, Entries: map[string]*Entry{}, path: path, lock: lock}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = j.Close()
		return nil, err
	}
	if err == nil {
		if err = json.Unmarshal(data, j); err != nil {
			_ = j.Close()
			return nil, err
		}
		if j.Version != 1 || j.Scope != scope || j.Entries == nil {
			_ = j.Close()
			return nil, fmt.Errorf("journal version or destination does not match this sync")
		}
	}
	return j, nil
}

func (j *Journal) Close() error { return j.lock.Close() }

func (j *Journal) Save() error {
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(j.path), ".notion-sync-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err = file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, j.path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(j.path))
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	return dir.Sync()
}
