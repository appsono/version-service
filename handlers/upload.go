package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"sono-version-service/database"
	"sono-version-service/models"
	"sono-version-service/storage"
)

type UploadHandler struct {
	storage      storage.Storage
	versionStore *models.VersionStore
	db           *database.DB
	baseURL      string
}

func NewUploadHandler(s storage.Storage, vs *models.VersionStore, db *database.DB, baseURL string) *UploadHandler {
	return &UploadHandler{
		storage:      s,
		versionStore: vs,
		db:           db,
		baseURL:      baseURL,
	}
}

type EnhancedUploadRequest struct {
	Channel      models.Channel `json:"channel"`
	Version      string         `json:"version"`
	VersionCode  int            `json:"version_code"`
	ReleaseNotes string         `json:"release_notes"`
	ApkURL       string         `json:"apk_url"`
	ApkBase64    string         `json:"apk_base64"`
	GitHubToken  string         `json:"github_token"`
}

func (r *EnhancedUploadRequest) Validate() bool {
	hasURL := r.ApkURL != ""
	hasBase64 := r.ApkBase64 != ""
	
	return r.Channel.IsValid() &&
		r.Version != "" &&
		r.VersionCode > 0 &&
		(hasURL || hasBase64)
}

func (h *UploadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req EnhancedUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !req.Validate() {
		http.Error(w, "Invalid request: missing required fields or APK source", http.StatusBadRequest)
		return
	}

	var apkData []byte
	var err error

	//try to get apk data from base64
	if req.ApkBase64 != "" {
		log.Printf("Processing base64 encoded APK for %s v%s", req.Channel, req.Version)
		apkData, err = base64.StdEncoding.DecodeString(req.ApkBase64)
		if err != nil {
			log.Printf("Failed to decode base64 APK: %v", err)
			h.logUpload(r, string(req.Channel), req.Version, "failed", "Failed to decode base64 APK", "base64")
			http.Error(w, "Failed to decode base64 APK data", http.StatusBadRequest)
			return
		}
		log.Printf("Successfully decoded APK (%d bytes)", len(apkData))
	} else if req.ApkURL != "" {
		//fall back to URL
		log.Printf("Downloading APK from: %s", req.ApkURL)
		apkData, err = h.downloadAPK(req.ApkURL, req.GitHubToken)
		if err != nil {
			log.Printf("Failed to download APK: %v", err)
			h.logUpload(r, string(req.Channel), req.Version, "failed", fmt.Sprintf("Failed to download APK: %v", err), req.ApkURL)
			http.Error(w, fmt.Sprintf("Failed to download APK from URL: %v", err), http.StatusBadGateway)
			return
		}
	}

	//compute hash
	hash := sha256.Sum256(apkData)
	sha256Hash := hex.EncodeToString(hash[:])

	fileName := fmt.Sprintf("%s/sono-%s-v%s.apk", req.Channel, req.Channel, req.Version)

	//upload to storage
	log.Printf("Uploading APK: %s (%d bytes)", fileName, len(apkData))
	if err := h.storage.Upload(r.Context(), fileName, bytes.NewReader(apkData), int64(len(apkData))); err != nil {
		log.Printf("Failed to store APK: %v", err)
		h.logUpload(r, string(req.Channel), req.Version, "failed", "Failed to store APK", req.ApkURL)
		http.Error(w, "Failed to store APK", http.StatusInternalServerError)
		return
	}

	//create version info
	versionInfo := &models.VersionInfo{
		Channel:      req.Channel,
		Version:      req.Version,
		VersionCode:  req.VersionCode,
		DownloadURL:  fmt.Sprintf("%s/api/v1/download/%s", h.baseURL, req.Channel),
		FileSize:     int64(len(apkData)),
		SHA256:       sha256Hash,
		ReleaseNotes: req.ReleaseNotes,
		PublishedAt:  time.Now().UTC(),
		FileName:     fileName,
	}

	//save to version store
	if err := h.versionStore.Set(versionInfo); err != nil {
		log.Printf("Failed to save version info: %v", err)
		h.logUpload(r, string(req.Channel), req.Version, "failed", "Failed to save metadata", req.ApkURL)
		http.Error(w, "Failed to save version metadata", http.StatusInternalServerError)
		return
	}

	//save to db
	if h.db != nil {
		h.db.InsertRelease(r.Context(), &database.Release{
			Channel:      string(req.Channel),
			Version:      req.Version,
			VersionCode:  req.VersionCode,
			FileName:     fileName,
			FileSize:     int64(len(apkData)),
			SHA256:       sha256Hash,
			ReleaseNotes: req.ReleaseNotes,
			PublishedAt:  time.Now().UTC(),
		})
	}

	source := req.ApkURL
	if source == "" {
		source = "base64"
	}
	h.logUpload(r, string(req.Channel), req.Version, "success", "Upload completed", source)
	log.Printf("Successfully uploaded %s v%s", req.Channel, req.Version)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully uploaded %s v%s", req.Channel, req.Version),
		"version": versionInfo,
	})
}

func (h *UploadHandler) downloadAPK(url, token string) ([]byte, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	//add token if provided
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (h *UploadHandler) logUpload(r *http.Request, channel, version, status, message, sourceURL string) {
	if h.db != nil {
		h.db.LogUpload(r.Context(), channel, version, status, message, sourceURL)
	}
}