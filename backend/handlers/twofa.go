package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"proxy-server/middleware"
	"proxy-server/models"
	"proxy-server/utils"
)

const (
	twoFARateLimitAttempts = 5
	twoFARateLimitWindow   = 5 * time.Minute
	backupCodesCount       = 10
)

func (h *AuthHandler) SetupTwoFA(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	key, secret, err := utils.GenerateKey(user.Username, user.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate 2FA secret")
		return
	}

	encryptedSecret, err := utils.EncryptSecret(secret)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to secure 2FA secret")
		return
	}

	if err := h.db.StoreTwoFASecret(ctx, user.ID, encryptedSecret); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to persist 2FA secret")
		return
	}

	qr, err := utils.GenerateQRCodeBase64(key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to render QR code")
		return
	}

	resp := models.TwoFASetupResponse{
		Secret:     secret,
		OTPAuthURL: key.URL(),
		QRCode:     qr,
	}
	h.logTwoFAAttempt(r.Context(), user.ID, getRequestIP(r), "setup_init", "totp", true, "2FA setup initiated")
	respondWithJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) VerifySetupTwoFA(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.TwoFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		respondWithError(w, http.StatusBadRequest, "Verification code is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	ip := getRequestIP(r)
	if h.isRateLimited(ctx, user.ID, ip) {
		h.logTwoFAAttempt(ctx, user.ID, ip, "setup_verify", "totp", false, "Rate limit exceeded")
		respondWithError(w, http.StatusTooManyRequests, "Too many attempts. Try again later.")
		return
	}

	secret, err := h.loadTwoFASecret(ctx, user.ID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "2FA is not initialized for this account")
		return
	}

	if !utils.ValidateTOTP(secret, req.Code, time.Now()) {
		h.logTwoFAAttempt(ctx, user.ID, ip, "setup_verify", "totp", false, "Invalid TOTP code")
		respondWithError(w, http.StatusBadRequest, "Invalid verification code")
		return
	}

	codes, err := utils.GenerateBackupCodes(backupCodesCount)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate backup codes")
		return
	}

	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, err := utils.HashPassword(code)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to secure backup codes")
			return
		}
		hashes = append(hashes, hash)
	}

	if err := h.db.ReplaceBackupCodes(ctx, user.ID, hashes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to persist backup codes")
		return
	}

	if err := h.db.SetTwoFAEnabled(ctx, user.ID, true); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to enable 2FA")
		return
	}

	h.logTwoFAAttempt(ctx, user.ID, ip, "setup_verify", "totp", true, "2FA activated")
	respondWithJSON(w, http.StatusOK, models.TwoFAVerifyResponse{
		Message:     "Two-factor authentication activated",
		BackupCodes: codes,
	})
}

func (h *AuthHandler) VerifyTwoFA(w http.ResponseWriter, r *http.Request) {
	var req models.TwoFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	token := extractToken(r, req.TempToken)
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "Missing verification token")
		return
	}

	claims, err := utils.ValidateToken(token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid verification token")
		return
	}
	if claims.TwoFactorVerified {
		respondWithError(w, http.StatusBadRequest, "Token is already verified")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil || !user.IsActive {
		respondWithError(w, http.StatusUnauthorized, "User not found or inactive")
		return
	}

	ip := getRequestIP(r)
	if h.isRateLimited(ctx, user.ID, ip) {
		h.logTwoFAAttempt(ctx, user.ID, ip, "login", "totp", false, "Rate limit exceeded")
		respondWithError(w, http.StatusTooManyRequests, "Too many attempts. Try again later.")
		return
	}

	method, err := h.validateTwoFactorCode(ctx, user.ID, req.Code, req.BackupCode)
	if err != nil {
		h.logTwoFAAttempt(ctx, user.ID, ip, "login", method, false, err.Error())
		respondWithError(w, http.StatusUnauthorized, "Invalid authentication code")
		return
	}

	tokenResp, err := utils.GenerateToken(user.ID, user.Username, user.IsAdmin, true)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	if user.IsAdmin {
		h.db.LogAdminAction(&user.ID, "LOGIN_SUCCESS", fmt.Sprintf("Admin login via 2FA (%s)", method), getRequestIP(r))
	}

	h.logTwoFAAttempt(ctx, user.ID, ip, "login", method, true, "2FA challenge passed")
	respondWithJSON(w, http.StatusOK, models.TwoFAVerifyResponse{
		Token:   tokenResp,
		User:    user,
		Message: "Two-factor verification successful",
	})
}

func (h *AuthHandler) DisableTwoFA(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.TwoFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Code) == "" && strings.TrimSpace(req.BackupCode) == "" {
		respondWithError(w, http.StatusBadRequest, "Provide either verification or backup code")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	ip := getRequestIP(r)
	if h.isRateLimited(ctx, user.ID, ip) {
		h.logTwoFAAttempt(ctx, user.ID, ip, "disable", "totp", false, "Rate limit exceeded")
		respondWithError(w, http.StatusTooManyRequests, "Too many attempts. Try again later.")
		return
	}

	method, err := h.validateTwoFactorCode(ctx, user.ID, req.Code, req.BackupCode)
	if err != nil {
		h.logTwoFAAttempt(ctx, user.ID, ip, "disable", method, false, err.Error())
		respondWithError(w, http.StatusUnauthorized, "Invalid authentication code")
		return
	}

	if err := h.db.ClearTwoFAData(ctx, user.ID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to disable 2FA")
		return
	}

	user.TwoFAEnabled = false
	h.logTwoFAAttempt(ctx, user.ID, ip, "disable", method, true, "2FA disabled")
	respondWithJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Two-factor authentication disabled",
	})
}

func (h *AuthHandler) RegenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	backup := strings.TrimSpace(r.URL.Query().Get("backup_code"))
	if code == "" && backup == "" {
		respondWithError(w, http.StatusBadRequest, "Verification or backup code is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	ip := getRequestIP(r)
	if h.isRateLimited(ctx, user.ID, ip) {
		h.logTwoFAAttempt(ctx, user.ID, ip, "backup_regen", "totp", false, "Rate limit exceeded")
		respondWithError(w, http.StatusTooManyRequests, "Too many attempts. Try again later.")
		return
	}

	method, err := h.validateTwoFactorCode(ctx, user.ID, code, backup)
	if err != nil {
		h.logTwoFAAttempt(ctx, user.ID, ip, "backup_regen", method, false, err.Error())
		respondWithError(w, http.StatusUnauthorized, "Invalid authentication code")
		return
	}

	codes, err := utils.GenerateBackupCodes(backupCodesCount)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate backup codes")
		return
	}

	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, err := utils.HashPassword(code)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to secure backup codes")
			return
		}
		hashes = append(hashes, hash)
	}

	if err := h.db.ReplaceBackupCodes(ctx, user.ID, hashes); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to persist backup codes")
		return
	}

	h.logTwoFAAttempt(ctx, user.ID, ip, "backup_regen", method, true, "Backup codes regenerated")
	respondWithJSON(w, http.StatusOK, models.TwoFABackupCodesResponse{
		Codes: codes,
	})
}

func (h *AuthHandler) validateTwoFactorCode(ctx context.Context, userID int, code, backupCode string) (string, error) {
	secret, err := h.loadTwoFASecret(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("2FA secret unavailable")
	}

	now := time.Now()
	code = strings.TrimSpace(code)
	if code != "" && utils.ValidateTOTP(secret, code, now) {
		return "totp", nil
	}

	backupCode = strings.TrimSpace(backupCode)
	if backupCode != "" {
		ok, err := h.db.ConsumeBackupCode(ctx, userID, backupCode)
		if err != nil {
			return "", err
		}
		if ok {
			return "backup", nil
		}
	}

	return "", fmt.Errorf("invalid verification code")
}

func (h *AuthHandler) isRateLimited(ctx context.Context, userID int, ip string) bool {
	count, err := h.db.CountRecentTwoFAAttempts(ctx, userID, ip, twoFARateLimitWindow)
	if err != nil {
		log.Printf("Failed to evaluate 2FA rate limit: %v", err)
		return false
	}
	return count >= twoFARateLimitAttempts
}

func (h *AuthHandler) loadTwoFASecret(ctx context.Context, userID int) (string, error) {
	secretEnc, _, err := h.db.GetTwoFASecret(ctx, userID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(secretEnc) == "" {
		return "", fmt.Errorf("2FA secret missing")
	}
	return utils.DecryptSecret(secretEnc)
}

func (h *AuthHandler) logTwoFAAttempt(ctx context.Context, userID int, ip, event, method string, success bool, message string) {
	entry := &models.TwoFALogEntry{
		UserID:  userID,
		IP:      ip,
		Event:   event,
		Method:  method,
		Success: success,
		Message: message,
	}
	if err := h.db.LogTwoFAAttempt(ctx, entry); err != nil {
		log.Printf("Failed to log 2FA attempt: %v", err)
	}
}

func extractToken(r *http.Request, fallback string) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	headerToken := r.Header.Get("X-2FA-Token")
	if headerToken != "" {
		return headerToken
	}
	return strings.TrimSpace(fallback)
}
