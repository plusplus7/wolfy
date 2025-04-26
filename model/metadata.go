package model

import (
	"encoding/json"
	"os"
)

const (
	metadataPath = "./runtime/metadata.json"
)

type Metadata struct {
	AnchorCode string `json:"anchor_code"`
}

type MetadataManager struct {
	Metadata *Metadata
}

func NewMetadataManager() *MetadataManager {
	m := &MetadataManager{}
	err := m.LoadFromFile()
	if err != nil {
		panic(err)
	}

	return m
}

func (m *MetadataManager) LoadFromFile() error {
	file, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}
	var metadata Metadata
	err = json.Unmarshal(file, &metadata)
	if err != nil {
		return err
	}

	m.Metadata = &metadata
	return nil
}

func (m *MetadataManager) Get() *Metadata {
	return m.Metadata
}

func (m *MetadataManager) Save(metadata *Metadata) error {
	m.Metadata = metadata
	return nil
}
