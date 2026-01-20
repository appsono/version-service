package models

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Channel string

const (
	ChannelStable  Channel = "stable"
	ChannelBeta    Channel = "beta"
	ChannelNightly Channel = "nightly"
)

func (c Channel) IsValid() bool {
	return c == ChannelStable || c == ChannelBeta || c == ChannelNightly
}

type VersionInfo struct {
	Channel      Channel   `json:"channel"`
	Version      string    `json:"version"`
	VersionCode  int       `json:"version_code"`
	DownloadURL  string    `json:"download_url"`
	FileSize     int64     `json:"file_size"`
	SHA256       string    `json:"sha256"`
	ReleaseNotes string    `json:"release_notes"`
	PublishedAt  time.Time `json:"published_at"`
	FileName     string    `json:"file_name"`
}

type VersionStore struct {
	mu       sync.RWMutex
	filePath string
	Versions map[Channel]*VersionInfo `json:"versions"`
}

func NewVersionStore(filePath string) (*VersionStore, error) {
	store := &VersionStore{
		filePath: filePath,
		Versions: make(map[Channel]*VersionInfo),
	}

	if err := store.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return store, nil
}

func (s *VersionStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var fileData struct {
		Versions map[Channel]*VersionInfo `json:"versions"`
	}
	if err := json.Unmarshal(data, &fileData); err != nil {
		return err
	}

	s.Versions = fileData.Versions
	return nil
}

func (s *VersionStore) save() error {
	data, err := json.MarshalIndent(struct {
		Versions map[Channel]*VersionInfo `json:"versions"`
	}{
		Versions: s.Versions,
	}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *VersionStore) Get(channel Channel) *VersionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Versions[channel]
}

func (s *VersionStore) Set(info *VersionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Versions[info.Channel] = info
	return s.save()
}

type UploadRequest struct {
	Channel      Channel `json:"channel"`
	Version      string  `json:"version"`
	VersionCode  int     `json:"version_code"`
	ReleaseNotes string  `json:"release_notes"`
	ApkURL       string  `json:"apk_url"`
}

func (r *UploadRequest) Validate() bool {
	return r.Channel.IsValid() &&
		r.Version != "" &&
		r.VersionCode > 0 &&
		r.ApkURL != ""
}