package proxy

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"proxy-server/database"
	"proxy-server/models"
	"proxy-server/utils"
)

type ProxyServer struct {
	db     *database.Database
	server *http.Server
}

func NewProxyServer(db *database.Database, port string) *ProxyServer {
	ps := &ProxyServer{
		db: db,
	}

	ps.server = &http.Server{
		Addr:         ":" + port,
		Handler:      http.HandlerFunc(ps.handleProxy),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return ps
}

func (ps *ProxyServer) Start() error {
	log.Printf("Proxy server starting on port %s", ps.server.Addr)
	return ps.server.ListenAndServe()
}

func (ps *ProxyServer) authenticateRequest(r *http.Request) (*utils.Claims, error) {
	authHeader := r.Header.Get("Proxy-Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing proxy authorization")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid authorization format")
	}

	var username, password string

	switch parts[0] {
	case "Basic":
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid base64 encoding")
		}
		creds := strings.SplitN(string(decoded), ":", 2)
		if len(creds) != 2 {
			return nil, fmt.Errorf("invalid credentials format")
		}
		username, password = creds[0], creds[1]

	case "Bearer":
		claims, err := utils.ValidateToken(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid token")
		}
		if claims.IsAdmin {
			return nil, fmt.Errorf("admin accounts cannot use proxy")
		}
		return claims, nil

	default:
		return nil, fmt.Errorf("unsupported authorization method")
	}

	user, err := ps.db.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user is inactive")
	}

	if user.IsAdmin {
		return nil, fmt.Errorf("admin accounts cannot use proxy")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid password")
	}

	return &utils.Claims{
		UserID:   user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
	}, nil
}

func (ps *ProxyServer) handleProxy(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	claims, err := ps.authenticateRequest(r)
	if err != nil {
		w.Header().Set("Proxy-Authenticate", `Basic realm="Proxy Server"`)
		http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
		log.Printf("Authentication failed: %v", err)
		return
	}

	settings, err := ps.db.GetUserProxySettings(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to load proxy settings", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodConnect {
		ps.handleHTTPS(w, r, claims, settings, startTime)
	} else {
		ps.handleHTTP(w, r, claims, settings, startTime)
	}
}

func (ps *ProxyServer) handleHTTP(w http.ResponseWriter, r *http.Request, claims *utils.Claims, prefs *models.UserProxySettings, startTime time.Time) {
	outboundReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create upstream request", http.StatusBadGateway)
		ps.logRequest(claims.UserID, r.Method, r.URL.String(), http.StatusBadGateway, 0, 0, startTime)
		return
	}

	outboundReq.Header = make(http.Header)
	for k, vv := range r.Header {
		if strings.EqualFold(k, "Proxy-Authorization") || strings.EqualFold(k, "Proxy-Connection") {
			continue
		}
		for _, v := range vv {
			outboundReq.Header.Add(k, v)
		}
	}

	if outboundReq.URL.Scheme == "" {
		outboundReq.URL.Scheme = "http"
	}
	if outboundReq.Host == "" {
		outboundReq.Host = outboundReq.URL.Host
	}

	targetHost := extractHost(outboundReq.URL.Host)
	if !isHostAllowed(prefs, targetHost) {
		http.Error(w, "Access to this host is not permitted", http.StatusForbidden)
		ps.logRequest(claims.UserID, r.Method, r.URL.String(), http.StatusForbidden, 0, 0, startTime)
		return
	}

	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior := outboundReq.Header.Get("X-Forwarded-For"); prior != "" {
			outboundReq.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			outboundReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(outboundReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		ps.logRequest(claims.UserID, r.Method, r.URL.String(), http.StatusBadGateway, 0, 0, startTime)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	bytesSent, _ := io.Copy(w, resp.Body)

	var requestSize int64
	if r.ContentLength > 0 {
		requestSize = r.ContentLength
	}

	ps.logRequest(claims.UserID, r.Method, r.URL.String(), resp.StatusCode, requestSize, bytesSent, startTime)
	ps.db.UpdateTrafficStats(claims.UserID, requestSize, bytesSent)
}

func (ps *ProxyServer) handleHTTPS(w http.ResponseWriter, r *http.Request, claims *utils.Claims, prefs *models.UserProxySettings, startTime time.Time) {
	targetHost := extractHost(r.Host)
	if !isHostAllowed(prefs, targetHost) {
		http.Error(w, "Access to this host is not permitted", http.StatusForbidden)
		ps.logRequest(claims.UserID, r.Method, r.Host, http.StatusForbidden, 0, 0, startTime)
		return
	}

	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		ps.logRequest(claims.UserID, r.Method, r.Host, http.StatusServiceUnavailable, 0, 0, startTime)
		return
	}
	defer destConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	var bytesSent, bytesReceived int64

	errChan := make(chan error, 2)

	go func() {
		n, err := io.Copy(destConn, clientConn)
		bytesSent = n
		errChan <- err
	}()

	go func() {
		n, err := io.Copy(clientConn, destConn)
		bytesReceived = n
		errChan <- err
	}()

	<-errChan

	ps.logRequest(claims.UserID, r.Method, r.Host, http.StatusOK, bytesSent, bytesReceived, startTime)
	ps.db.UpdateTrafficStats(claims.UserID, bytesSent, bytesReceived)
}

func (ps *ProxyServer) logRequest(userID int, method, url string, statusCode int, bytesSent, bytesReceived int64, startTime time.Time) {
	duration := time.Since(startTime).Milliseconds()

	requestLog := &models.RequestLog{
		UserID:        &userID,
		Method:        method,
		URL:           url,
		StatusCode:    statusCode,
		BytesSent:     bytesSent,
		BytesReceived: bytesReceived,
		DurationMs:    int(duration),
	}

	if err := ps.db.LogRequest(requestLog); err != nil {
		log.Printf("Failed to log request: %v", err)
	}
}

func init() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: false,
	}
}

func extractHost(raw string) string {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			return parsedHost
		}
		if idx := strings.Index(host, ":"); idx > -1 {
			return host[:idx]
		}
	}
	return host
}

func isHostAllowed(prefs *models.UserProxySettings, host string) bool {
	if prefs == nil {
		return true
	}
	if host == "" {
		return true
	}
	switch prefs.ProxyType {
	case "whitelist":
		return matchesList(prefs.Whitelist, host)
	case "blacklist":
		return !matchesList(prefs.Blacklist, host)
	default:
		return true
	}
}

func matchesList(list []string, host string) bool {
	if len(list) == 0 {
		return false
	}
	lowerHost := strings.ToLower(host)
	for _, entry := range list {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		if strings.Contains(lowerHost, entry) {
			return true
		}
	}
	return false
}
