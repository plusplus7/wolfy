package songs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const CacheVersion = 1

type staticSong struct {
	Title    string            `json:"title"`
	Alias    []string          `json:"alias"`
	Image    string            `json:"image"`
	Category string            `json:"category"`
	Levels   []staticSongLevel `json:"levels"`
}

type staticSongLevel struct {
	Type       string `json:"type"`
	Difficulty string `json:"difficulty"`
	Level      string `json:"level"`
}

func NewMaimaiStorageFromCache(cachePath string) (*MaimaiStorage, error) {
	cache, err := loadMaimaiStorageCache(cachePath)
	if err != nil {
		return nil, err
	}
	return storageFromCache(cachePath, cache), nil
}

func dumpMaimaiStorageCache(storage *MaimaiStorage, cachePath string) error {
	if storage == nil {
		return fmt.Errorf("cannot dump nil maimai storage")
	}
	cache := MaimaiStorageCache{
		Version: CacheVersion,
		Records: storage.records,
		Aliases: storage.aliases,
	}
	return writeMaimaiStorageCache(cachePath, cache)
}

func loadMaimaiStorageCache(cachePath string) (*MaimaiStorageCache, error) {
	content, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	var cache MaimaiStorageCache
	if err := json.Unmarshal(content, &cache); err != nil {
		return nil, err
	}
	if cache.Version != CacheVersion {
		return nil, fmt.Errorf("unsupported maimai storage cache version %d", cache.Version)
	}
	if cache.Records == nil {
		cache.Records = map[int]*MaimaiRecord{}
	}
	if cache.Aliases == nil {
		cache.Aliases = map[int][]string{}
	}
	return &cache, nil
}

func writeMaimaiStorageCache(cachePath string, cache MaimaiStorageCache) error {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, content, 0644)
}

func storageFromCache(cachePath string, cache *MaimaiStorageCache) *MaimaiStorage {
	return &MaimaiStorage{
		filePath: cachePath,
		records:  cache.Records,
		aliases:  cache.Aliases,
	}
}

func ConvertStaticSongsJSONToCache(sourcePath, cachePath string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	var source map[string]staticSong
	if err := json.Unmarshal(content, &source); err != nil {
		return err
	}

	cache := MaimaiStorageCache{
		Version: CacheVersion,
		Records: map[int]*MaimaiRecord{},
		Aliases: map[int][]string{},
	}
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		id, err := strconv.Atoi(key)
		if err != nil {
			return fmt.Errorf("invalid static song id %q: %w", key, err)
		}
		song := source[key]
		levels := make([]MaimaiLevel, 0, len(song.Levels))
		for _, level := range song.Levels {
			levels = append(levels, MaimaiLevel{
				Type:       level.Type,
				Difficulty: level.Level,
				Level:      level.Difficulty,
			})
		}
		cache.Records[id] = &MaimaiRecord{
			ID:        id,
			Title:     song.Title,
			ImagePath: coverPath(id),
			Levels:    levels,
			Category:  song.Category,
		}
		cache.Aliases[id] = uniqueStrings(append([]string{song.Title}, song.Alias...))
	}

	return writeMaimaiStorageCache(cachePath, cache)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
