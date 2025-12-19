package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"proxy-server/database"
	"proxy-server/models"
	"proxy-server/utils"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthMiddleware struct {
	db *database.Database
}

func NewAuthMiddleware(db *database.Database) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		claims, err := utils.ValidateToken(parts[1])
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		user, err := m.db.GetUserByID(claims.UserID)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "User not found")
			return
		}

		if !user.IsActive {
			respondWithError(w, http.StatusForbidden, "User account is inactive")
			return
		}

		if user.TwoFAEnabled && !claims.TwoFactorVerified {
			respondWithError(w, http.StatusForbidden, "Two-factor authentication required")
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(UserContextKey).(*models.User)
		if !ok || user == nil {
			respondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		if !user.IsAdmin {
			respondWithError(w, http.StatusForbidden, "Admin access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

func GetUserFromContext(r *http.Request) *models.User {
	user, _ := r.Context().Value(UserContextKey).(*models.User)
	return user
}
