package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

func (h *SystemHandler) GetPublicIP(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org?format=json")
	if err != nil {
		respondWithError(w, http.StatusServiceUnavailable, "Unable to determine public IP")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respondWithError(w, http.StatusServiceUnavailable, "Unable to determine public IP")
		return
	}

	var payload struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || payload.IP == "" {
		respondWithError(w, http.StatusServiceUnavailable, "Unable to determine public IP")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"ip": payload.IP})
}
