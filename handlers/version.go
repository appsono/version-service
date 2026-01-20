package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"sono-version-service/models"
)

type VersionHandler struct {
	versionStore *models.VersionStore
}

func NewVersionHandler(vs *models.VersionStore) *VersionHandler {
	return &VersionHandler{versionStore: vs}
}

func (h *VersionHandler) Handle(w http.ResponseWriter, r *http.Request) {
	channelStr := chi.URLParam(r, "channel")
	channel := models.Channel(channelStr)

	if !channel.IsValid() {
		http.Error(w, "Invalid channel. Must be: stable, beta, or nightly", http.StatusBadRequest)
		return
	}

	versionInfo := h.versionStore.Get(channel)
	if versionInfo == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "not_found",
			"message": "No version available for this channel",
			"channel": channel,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versionInfo)
}