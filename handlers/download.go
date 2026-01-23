package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"sono-version-service/database"
	"sono-version-service/models"
	"sono-version-service/storage"
)

type DownloadHandler struct {
	storage      storage.Storage
	versionStore *models.VersionStore
	db           *database.DB
}

func NewDownloadHandler(s storage.Storage, vs *models.VersionStore, db *database.DB) *DownloadHandler {
	return &DownloadHandler{
		storage:      s,
		versionStore: vs,
		db:           db,
	}
}

func (h *DownloadHandler) Handle(w http.ResponseWriter, r *http.Request) {
	channelStr := chi.URLParam(r, "channel")
	channel := models.Channel(channelStr)

	if !channel.IsValid() {
		http.Error(w, "Invalid channel. Must be: stable, beta, or nightly", http.StatusBadRequest)
		return
	}

	versionInfo := h.versionStore.Get(channel)
	if versionInfo == nil {
		http.Error(w, "No version available for this channel", http.StatusNotFound)
		return
	}

	reader, size, err := h.storage.Download(r.Context(), versionInfo.FileName)
	if err != nil {
		log.Printf("Failed to download APK: %v", err)
		http.Error(w, "Failed to retrieve APK", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	if h.db != nil {
		//capture values before goroutine to avoid race conditions
		//use background context since request context may be cancelled after response
		ch := string(channel)
		ver := versionInfo.Version
		ip := getClientIP(r)
		ua := r.UserAgent()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := h.db.LogDownload(ctx, ch, ver, ip, ua); err != nil {
				log.Printf("Failed to log download: %v", err)
			}
		}()
	}

	w.Header().Set("Content-Type", "application/vnd.android.package-archive")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"sono-%s-v%s.apk\"", channel, versionInfo.Version))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("X-Version", versionInfo.Version)
	w.Header().Set("X-SHA256", versionInfo.SHA256)

	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("Failed to stream APK: %v", err)
	}
}

func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}