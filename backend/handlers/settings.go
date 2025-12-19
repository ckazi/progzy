package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"proxy-server/database"
	"proxy-server/middleware"
)

type SettingsHandler struct {
	db *database.Database
}

func NewSettingsHandler(db *database.Database) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.db.GetProxySettings()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch settings")
		return
	}

	respondWithJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Key == "" {
		respondWithError(w, http.StatusBadRequest, "Key is required")
		return
	}

	prev, err := h.db.GetProxySetting(req.Key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch current setting")
		return
	}

	if err := h.db.UpdateProxySetting(req.Key, req.Value); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update setting")
		return
	}

	settings, err := h.db.GetProxySettings()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch updated settings")
		return
	}

	if actor := middleware.GetUserFromContext(r); actor != nil {
		oldValue := ""
		if prev != nil {
			oldValue = prev.Value
		}
		details := fmt.Sprintf("Setting %s changed: %q -> %q", req.Key, oldValue, req.Value)
		h.db.LogAdminAction(&actor.ID, "SETTINGS_UPDATE", details, getRequestIP(r))
	}

	respondWithJSON(w, http.StatusOK, settings)
}
