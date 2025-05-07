package model

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

const (
	metadataPath = "./runtime/metadata.json"
)

type SystemEvent int64

const (
	SystemReboot SystemEvent = iota
	SystemExit
)

type ServiceInfo struct {
	info string
	err  error
}

type SystemInfo struct {
	AnchorCode  string `json:"anchor_code"`
	AppID       int64  `json:"app_id"`
	SystemError string `json:"system_error"`
	Game        string `json:"game"`

	BilibiliGameID       string `json:"bilibili_game_id"`
	BilibiliAccessKey    string `json:"bilibili_access_key"`
	BilibiliAccessSecret string `json:"bilibili_access_secret"`
	RemoteSignatoryAddr  string `json:"remote_signatory_addr"`

	ServiceInfo map[string]ServiceInfo `json:"service"`
}

type SystemInfoManager struct {
	metadata SystemInfo
	lock     *sync.Mutex
}

func NewSystemInfoManager() *SystemInfoManager {
	m := &SystemInfoManager{lock: new(sync.Mutex)}
	err := m.LoadFromFile()
	if err != nil {
		m.metadata = SystemInfo{
			Game:                "maimai",
			AnchorCode:          "",
			AppID:               1748453177364,
			RemoteSignatoryAddr: "https://plusplus7.com:42376",
			ServiceInfo:         make(map[string]ServiceInfo),
		}
		err = m.Save(m.metadata)
		if err != nil {
			panic(err)
		}
	}

	return m
}

func (m *SystemInfoManager) LoadFromFile() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	file, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}
	var metadata SystemInfo
	err = json.Unmarshal(file, &metadata)
	if err != nil {
		return err
	}

	m.metadata = metadata
	return nil
}

func (m *SystemInfoManager) Get() SystemInfo {
	return m.metadata
}

func (m *SystemInfoManager) SetServiceInfo(serviceName, status string, sysErr error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	get := m.Get()
	get.ServiceInfo[serviceName] = ServiceInfo{status, sysErr}
	err := m.save(get)
	if err != nil {
		log.Fatalf("Failed to save system info: %v", err)
		return
	}

}

func (m *SystemInfoManager) save(metadata SystemInfo) error {
	m.metadata = metadata

	result, err := json.Marshal(m.metadata)
	if err != nil {
		return err
	}
	err = os.WriteFile(metadataPath, result, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (m *SystemInfoManager) Save(metadata SystemInfo) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.save(metadata)
}
