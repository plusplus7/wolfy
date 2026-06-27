package songs

import (
	"context"
	"errors"
	"os"
	"sync"
	"wolfy/components"
)

const (
	SongsComponentName = "songs"
	ParamSongPackage   = "song_package_path"
	ParamAliasFile     = "alias_file_path"
	ParamCachePath     = "cache_path"

	DefaultCachePath = "./runtime/maimai.songs.cache.json"
)

type SongsComponent struct {
	*components.BaseComponent
	mu      sync.RWMutex
	storage *MaimaiStorage
}

func NewSongsComponent() *SongsComponent {
	return &SongsComponent{
		BaseComponent: components.NewBaseComponent(SongsComponentName, []string{
			ParamSongPackage,
			ParamAliasFile,
			ParamCachePath,
		}),
	}
}

func (s *SongsComponent) Start(ctx context.Context) error {
	params := s.Params()
	cachePath := params[ParamCachePath]
	songPackage := params[ParamSongPackage]
	aliasFile := params[ParamAliasFile]
	if cachePath != "" {
		storage, err := NewMaimaiStorageFromCache(cachePath)
		if err != nil {
			s.setStorage(nil)
			s.RecordEvent(components.ComponentEventSongsStorageLoadFailed, "cache_path="+cachePath+" error="+err.Error())
			s.SetStatus(components.StatusError, err)
			return err
		}
		s.setStorage(storage)
		s.RecordEvent(components.ComponentEventSongsStorageLoaded, "cache_path="+cachePath)
		s.SetStatus(components.StatusRunning, nil)
		return nil
	}
	if songPackage == "" {
		err := errors.New("cache_path or song_package_path is empty")
		s.setStorage(nil)
		s.RecordEvent(components.ComponentEventSongsStorageLoadFailed, err.Error())
		s.SetStatus(components.StatusWaiting, err)
		return nil
	}
	if _, err := os.Stat(songPackage); err != nil {
		s.setStorage(nil)
		s.RecordEvent(components.ComponentEventSongsStorageLoadFailed, "song_package_path="+songPackage+" error="+err.Error())
		s.SetStatus(components.StatusError, err)
		return err
	}
	if aliasFile != "" {
		if _, err := os.Stat(aliasFile); err != nil {
			s.setStorage(nil)
			s.RecordEvent(components.ComponentEventSongsStorageLoadFailed, "alias_file_path="+aliasFile+" error="+err.Error())
			s.SetStatus(components.StatusError, err)
			return err
		}
	}

	storage, err := NewMaimaiStorageFromPackageWithCache(songPackage, aliasFile, DefaultCachePath)
	if err != nil {
		s.setStorage(nil)
		s.RecordEvent(components.ComponentEventSongsStorageLoadFailed, "song_package_path="+songPackage+" error="+err.Error())
		s.SetStatus(components.StatusError, err)
		return err
	}
	s.setStorage(storage)
	s.RecordEvent(components.ComponentEventSongsStorageLoaded, "song_package_path="+songPackage+" cache_path="+DefaultCachePath)
	s.SetStatus(components.StatusRunning, nil)
	return nil
}

func (s *SongsComponent) Stop(ctx context.Context) error {
	s.setStorage(nil)
	s.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (s *SongsComponent) Restart(ctx context.Context) error {
	s.SetStatus(components.StatusRestarting, nil)
	return s.Start(ctx)
}

func (s *SongsComponent) Storage() *MaimaiStorage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.storage
}

func (s *SongsComponent) setStorage(storage *MaimaiStorage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storage = storage
}
