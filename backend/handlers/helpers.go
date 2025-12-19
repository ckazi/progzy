package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

func getRequestIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func formatAuditJSON(payload interface{}) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(data)
}
