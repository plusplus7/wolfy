package components

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"wolfy/internal/fileutil"
)

type ParamStore struct {
	mu     sync.Mutex
	path   string
	params map[string]map[string]string
}

func NewParamStore(path string) (*ParamStore, error) {
	store := &ParamStore{
		path:   path,
		params: map[string]map[string]string{},
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ParamStore) Ensure(component string, keys []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.params[component]
	if !ok {
		current = map[string]string{}
		s.params[component] = current
	}
	for _, key := range keys {
		if _, ok := current[key]; !ok {
			current[key] = ""
		}
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return cloneMap(current), nil
}

func (s *ParamStore) Update(component string, params map[string]string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.params[component]
	if !ok {
		return nil, errors.New("unknown component: " + component)
	}
	for key, value := range params {
		current[key] = value
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return cloneMap(current), nil
}

func (s *ParamStore) load() error {
	file, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(file) == 0 {
		return nil
	}
	return json.Unmarshal(file, &s.params)
}

func (s *ParamStore) saveLocked() error {
	if dir := filepath.Dir(s.path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	result, err := json.MarshalIndent(s.params, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWriteFile(s.path, result, 0644)
}
