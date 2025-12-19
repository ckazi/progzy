package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"proxy-server/database"
	"proxy-server/models"
	"proxy-server/utils"
)

const handlerTimeout = 5 * time.Second

type AuthHandler struct {
	db *database.Database
}

func NewAuthHandler(db *database.Database) *AuthHandler {
	return &AuthHandler{db: db}
}

func (h *AuthHandler) CheckInit(w http.ResponseWriter, r *http.Request) {
	initialized, err := h.db.IsInitialized()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]bool{"initialized": initialized})
}

func (h *AuthHandler) InitSetup(w http.ResponseWriter, r *http.Request) {
	initialized, err := h.db.IsInitialized()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if initialized {
		respondWithError(w, http.StatusBadRequest, "System already initialized")
		return
	}

	var req models.InitSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	userCreate := &models.UserCreate{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Comment:  "Initial admin user",
		IsAdmin:  true,
	}

	user, err := h.db.CreateUser(userCreate, hashedPassword)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.IsAdmin, true)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	respondWithJSON(w, http.StatusCreated, models.LoginResponse{
		Token:   token,
		User:    *user,
		Message: "Admin user created successfully",
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	user, err := h.db.GetUserByUsernameCtx(ctx, req.Username)
	if err != nil {
		h.logAuditEvent(nil, "LOGIN_FAIL", fmt.Sprintf("username=%s reason=user_not_found", strings.TrimSpace(req.Username)), r)
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !user.IsActive {
		h.logAuditEvent(&user.ID, "LOGIN_FAIL", fmt.Sprintf("username=%s reason=inactive", user.Username), r)
		respondWithError(w, http.StatusUnauthorized, "User account is inactive")
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		h.logAuditEvent(&user.ID, "LOGIN_FAIL", fmt.Sprintf("username=%s reason=invalid_password", user.Username), r)
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !user.IsAdmin {
		h.logAuditEvent(&user.ID, "LOGIN_FAIL", fmt.Sprintf("username=%s reason=not_admin", user.Username), r)
		respondWithError(w, http.StatusUnauthorized, "Admin access required")
		return
	}

	if user.TwoFAEnabled {
		tempToken, err := utils.GenerateTempToken(user.ID, user.Username, user.IsAdmin)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to generate 2FA token")
			return
		}
		respondWithJSON(w, http.StatusOK, models.LoginResponse{
			User:        *user,
			Requires2FA: true,
			TempToken:   tempToken,
			Message:     "Two-factor authentication required",
		})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.IsAdmin, true)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	response := models.LoginResponse{
		Token: token,
		User:  *user,
	}
	if user.IsAdmin {
		h.logAuditEvent(&user.ID, "LOGIN_SUCCESS", "Admin login (password)", r)
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) logAuditEvent(userID *int, action, details string, r *http.Request) {
	h.db.LogAdminAction(userID, action, details, getRequestIP(r))
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, models.ErrorResponse{Error: message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
